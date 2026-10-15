package listing

import (
	"context"
	"slices"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/catalog"
	"github.com/pauluno777/betailo/backend/domain"
)

// SearchListings backs the public marketplace's search bar and filter
// sidebar — every filter, the "verified only" toggle, and sort order are
// resolved by ListingSearchRepository server-side; this usecase is a
// thin pass-through on purpose; there is no business rule left for it
// to own that the repository doesn't already enforce.
type SearchListings struct {
	Search    ports.ListingSearchRepository
	Favorites ports.FavoriteRepository
	// Catalog turns a category filter into its sub-categories.
	Catalog ports.CatalogSource
}

// Execute runs the search. A non-zero viewerID (a signed-in caller)
// also flags which results that user has saved.
func (uc *SearchListings) Execute(ctx context.Context, criteria ports.ListingSearchCriteria, viewerID bson.ObjectID) (ports.ListingSearchResult, error) {
	if len(criteria.Categories) > 0 || strings.TrimSpace(criteria.Query) != "" {
		cat, err := catalog.Of(ctx, uc.Catalog)
		if err != nil {
			return ports.ListingSearchResult{}, err
		}
		if len(criteria.Categories) > 0 {
			criteria.Species = expandCategories(cat, criteria.Categories, criteria.Species)
			criteria.Categories = nil
		}
		// Farms store region and town keys: each word also finds the
		// places whose name, in either language, contains it.
		criteria.QueryPlaces = nil
		for _, word := range strings.Fields(criteria.Query) {
			criteria.QueryPlaces = append(criteria.QueryPlaces, cat.PlaceKeys(word))
		}
	}
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

// expandCategories is the species a category filter stands for: every
// sub-category of each category, except where the explicit species
// already pick some of that category ("Volaille" + "Pondeuses" is just
// the laying hens). An unknown category matches nothing.
func expandCategories(cat *domain.Catalog, categories, species []string) []string {
	out := append([]string{}, species...)
	matched := false
	for _, key := range categories {
		subs := cat.SpeciesOf(key)
		narrowed := false
		for _, s := range subs {
			if slices.Contains(species, s) {
				narrowed = true
			}
		}
		if len(subs) > 0 {
			matched = true
		}
		if !narrowed {
			out = append(out, subs...)
		}
	}
	if !matched && len(out) == 0 {
		return []string{"-"} // a category nobody knows: no results, not everything
	}
	return out
}
