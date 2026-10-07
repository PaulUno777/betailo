package ports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// ListingSearchCriteria is every filter/sort/page parameter the public
// marketplace's search bar and filter sidebar expose — resolved
// entirely server-side (see ListingSearchRepository's doc comment) so
// "what counts as verified" or how ties break in a sort is one
// implementation, never something the API and a client could disagree
// on. Every field's zero value means "no filter"/"default".
type ListingSearchCriteria struct {
	// Species, CycleTypes and Statuses match any of their values. An
	// empty Statuses means the cycles open for reservation (planned and
	// in progress) — except for an IDs lookup, which shows every status.
	Species    []string
	CycleTypes []string
	Statuses   []domain.CycleStatus
	// Country and Region match the farm's exactly (see domain.Farm).
	Country      string
	Region       string
	MinPrice     float64
	MaxPrice     float64
	VerifiedOnly bool
	// AvailableFrom/AvailableTo keep lots open at some point in the
	// window: started by its end, not ended before its start. Either
	// bound may be zero.
	AvailableFrom time.Time
	AvailableTo   time.Time
	// Query is matched word by word: every word must appear in the
	// title, the farm's name or town, its region, or the species.
	Query string
	// IDs, when non-empty, restricts results to these listings (still
	// only published, unflagged ones). Backs "my favorites".
	IDs []bson.ObjectID
	// Sort is one of the ListingSort* constants; anything else
	// (including "") falls back to ListingSortAvailability.
	Sort     string
	Page     int
	PageSize int
}

// Accepted ListingSearchCriteria.Sort values — the HTTP layer passes
// the "sort" query parameter through verbatim, so these strings are the
// public API contract.
const (
	ListingSortAvailability = "availability" // soonest cycle start first
	ListingSortRecent       = "recent"       // newest listing first
	ListingSortPriceAsc     = "price_asc"
	ListingSortPriceDesc    = "price_desc"
	ListingSortRating       = "rating" // best-reviewed farm first
)

// ListingSearchItem is the marketplace read model — a Listing enriched
// with exactly the joined fields ListingCard/the listing detail page
// need, computed once server-side rather than requiring the frontend to
// stitch together separate farm/cycle/review calls per card.
type ListingSearchItem struct {
	Listing *domain.Listing

	Species     string
	CycleType   string
	CycleStatus domain.CycleStatus
	StartsAt    time.Time
	EndsAt      *time.Time
	Capacity    int
	Available   int

	FarmID         bson.ObjectID
	FarmName       string
	FarmLocation   string
	FarmCountry    string
	FarmRegion     string
	FarmVerified   bool
	FarmTrustScore float64

	ReviewAverage float64
	ReviewCount   int

	// Favorited is set by the usecase layer for a signed-in viewer.
	Favorited bool
}

// ListingSearchResult carries the page plus the total match count, for
// pagination controls.
type ListingSearchResult struct {
	Items  []ListingSearchItem
	Total  int
	Facets SearchFacets
}

// SearchFacets are the counts behind each filter, so a buyer sees how
// many lots a choice leads to before making it. Each dimension is
// counted with every other active filter applied but not its own, so
// picking a species still shows how many lots the other species have.
type SearchFacets struct {
	Species    []FacetCount
	CycleTypes []FacetCount
	Statuses   []FacetCount
	Regions    []RegionFacet
	// PriceMin/PriceMax bound the unit prices the price filter can
	// select from (zero when nothing matches).
	PriceMin float64
	PriceMax float64
}

type FacetCount struct {
	Value string
	Count int
}

type RegionFacet struct {
	Country string
	Region  string
	Count   int
}

// ListingSearchRepository is the public marketplace's read path — kept
// separate from ListingStore (the CRUD port application/usecase/listing
// otherwise depends on) so every existing caller of ListingStore is
// unaffected: only application/usecase/listing.SearchListings and its
// one HTTP handler need this wider, joined query.
type ListingSearchRepository interface {
	Search(ctx context.Context, criteria ListingSearchCriteria) (ListingSearchResult, error)
	// PublishedPrices returns up to limit unit prices of published,
	// unflagged listings of a species, optionally only from farms whose
	// location matches region (the price hint's sample).
	PublishedPrices(ctx context.Context, species, region string, limit int) ([]float64, error)
}
