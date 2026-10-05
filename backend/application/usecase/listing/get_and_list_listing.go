package listing

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// GetListing is the usecase: look up a listing by ID for a viewer
// (zero ID when anonymous), applying the visibility rule in visibleTo:
// a buyer can view any public listing before reserving it, and a
// farmer can also load their own drafts to edit them.
type GetListing struct {
	Listings ports.ListingStore
	Cycles   ports.CycleStore
	Farms    ports.FarmRepository
}

func (uc *GetListing) Execute(ctx context.Context, id, viewerID bson.ObjectID) (*domain.Listing, error) {
	l, err := uc.Listings.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if isPublic(l) {
		return l, nil
	}
	isOwner := false
	if !viewerID.IsZero() && l.CycleID != nil {
		c, err := uc.Cycles.FindByID(ctx, *l.CycleID)
		if err != nil {
			return nil, err
		}
		f, err := uc.Farms.FindByID(ctx, c.FarmID)
		if err != nil {
			return nil, err
		}
		isOwner = f.OwnerID == viewerID
	}
	if err := visibleTo(l, isOwner); err != nil {
		return nil, err
	}
	return l, nil
}
