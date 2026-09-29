package admin

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ListEscrowTransactions backs the admin console's "Séquestre" table —
// every reservation platform-wide, for the transactions list rather
// than a farm-scoped one (see
// application/usecase/reservation.ListReservationsByFarm for that).
type ListEscrowTransactions struct {
	Cycles       ports.CycleStore
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
}

func (uc *ListEscrowTransactions) Execute(ctx context.Context) ([]*domain.Reservation, error) {
	listingIDs, err := allListingIDs(ctx, uc.Cycles, uc.Listings)
	if err != nil {
		return nil, err
	}
	return uc.Reservations.ListByListingIDs(ctx, listingIDs)
}

// allListingIDs enumerates every listing platform-wide by walking every
// cycle in every status — there is no standalone "list all listings"
// port method (see ports.ListingStore's doc comment on why it stays
// narrow), so admin oversight composes this from the CRUD methods that
// already exist rather than widening that port for every other caller.
func allListingIDs(ctx context.Context, cycles ports.CycleStore, listings ports.ListingStore) ([]bson.ObjectID, error) {
	var ids []bson.ObjectID
	for _, status := range []domain.CycleStatus{
		domain.CycleStatusPlanned, domain.CycleStatusInProgress, domain.CycleStatusDone, domain.CycleStatusCancelled,
	} {
		cs, err := cycles.ListByStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		for _, c := range cs {
			l, err := listings.FindByCycleID(ctx, c.ID)
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			if err != nil {
				return nil, err
			}
			ids = append(ids, l.ID)
		}
	}
	return ids, nil
}
