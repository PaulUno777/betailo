package reservation

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// PayReservation lets a buyer pay an existing unpaid ("réserver sans
// payer") reservation before internal/worker expires it — moves it from
// pending/unpaid to confirmed/held/priority via the same capturePayment
// path ReserveCycle uses for pay-now.
type PayReservation struct {
	Reservations ports.ReservationRepository
	Payments     ports.PaymentProvider
	Clock        ports.Clock
}

type PayReservationInput struct {
	ReservationID bson.ObjectID
	BuyerID       bson.ObjectID
	PaymentMethod ports.PaymentMethod
}

// Execute returns domain.ErrNotFound if the reservation doesn't exist,
// domain.ErrForbidden if the caller isn't the reservation's buyer, and
// domain.ErrConflict if it's no longer pending (already paid, delivered
// or cancelled/expired).
func (uc *PayReservation) Execute(ctx context.Context, in PayReservationInput) (*domain.Reservation, error) {
	res, err := uc.Reservations.FindByID(ctx, in.ReservationID)
	if err != nil {
		return nil, err
	}
	if res.BuyerID != in.BuyerID {
		return nil, fmt.Errorf("reservation: not the buyer: %w", domain.ErrForbidden)
	}
	if res.Status != domain.ReservationPending {
		return nil, fmt.Errorf("reservation: not awaiting payment: %w", domain.ErrConflict)
	}

	if err := capturePayment(ctx, uc.Payments, uc.Reservations, uc.Clock, res, in.PaymentMethod); err != nil {
		return nil, err
	}
	return res, nil
}
