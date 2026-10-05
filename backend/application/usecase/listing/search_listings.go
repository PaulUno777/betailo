package listing

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// SearchListings backs the public marketplace's search bar and filter
// sidebar — every filter, the "verified only" toggle, and sort order are
// resolved by ListingSearchRepository server-side; this usecase is a
// thin pass-through on purpose; there is no business rule left for it
// to own that the repository doesn't already enforce.
type SearchListings struct {
	Search    ports.ListingSearchRepository
	Favorites ports.FavoriteRepository
}

// Execute runs the search. A non-zero viewerID (a signed-in caller)
// also flags which results that user has saved.
func (uc *SearchListings) Execute(ctx context.Context, criteria ports.ListingSearchCriteria, viewerID bson.ObjectID) (ports.ListingSearchResult, error) {
	result, err := uc.Search.Search(ctx, criteria)
	if err != nil || viewerID.IsZero() || len(result.Items) == 0 {
		return result, err
	}
	ids := make([]bson.ObjectID, 0, len(result.Items))
	for _, it := range result.Items {
		ids = append(ids, it.Listing.ID)
	}
	saved, err := uc.Favorites.FavoritedAmong(ctx, viewerID, ids)
	if err != nil {
		return ports.ListingSearchResult{}, err
	}
	for i := range result.Items {
		result.Items[i].Favorited = saved[result.Items[i].Listing.ID]
	}
	return result, nil
}
