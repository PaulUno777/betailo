package admin

import (
	"context"
	"errors"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// EscrowItem is a reservation as the "Séquestre" table shows it: with
// the two people it binds.
type EscrowItem struct {
	Reservation *domain.Reservation
	BuyerName   string
	FarmName    string
}

// ListEscrowTransactions backs the back office's "Séquestre" table —
// every reservation platform-wide, latest first, rather than a
// farm-scoped list (see
// application/usecase/reservation.ListReservationsByFarm for that).
type ListEscrowTransactions struct {
	Cycles       ports.CycleStore
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
	// Farms and Users name the farm and the buyer; without them the
	// rows come back unnamed.
	Farms ports.FarmRepository
	Users ports.UserRepository
}

func (uc *ListEscrowTransactions) Execute(ctx context.Context) ([]EscrowItem, error) {
	farmOfListing := map[bson.ObjectID]bson.ObjectID{}
	var listingIDs []bson.ObjectID
	for _, status := range allCycleStatuses {
		cs, err := uc.Cycles.ListByStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		for _, c := range cs {
			l, err := uc.Listings.FindByCycleID(ctx, c.ID)
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			if err != nil {
				return nil, err
			}
			listingIDs = append(listingIDs, l.ID)
			farmOfListing[l.ID] = c.FarmID
		}
	}
	reservations, err := uc.Reservations.ListByListingIDs(ctx, listingIDs)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(reservations, func(i, j int) bool { return reservations[i].CreatedAt.After(reservations[j].CreatedAt) })

	farmNames, buyerNames := map[bson.ObjectID]string{}, map[bson.ObjectID]string{}
	out := make([]EscrowItem, 0, len(reservations))
	for _, r := range reservations {
		item := EscrowItem{Reservation: r}
		if uc.Farms != nil {
			farmID := farmOfListing[r.ListingID]
			name, seen := farmNames[farmID]
			if !seen {
				if f, err := uc.Farms.FindByID(ctx, farmID); err == nil {
					name = f.Name
				}
				farmNames[farmID] = name
			}
			item.FarmName = name
		}
		if uc.Users != nil {
			name, seen := buyerNames[r.BuyerID]
			if !seen {
				if u, err := uc.Users.FindByID(ctx, r.BuyerID); err == nil {
					name = displayName(u)
				}
				buyerNames[r.BuyerID] = name
			}
			item.BuyerName = name
		}
		out = append(out, item)
	}
	return out, nil
}

var allCycleStatuses = []domain.CycleStatus{
	domain.CycleStatusPlanned, domain.CycleStatusInProgress, domain.CycleStatusDone, domain.CycleStatusCancelled,
}

// allListingIDs enumerates every listing platform-wide by walking every
// cycle in every status — there is no standalone "list all listings"
// port method (see ports.ListingStore's doc comment on why it stays
// narrow), so admin oversight composes this from the CRUD methods that
// already exist rather than widening that port for every other caller.
func allListingIDs(ctx context.Context, cycles ports.CycleStore, listings ports.ListingStore) ([]bson.ObjectID, error) {
	var ids []bson.ObjectID
	for _, status := range allCycleStatuses {
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
