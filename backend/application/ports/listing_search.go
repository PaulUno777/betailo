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
	Species      string
	CycleType    string
	Region       string // matched against the farm's location
	MinPrice     float64
	MaxPrice     float64
	VerifiedOnly bool
	Query        string // matched against the listing's title
	// Sort is "availability" (default — soonest cycle start first),
	// "price_asc", or "price_desc".
	Sort     string
	Page     int
	PageSize int
}

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
	FarmVerified   bool
	FarmTrustScore float64

	ReviewAverage float64
	ReviewCount   int
}

// ListingSearchResult carries the page plus the total match count, for
// pagination controls.
type ListingSearchResult struct {
	Items []ListingSearchItem
	Total int
}

// ListingSearchRepository is the public marketplace's read path — kept
// separate from ListingStore (the CRUD port application/usecase/listing
// otherwise depends on) so every existing caller of ListingStore is
// unaffected: only application/usecase/listing.SearchListings and its
// one HTTP handler need this wider, joined query.
type ListingSearchRepository interface {
	Search(ctx context.Context, criteria ListingSearchCriteria) (ListingSearchResult, error)
}
