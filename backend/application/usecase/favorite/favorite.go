// Package favorite implements a buyer's saved listings.
package favorite

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// maxFavorites bounds "my favorites" to one page. Nobody needs more
// saved listings than this, and it keeps the list a single query.
const maxFavorites = 100

// Add saves a listing for a user. Only a published, unflagged listing
// can be saved (anything else reads as domain.ErrNotFound, the same as
// it does on the public marketplace).
type Add struct {
	Listings  ports.ListingStore
	Favorites ports.FavoriteRepository
	Clock     ports.Clock
}

func (uc *Add) Execute(ctx context.Context, userID, listingID bson.ObjectID) error {
	l, err := uc.Listings.FindByID(ctx, listingID)
	if err != nil {
		return err
	}
	if l.Status != domain.ListingPublished || l.Flagged {
		return fmt.Errorf("favorite: listing is not available: %w", domain.ErrNotFound)
	}
	return uc.Favorites.Add(ctx, userID, listingID, uc.Clock.Now())
}

// Remove un-saves a listing. Removing one that was never saved is fine.
type Remove struct {
	Favorites ports.FavoriteRepository
}

func (uc *Remove) Execute(ctx context.Context, userID, listingID bson.ObjectID) error {
	return uc.Favorites.Remove(ctx, userID, listingID)
}

// ListMine returns a user's saved listings as marketplace items, most
// recently saved first. A saved listing that has since been unpublished
// or flagged simply drops out, as it does from search.
type ListMine struct {
	Favorites ports.FavoriteRepository
	Search    ports.ListingSearchRepository
}

func (uc *ListMine) Execute(ctx context.Context, userID bson.ObjectID) ([]ports.ListingSearchItem, error) {
	ids, err := uc.Favorites.ListListingIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []ports.ListingSearchItem{}, nil
	}
	if len(ids) > maxFavorites {
		ids = ids[:maxFavorites]
	}

	result, err := uc.Search.Search(ctx, ports.ListingSearchCriteria{IDs: ids, PageSize: maxFavorites})
	if err != nil {
		return nil, err
	}
	byID := make(map[bson.ObjectID]ports.ListingSearchItem, len(result.Items))
	for _, it := range result.Items {
		it.Favorited = true
		byID[it.Listing.ID] = it
	}
	out := make([]ports.ListingSearchItem, 0, len(byID))
	for _, id := range ids {
		if it, ok := byID[id]; ok {
			out = append(out, it)
		}
	}
	return out, nil
}
