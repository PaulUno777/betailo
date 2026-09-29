package reservation

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ListMyReservations backs the buyer's "Mes réservations" page.
type ListMyReservations struct {
	Reservations ports.ReservationRepository
}

func (uc *ListMyReservations) Execute(ctx context.Context, buyerID bson.ObjectID) ([]*domain.Reservation, error) {
	return uc.Reservations.ListByBuyer(ctx, buyerID)
}

// ListReservationsByFarm backs a farmer's "Réservations" portal page:
// every reservation against any listing published from any of their
// cycles.
type ListReservationsByFarm struct {
	Farms        ports.FarmRepository
	Cycles       ports.CycleStore
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
}

type ListReservationsByFarmInput struct {
	FarmID  bson.ObjectID
	OwnerID bson.ObjectID
}

// Execute returns domain.ErrForbidden if the caller doesn't own the
// farm.
func (uc *ListReservationsByFarm) Execute(ctx context.Context, in ListReservationsByFarmInput) ([]*domain.Reservation, error) {
	f, err := uc.Farms.FindByID(ctx, in.FarmID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != in.OwnerID {
		return nil, fmt.Errorf("reservation: not the farm owner: %w", domain.ErrForbidden)
	}

	cycles, err := uc.Cycles.ListByFarm(ctx, in.FarmID)
	if err != nil {
		return nil, err
	}

	listingIDs := make([]bson.ObjectID, 0, len(cycles))
	for _, c := range cycles {
		l, err := uc.Listings.FindByCycleID(ctx, c.ID)
		if errors.Is(err, domain.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		listingIDs = append(listingIDs, l.ID)
	}

	return uc.Reservations.ListByListingIDs(ctx, listingIDs)
}
