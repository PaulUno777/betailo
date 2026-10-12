// Package farmstats holds the derived per-farm numbers more than one
// usecase needs (the trust score, a farm's public profile), so each
// number has exactly one definition.
package farmstats

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Sources is the read access ReleasedSales needs.
type Sources struct {
	Cycles       ports.CycleStore
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
}

// ReleasedSales counts reservations whose escrow was released, across
// every listing published from every cycle of the given farms. A "sale"
// is a completed, paid transaction, not merely a confirmed reservation.
func ReleasedSales(ctx context.Context, src Sources, farmIDs ...bson.ObjectID) (int, error) {
	var listingIDs []bson.ObjectID
	for _, farmID := range farmIDs {
		cycles, err := src.Cycles.ListByFarm(ctx, farmID)
		if err != nil {
			return 0, err
		}
		for _, c := range cycles {
			l, err := src.Listings.FindByCycleID(ctx, c.ID)
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			if err != nil {
				return 0, err
			}
			listingIDs = append(listingIDs, l.ID)
		}
	}
	if len(listingIDs) == 0 {
		return 0, nil
	}

	reservations, err := src.Reservations.ListByListingIDs(ctx, listingIDs)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, r := range reservations {
		if r.Payment != nil && r.Payment.Status == domain.PaymentReleased {
			count++
		}
	}
	return count, nil
}
