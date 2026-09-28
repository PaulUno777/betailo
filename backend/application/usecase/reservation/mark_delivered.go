package reservation

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// MarkDelivered is the farmer-triggered step: flags a paid reservation
// as delivered, starting the confirmation window (buyer confirms
// explicitly via ConfirmDelivery, or internal/worker auto-releases
// escrow 48h later if they don't and no dispute is open).
type MarkDelivered struct {
	Reservations ports.ReservationRepository
	Listings     ports.ListingStore
	Cycles       ports.CycleStore
	Farms        ports.FarmRepository
	Clock        ports.Clock
}

type MarkDeliveredInput struct {
	ReservationID bson.ObjectID
	FarmOwnerID   bson.ObjectID
}

// Execute returns domain.ErrForbidden if the caller doesn't own the
// farm behind the reservation's listing, and domain.ErrConflict if the
// reservation isn't a paid, held reservation yet (still pending/unpaid,
// or already delivered/cancelled).
func (uc *MarkDelivered) Execute(ctx context.Context, in MarkDeliveredInput) (*domain.Reservation, error) {
	res, err := uc.Reservations.FindByID(ctx, in.ReservationID)
	if err != nil {
		return nil, err
	}

	ownerID, err := farmOwner(ctx, uc.Listings, uc.Cycles, uc.Farms, res.ListingID)
	if err != nil {
		return nil, err
	}
	if ownerID != in.FarmOwnerID {
		return nil, fmt.Errorf("reservation: not the farm owner: %w", domain.ErrForbidden)
	}
	if res.Status != domain.ReservationConfirmed {
		return nil, fmt.Errorf("reservation: only a paid, held reservation can be marked delivered: %w", domain.ErrConflict)
	}

	now := uc.Clock.Now()
	res.Status = domain.ReservationDelivered
	res.DeliveredAt = &now
	if err := uc.Reservations.Update(ctx, res); err != nil {
		return nil, err
	}
	return res, nil
}
