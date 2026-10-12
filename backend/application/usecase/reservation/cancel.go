package reservation

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// CancelReservation lets a buyer leave the waiting list: withdraw a
// reservation they haven't paid. It held no places, so the cycle is
// untouched. A paid one is held in escrow and goes through a dispute
// instead, so it can't be cancelled here.
type CancelReservation struct {
	Reservations ports.ReservationRepository
	Clock        ports.Clock

	Parties       Parties
	Notifications ports.NotificationRepository
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
		return nil, domain.Coded("reservation.cancel_paid", fmt.Errorf("reservation: only an unpaid reservation can be cancelled: %w", domain.ErrConflict))
	}

	// A reservation on the waiting list holds no places: there is nothing
	// to give back to the cycle.
	res.Status = domain.ReservationCancelled
	if res.Payment != nil {
		res.Payment.Status = domain.PaymentFailed
	}
	if err := uc.Reservations.Update(ctx, res); err != nil {
		return nil, err
	}
	uc.Parties.notifyFarmer(ctx, uc.Notifications, uc.Clock, res, domain.NotifReservationCancelled, "Réservation %s annulée par l'acheteur.")
	return res, nil
}
