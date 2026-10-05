package reservation

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// CancelReservation lets a buyer withdraw a reservation they haven't
// paid yet, giving the places straight back to the cycle. A paid one is
// held in escrow and goes through a dispute instead, so it can't be
// cancelled here.
type CancelReservation struct {
	Reservations ports.ReservationRepository
	Listings     ports.ListingRepository
	Cycles       ports.CycleRepository
	Tx           ports.TxManager
}

type CancelReservationInput struct {
	ReservationID bson.ObjectID
	BuyerID       bson.ObjectID
}

// Execute returns domain.ErrForbidden for someone else's reservation and
// domain.ErrConflict unless it's still pending and unpaid.
func (uc *CancelReservation) Execute(ctx context.Context, in CancelReservationInput) (*domain.Reservation, error) {
	res, err := uc.Reservations.FindByID(ctx, in.ReservationID)
	if err != nil {
		return nil, err
	}
	if res.BuyerID != in.BuyerID {
		return nil, fmt.Errorf("reservation: not the buyer: %w", domain.ErrForbidden)
	}
	if res.Status != domain.ReservationPending || (res.Payment != nil && res.Payment.Status != domain.PaymentPending) {
		return nil, fmt.Errorf("reservation: only an unpaid reservation can be cancelled: %w", domain.ErrConflict)
	}

	if err := releasePlaces(ctx, uc.Tx, uc.Listings, uc.Cycles, uc.Reservations, res, domain.PaymentFailed); err != nil {
		return nil, err
	}
	return res, nil
}

// releasePlaces cancels a pending reservation and returns its quantity
// to the cycle in one transaction. Without the transaction, a failed
// update after the stock was restored would leave the reservation
// pending, and the next attempt would restore the same places twice.
func releasePlaces(ctx context.Context, tx ports.TxManager, listings ports.ListingRepository, cycles ports.CycleRepository, reservations ports.ReservationRepository, res *domain.Reservation, paymentStatus domain.PaymentStatus) error {
	cycleID, _, err := listings.CycleAndPrice(ctx, res.ListingID)
	if err != nil {
		return err
	}
	return tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := cycles.IncrementAvailability(ctx, cycleID, res.Quantity); err != nil {
			return err
		}
		res.Status = domain.ReservationCancelled
		if res.Payment != nil {
			res.Payment.Status = paymentStatus
		}
		return reservations.Update(ctx, res)
	})
}
