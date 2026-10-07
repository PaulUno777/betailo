// Package reservation implements the reservation and escrow state
// machine — the response to scarcity (anti-overselling, via a MongoDB
// multi-document transaction) and the payment lifecycle each
// reservation carries (pending -> confirmed -> delivered -> released,
// with failed/refunded as alternate exits) — see the reservation and
// escrow sections of the "Concepts & Architecture" document.
//
// Every usecase here depends only on application/ports interfaces,
// never on *mongo.Client or a real payment SDK directly.
package reservation

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/notify"
	"github.com/pauluno777/betailo/backend/domain"
)

// ServiceFeeRate is Betailo's cut on every reservation — computed here,
// once, so the frontend never hardcodes (or could drift from) the
// percentage; see the design's "Frais de service" line item.
const ServiceFeeRate = 0.02

// ReserveCycle is the usecase. Only payment takes places:
//
//   - Pay now: the availability check-and-decrement and the insert run
//     in one transaction, then the payment is captured — never before,
//     so a buyer is never charged for a reservation that failed on
//     capacity. If the capture fails, the places go back and the
//     reservation stays on the waiting list.
//   - Reserve without paying: the reservation joins the cycle's waiting
//     list. It holds no places and never expires; its buyer can pay
//     while places remain, and paid buyers are always served first.
//     Unpaid reservations still open when the cycle closes are cancelled
//     (CloseUnpaidForCycle).
type ReserveCycle struct {
	Listings     ports.ListingStore
	Cycles       ports.CycleRepository
	Farms        ports.FarmRepository
	Sequencer    ports.Sequencer
	Reservations ports.ReservationRepository
	Payments     ports.PaymentProvider
	Tx           ports.TxManager
	Clock        ports.Clock
	// Notifications tells the farmer about the new reservation. Optional.
	Notifications ports.NotificationRepository
}

// ReserveCycleInput is what a caller (an HTTP handler) supplies.
type ReserveCycleInput struct {
	ListingID bson.ObjectID
	BuyerID   bson.ObjectID
	Quantity  int

	// PayNow requests immediate capture ("payer maintenant": the places
	// are taken, escrow held right away). false joins the waiting list
	// ("réserver sans payer"): no places held until the buyer pays.
	PayNow        bool
	PaymentMethod ports.PaymentMethod
	// PayerPhone is the Mobile Money number as typed; required for
	// mtn_momo and orange_money, ignored for card.
	PayerPhone string
}

// ReserveCycleOutput wraps the usecase's result. PaymentCaptured is
// false when PayNow was requested but the capture itself failed — the
// reservation still exists, on the waiting list; the caller surfaces
// that as a retryable "pay again" state via PayReservation.
type ReserveCycleOutput struct {
	Reservation     *domain.Reservation
	PaymentCaptured bool
}

