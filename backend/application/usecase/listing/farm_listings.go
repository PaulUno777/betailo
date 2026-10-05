package listing

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// FarmListing is one row of the farmer's "Annonces": the listing in any
// status, its cycle, and how much of it is spoken for.
type FarmListing struct {
	Listing            *domain.Listing
	Cycle              *domain.Cycle
	ActiveReservations int
}

// ListFarmListings is the owner's view of every listing on their farm,
// drafts and unpublished ones included, newest first.
type ListFarmListings struct {
	Farms        ports.FarmRepository
	Cycles       ports.CycleStore
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
}

func (uc *ListFarmListings) Execute(ctx context.Context, farmID, ownerID bson.ObjectID) ([]FarmListing, error) {
	f, err := uc.Farms.FindByID(ctx, farmID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != ownerID {
		return nil, fmt.Errorf("listing: not the farm owner: %w", domain.ErrForbidden)
	}
	cycles, err := uc.Cycles.ListByFarm(ctx, farmID)
	if err != nil {
		return nil, err
	}

	var out []FarmListing
	var ids []bson.ObjectID
	for _, c := range cycles {
		l, err := uc.Listings.FindByCycleID(ctx, c.ID)
		if errors.Is(err, domain.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, FarmListing{Listing: l, Cycle: c})
		ids = append(ids, l.ID)
	}
	if len(ids) > 0 {
		reservations, err := uc.Reservations.ListByListingIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		active := map[bson.ObjectID]int{}
		for _, r := range reservations {
			if domain.GroupOf(r) == domain.GroupActive {
				active[r.ListingID]++
			}
		}
		for i := range out {
			out[i].ActiveReservations = active[out[i].Listing.ID]
		}
	}
	// ObjectIDs embed creation time: newest listing first.
	slices.SortFunc(out, func(a, b FarmListing) int { return b.Listing.ID.Timestamp().Compare(a.Listing.ID.Timestamp()) })
	if out == nil {
		out = []FarmListing{}
	}
	return out, nil
}
