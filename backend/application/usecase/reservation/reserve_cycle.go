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
	Listings     ports.ListingRepository
	Cycles       ports.CycleRepository
	Reservations ports.ReservationRepository
	Payments     ports.PaymentProvider
	Tx           ports.TxManager
	Clock        ports.Clock
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

	cycleID, unitPrice, err := uc.Listings.CycleAndPrice(ctx, in.ListingID)
	if err != nil {
		return nil, err
	}

	cyc, err := uc.Cycles.FindByID(ctx, cycleID)
	if err != nil {
		return nil, err
	}
	if cyc.MinPerReservation > 0 && in.Quantity < cyc.MinPerReservation {
		return nil, fmt.Errorf("reservation: quantity below the cycle's minimum of %d: %w", cyc.MinPerReservation, domain.ErrInvalid)
	}
	if cyc.MaxPerReservation > 0 && in.Quantity > cyc.MaxPerReservation {
		return nil, fmt.Errorf("reservation: quantity above the cycle's maximum of %d: %w", cyc.MaxPerReservation, domain.ErrInvalid)
	}

	subtotal := unitPrice * float64(in.Quantity)
	fee := subtotal * ServiceFeeRate
	now := uc.Clock.Now()
	deadline := now.Add(UnpaidReservationGrace)

	res := &domain.Reservation{
		ID:              bson.NewObjectID(),
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

	if !in.PayNow {
		return &ReserveCycleOutput{Reservation: res}, nil
	}

	if err := capturePayment(ctx, uc.Payments, uc.Reservations, uc.Clock, res, in.PaymentMethod); err != nil {
		return &ReserveCycleOutput{Reservation: res, PaymentCaptured: false}, nil
	}
	return &ReserveCycleOutput{Reservation: res, PaymentCaptured: true}, nil
}

// capturePayment centralizes the pay-now transition shared by
// ReserveCycle (immediate) and PayReservation (a pending reservation
// paid later): take payment, then persist the reservation as
// confirmed, held in escrow, and prioritized.
func capturePayment(ctx context.Context, payments ports.PaymentProvider, reservations ports.ReservationRepository, clock ports.Clock, res *domain.Reservation, method ports.PaymentMethod) error {
	if _, err := payments.Capture(ctx, res.ID.Hex(), res.TotalAmount, method); err != nil {
		return err
	}
	now := clock.Now()
	res.Status = domain.ReservationConfirmed
	res.Priority = true
	res.PaymentDeadline = nil
	res.Payment.Status = domain.PaymentHeld
	res.Payment.HeldAt = &now
	return reservations.Update(ctx, res)
}