func (uc *ReserveCycle) Execute(ctx context.Context, in ReserveCycleInput) (*ReserveCycleOutput, error) {
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("reservation: quantity must be positive: %w", domain.ErrInvalid)
	}

	var payment validPayment
	if in.PayNow {
		p, err := validatePayment(in.PaymentMethod, in.PayerPhone)
		if err != nil {
			return nil, err
		}
		payment = p
	}

	ok, err := checkReservable(ctx, uc.Listings, uc.Cycles, uc.Farms, in.ListingID, in.BuyerID, in.Quantity)
	if err != nil {
		return nil, err
	}
	l, cycleID, unitPrice, farm := ok.listing, ok.cycle.ID, ok.listing.UnitPrice, ok.farm

	seq, err := uc.Sequencer.Next(ctx, domain.ReservationRefPrefix)
	if err != nil {
		return nil, err
	}

	amounts := priceOf(unitPrice, in.Quantity)
	subtotal, fee := amounts.Subtotal, amounts.ServiceFee
	now := uc.Clock.Now()

	res := &domain.Reservation{
		ID:          bson.NewObjectID(),
		Reference:   domain.FormatReference(domain.ReservationRefPrefix, seq),
		ListingID:   in.ListingID,
		BuyerID:     in.BuyerID,
		Quantity:    in.Quantity,
		UnitPrice:   unitPrice,
		ServiceFee:  fee,
		TotalAmount: subtotal + fee,
		Status:      domain.ReservationPending,
		CreatedAt:   now,
		Payment:     &domain.PaymentInfo{Status: domain.PaymentPending, Amount: subtotal + fee},
	}

	captured := false
	if in.PayNow {
		err = uc.Tx.WithinTx(ctx, func(ctx context.Context) error {
			if err := uc.Cycles.DecrementAvailability(ctx, cycleID, in.Quantity); err != nil {
				return err
			}
			return uc.Reservations.Insert(ctx, res)
		})
		if err != nil {
			return nil, soldOut(err)
		}
		if capturePayment(ctx, uc.Payments, uc.Reservations, uc.Clock, res, payment) == nil {
			captured = true
		} else if err := uc.Cycles.IncrementAvailability(ctx, cycleID, in.Quantity); err != nil {
			// The reservation stays on the waiting list either way; a
			// failed give-back only under-sells, it never over-sells.
			return nil, err
		}
	} else {
		if in.Quantity > ok.cycle.Capacity {
			return nil, domain.CodedWith("reservation.above_capacity", map[string]any{"max": ok.cycle.Capacity},
				fmt.Errorf("reservation: more than the cycle's %d places: %w", ok.cycle.Capacity, domain.ErrInvalid))
		}
		if err := uc.Reservations.Insert(ctx, res); err != nil {
			return nil, err
		}
	}

	typ, fallback := domain.NotifReservationCreated, "Nouvelle réservation %s."
	if captured {
		typ, fallback = domain.NotifReservationPaid, "Réservation %s payée, fonds en séquestre."
	}
	notify.Send(ctx, uc.Notifications, now, notify.Notice{
		UserID: farm.OwnerID, Type: typ, Params: reservationParams(res, l.Title),
		Link: domain.LinkFarmerReservations, Fallback: fmt.Sprintf(fallback, res.Reference),
	})

	return &ReserveCycleOutput{Reservation: res, PaymentCaptured: captured}, nil
}

// soldOut gives a capacity refusal the code clients show as "complet".
func soldOut(err error) error {
	if errors.Is(err, domain.ErrInsufficientCapacity) {
		return domain.Coded("reservation.sold_out", err)
	}
	return err
}

// capturePayment centralizes the pay-now transition shared by
// ReserveCycle (immediate) and PayReservation (a pending reservation
// paid later): take payment, then persist the reservation as
// confirmed, held in escrow, and prioritized.
func capturePayment(ctx context.Context, payments ports.PaymentProvider, reservations ports.ReservationRepository, clock ports.Clock, res *domain.Reservation, p validPayment) error {
	providerRef, err := payments.Capture(ctx, ports.CaptureRequest{
		Reference: res.Reference, Amount: res.TotalAmount, Method: p.method, PayerPhone: p.phone,
	})
	if err != nil {
		return err
	}
	now := clock.Now()
	res.Status = domain.ReservationConfirmed
	res.Priority = true
	res.Payment.Status = domain.PaymentHeld
	res.Payment.HeldAt = &now
	res.Payment.Method = string(p.method)
	res.Payment.ProviderRef = providerRef
	if p.phone != "" {
		res.Payment.PayerPhoneMasked = domain.MaskPhone(p.phone)
	}
	return reservations.Update(ctx, res)
}

// validPayment is a checked payment method plus, for Mobile Money, the
// payer's number in E.164.
type validPayment struct {
	method ports.PaymentMethod
	phone  string
}

