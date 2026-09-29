package listing

import (
	"context"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// SearchListings backs the public marketplace's search bar and filter
// sidebar — every filter, the "verified only" toggle, and sort order are
// resolved by ListingSearchRepository server-side; this usecase is a
// thin pass-through on purpose; there is no business rule left for it
// to own that the repository doesn't already enforce.
type SearchListings struct {
	Search ports.ListingSearchRepository
}

func (uc *SearchListings) Execute(ctx context.Context, criteria ports.ListingSearchCriteria) (ports.ListingSearchResult, error) {
	return uc.Search.Search(ctx, criteria)
}
