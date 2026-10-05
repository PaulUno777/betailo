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
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/notify"
	"github.com/pauluno777/betailo/backend/domain"
)

// ServiceFeeRate is Betailo's cut on every reservation — computed here,
// once, so the frontend never hardcodes (or could drift from) the
// percentage; see the design's "Frais de service" line item.
const ServiceFeeRate = 0.02

// UnpaidReservationGrace is how long a "réserver sans payer" reservation
// holds its quantity before internal/worker expires it and restores
// capacity to the cycle.
const UnpaidReservationGrace = 24 * time.Hour

// ReserveCycle is the usecase: read the listing and cycle, validate the
// quantity against the cycle's bounds, then run the availability
// check-and-decrement and the reservation insert inside one TxManager so
// they succeed or fail together. If the buyer chose to pay now, capture
// happens only after that transaction commits — never before, so a
// buyer is never charged for a reservation that failed on capacity.
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

	// PayNow requests immediate capture ("payer maintenant" — priority,
	// escrow held right away). false reserves the quantity without
	// paying ("réserver sans payer"): non-priority, subject to
	// UnpaidReservationGrace and the worker's expiry sweep.
	PayNow        bool
	PaymentMethod ports.PaymentMethod
	// PayerPhone is the Mobile Money number as typed; required for
	// mtn_momo and orange_money, ignored for card.
	PayerPhone string
}

// ReserveCycleOutput wraps the usecase's result. PaymentCaptured is
// false when PayNow was requested but the capture itself failed — the
// reservation still exists (unpaid, with a deadline); the caller decides
// whether to surface that as a retryable "pay again" state via
// PayReservation.
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
	deadline := now.Add(UnpaidReservationGrace)

	res := &domain.Reservation{
		ID:              bson.NewObjectID(),
		Reference:       domain.FormatReference(domain.ReservationRefPrefix, seq),
		ListingID:       in.ListingID,
		BuyerID:         in.BuyerID,
		Quantity:        in.Quantity,
		UnitPrice:       unitPrice,
		ServiceFee:      fee,
		TotalAmount:     subtotal + fee,
		Status:          domain.ReservationPending,
		CreatedAt:       now,
		PaymentDeadline: &deadline,
		Payment:         &domain.PaymentInfo{Status: domain.PaymentPending, Amount: subtotal + fee},
	}

	err = uc.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.Cycles.DecrementAvailability(ctx, cycleID, in.Quantity); err != nil {
			return err
		}
		return uc.Reservations.Insert(ctx, res)
	})
	if err != nil {
		return nil, err
	}

	captured := false
	if in.PayNow {
		captured = capturePayment(ctx, uc.Payments, uc.Reservations, uc.Clock, res, payment) == nil
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
	res.PaymentDeadline = nil
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
			return validPayment{}, fmt.Errorf("payment: mobile money needs a Cameroonian mobile number: %w", domain.ErrInvalid)
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
		return nil, fmt.Errorf("reservation: the cycle is no longer open: %w", domain.ErrConflict)
	}
	farm, err := farms.FindByID(ctx, cyc.FarmID)
	if err != nil {
		return nil, err
	}
	// A farmer reserving their own animals would mint released "sales"
	// that feed their trust score.
	if farm.OwnerID == buyerID {
		return nil, fmt.Errorf("reservation: you can't reserve your own listing: %w", domain.ErrForbidden)
	}
	if cyc.MinPerReservation > 0 && quantity < cyc.MinPerReservation {
		return nil, fmt.Errorf("reservation: quantity below the cycle's minimum of %d: %w", cyc.MinPerReservation, domain.ErrInvalid)
	}
	if cyc.MaxPerReservation > 0 && quantity > cyc.MaxPerReservation {
		return nil, fmt.Errorf("reservation: quantity above the cycle's maximum of %d: %w", cyc.MaxPerReservation, domain.ErrInvalid)
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