// validatePayment checks the method is one we take and that Mobile Money
// comes with a Cameroonian mobile number (the only kind MoMo and Orange
// Money work with). Card needs no number.
func validatePayment(method ports.PaymentMethod, rawPhone string) (validPayment, error) {
	switch method {
	case ports.PaymentMethodCard:
		return validPayment{method: method}, nil
	case ports.PaymentMethodMTNMoMo, ports.PaymentMethodOrangeMoney:
		phone, err := domain.NormalizePhone(rawPhone)
		if err != nil {
			return validPayment{}, fmt.Errorf("payment: mobile money number: %w", err)
		}
		if !domain.IsCameroonMobile(phone) {
			return validPayment{}, domain.Coded("payment.phone_invalid", fmt.Errorf("payment: mobile money needs a Cameroonian mobile number: %w", domain.ErrInvalid))
		}
		return validPayment{method: method, phone: phone}, nil
	default:
		return validPayment{}, fmt.Errorf("payment: unknown payment method %q: %w", method, domain.ErrInvalid)
	}
}

// reservable is what checkReservable resolved on the way.
type reservable struct {
	listing *domain.Listing
	cycle   *domain.Cycle
	farm    *domain.Farm
}

// checkReservable is every rule about whether buyerID may reserve
// quantity places on a listing, shared by ReserveCycle and Quote so the
// checkout preview can never disagree with the reservation itself.
// Remaining capacity is enforced atomically by the reservation's own
// decrement; Quote checks it too, as a preview.
func checkReservable(ctx context.Context, listings ports.ListingStore, cycles ports.CycleRepository, farms ports.FarmRepository, listingID, buyerID bson.ObjectID, quantity int) (*reservable, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("reservation: quantity must be positive: %w", domain.ErrInvalid)
	}
	l, err := listings.FindByID(ctx, listingID)
	if err != nil {
		return nil, err
	}
	// The same rule as the public listing page: only a published,
	// unflagged listing is on sale.
	if l.Status != domain.ListingPublished || l.Flagged || l.CycleID == nil {
		return nil, fmt.Errorf("reservation: listing is not available: %w", domain.ErrNotFound)
	}
	cyc, err := cycles.FindByID(ctx, *l.CycleID)
	if err != nil {
		return nil, err
	}
	if cyc.Status != domain.CycleStatusPlanned && cyc.Status != domain.CycleStatusInProgress {
		return nil, domain.Coded("reservation.cycle_closed", fmt.Errorf("reservation: the cycle is no longer open: %w", domain.ErrConflict))
	}
	farm, err := farms.FindByID(ctx, cyc.FarmID)
	if err != nil {
		return nil, err
	}
	// A farmer reserving their own animals would mint released "sales"
	// that feed their trust score.
	if farm.OwnerID == buyerID {
		return nil, domain.Coded("reservation.own_listing", fmt.Errorf("reservation: you can't reserve your own listing: %w", domain.ErrForbidden))
	}
	if cyc.MinPerReservation > 0 && quantity < cyc.MinPerReservation {
		return nil, domain.CodedWith("reservation.below_min", map[string]any{"min": cyc.MinPerReservation}, fmt.Errorf("reservation: quantity below the cycle's minimum of %d: %w", cyc.MinPerReservation, domain.ErrInvalid))
	}
	if cyc.MaxPerReservation > 0 && quantity > cyc.MaxPerReservation {
		return nil, domain.CodedWith("reservation.above_max", map[string]any{"max": cyc.MaxPerReservation}, fmt.Errorf("reservation: quantity above the cycle's maximum of %d: %w", cyc.MaxPerReservation, domain.ErrInvalid))
	}
	return &reservable{listing: l, cycle: cyc, farm: farm}, nil
}

// Amounts is a reservation's price breakdown.
type Amounts struct {
	UnitPrice  float64
	Subtotal   float64
	ServiceFee float64
	Total      float64
}

func priceOf(unitPrice float64, quantity int) Amounts {
	subtotal := unitPrice * float64(quantity)
	fee := subtotal * ServiceFeeRate
	return Amounts{UnitPrice: unitPrice, Subtotal: subtotal, ServiceFee: fee, Total: subtotal + fee}
}
