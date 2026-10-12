package reservation

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// PayReservation lets a buyer pay a reservation on the waiting list
// ("réserver sans payer"). Paying is what takes places: they are
// decremented first (refused as sold out when too few remain, since paid
// buyers come first), then the payment is captured through the same
// capturePayment path ReserveCycle uses; a failed capture gives the
// places back.
type PayReservation struct {
	Reservations  ports.ReservationRepository
	Listings      ports.ListingRepository
	Cycles        ports.CycleRepository
	Payments      ports.PaymentProvider
	Clock         ports.Clock
	Parties       Parties
	Notifications ports.NotificationRepository
}

type PayReservationInput struct {
	ReservationID bson.ObjectID
	BuyerID       bson.ObjectID
	PaymentMethod ports.PaymentMethod
	PayerPhone    string
}

// Execute returns domain.ErrNotFound if the reservation doesn't exist,
// domain.ErrForbidden if the caller isn't the reservation's buyer, and
// domain.ErrConflict if it's no longer pending (already paid, delivered
// or cancelled), and domain.ErrInsufficientCapacity (code
// reservation.sold_out) once paid buyers have taken the places.
func (uc *PayReservation) Execute(ctx context.Context, in PayReservationInput) (*domain.Reservation, error) {
	res, err := uc.Reservations.FindByID(ctx, in.ReservationID)
	if err != nil {
		return nil, err
	}
	if res.BuyerID != in.BuyerID {
		return nil, fmt.Errorf("reservation: not the buyer: %w", domain.ErrForbidden)
	}
	if res.Status != domain.ReservationPending {
		return nil, domain.Coded("reservation.not_awaiting_payment", fmt.Errorf("reservation: not awaiting payment: %w", domain.ErrConflict))
	}

	payment, err := validatePayment(in.PaymentMethod, in.PayerPhone)
	if err != nil {
		return nil, err
	}
	cycleID, _, err := uc.Listings.CycleAndPrice(ctx, res.ListingID)
	if err != nil {
		return nil, err
	}
	if err := uc.Cycles.DecrementAvailability(ctx, cycleID, res.Quantity); err != nil {
		return nil, soldOut(err)
	}
	if err := capturePayment(ctx, uc.Payments, uc.Reservations, uc.Clock, res, payment); err != nil {
		if restoreErr := uc.Cycles.IncrementAvailability(ctx, cycleID, res.Quantity); restoreErr != nil {
			return nil, errors.Join(err, restoreErr)
		}
		return nil, err
	}
	uc.Parties.notifyFarmer(ctx, uc.Notifications, uc.Clock, res, domain.NotifReservationPaid, "Réservation %s payée, fonds en séquestre.")
	return res, nil
}
