package cycle

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// DeleteCycle is the usecase: remove a cycle the caller's farm owns,
// provided it has no listing — deleting a cycle out from under an
// existing listing would orphan that listing's cycleId reference (and
// the reservations that reference the listing).
type DeleteCycle struct {
	Farms    ports.FarmRepository
	Cycles   ports.CycleStore
	Listings ports.ListingStore
}

// Execute returns domain.ErrNotFound if the cycle or its farm doesn't
// exist, domain.ErrForbidden if ownerID isn't the farm's owner, and
// domain.ErrConflict if the cycle still has a listing.
func (uc *DeleteCycle) Execute(ctx context.Context, cycleID, ownerID bson.ObjectID) error {
	c, err := uc.Cycles.FindByID(ctx, cycleID)
	if err != nil {
		return err
	}

	f, err := uc.Farms.FindByID(ctx, c.FarmID)
	if err != nil {
		return err
	}
	if f.OwnerID != ownerID {
		return fmt.Errorf("cycle: not the farm owner: %w", domain.ErrForbidden)
	}

	_, err = uc.Listings.FindByCycleID(ctx, cycleID)
	if err == nil {
		return domain.Coded("cycle.has_listing", fmt.Errorf("cycle: has an existing listing: %w", domain.ErrConflict))
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return err
	}

	return uc.Cycles.Delete(ctx, cycleID)
}
