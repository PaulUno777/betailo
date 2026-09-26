package listing

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// GetListing is the usecase: look up a listing by ID. Needs no
// authentication — a buyer must be able to view a listing before
// reserving it.
type GetListing struct {
	Listings ports.ListingStore
}

func (uc *GetListing) Execute(ctx context.Context, id bson.ObjectID) (*domain.Listing, error) {
	return uc.Listings.FindByID(ctx, id)
}

// ListPublishedListings is the usecase: the marketplace-facing browse
// view. Needs no authentication.
type ListPublishedListings struct {
	Listings ports.ListingStore
}

func (uc *ListPublishedListings) Execute(ctx context.Context) ([]*domain.Listing, error) {
	return uc.Listings.ListPublished(ctx)
}
