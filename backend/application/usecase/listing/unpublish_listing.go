package listing

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// UnpublishListing is this module's "delete": it sets the listing's
// status to domain.ListingUnpublished rather than removing the
// document. A hard delete would orphan any reservation that references
// this listing's ID — see ADR-0006's reasoning for the analogous choice
// on Register.Execute, and ADR-0001 on embedding vs. referencing.
type UnpublishListing struct {
	Farms    ports.FarmRepository
	Cycles   ports.CycleStore
	Listings ports.ListingStore
}

// Execute returns domain.ErrForbidden if ownerID doesn't own the
// listing's cycle's farm.
func (uc *UnpublishListing) Execute(ctx context.Context, listingID, ownerID bson.ObjectID) error {
	l, err := uc.Listings.FindByID(ctx, listingID)
	if err != nil {
		return err
	}

	actualOwner, err := ownerOf(ctx, uc.Cycles, uc.Farms, l)
	if err != nil {
		return err
	}
	if actualOwner != ownerID {
		return fmt.Errorf("listing: not the owner: %w", domain.ErrForbidden)
	}

	l.Status = domain.ListingUnpublished
	return uc.Listings.Update(ctx, l)
}
