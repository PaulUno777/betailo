package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// ReviewSummary is the aggregated view RatingSummary (the design
// component) renders: average, total count, and a 1-5 star histogram.
type ReviewSummary struct {
	Average   float64
	Count     int
	Histogram map[int]int // rating (1-5) -> count
	// Criteria averages each detailed score over the reviews that gave
	// it (0 when none did).
	Criteria CriteriaAverages
}

// CriteriaAverages mirrors domain.ReviewCriteria as averages.
type CriteriaAverages struct {
	Health, Conformity, Communication, Timeliness float64
}

// ReviewFilter narrows a farm's review list, matching the design's chips.
type ReviewFilter string

const (
	ReviewFilterAll      ReviewFilter = ""
	ReviewFilterPhotos   ReviewFilter = "photos"
	ReviewFilterFive     ReviewFilter = "5"
	ReviewFilterCritical ReviewFilter = "critical" // 1-3 stars
)

// ReviewStore is the review page's side of reviews: paging and filters,
// single-review lookups, the farmer's reply and helpful votes.
type ReviewStore interface {
	// ListPage returns one page of a farm's reviews, newest first, plus
	// the total matching the filter.
	ListPage(ctx context.Context, farmID bson.ObjectID, filter ReviewFilter, page, pageSize int) ([]*domain.Review, int, error)
	// FindByID returns domain.ErrNotFound for an unknown review.
	FindByID(ctx context.Context, id bson.ObjectID) (*domain.Review, error)
	// SetReply records the farmer's reply only if there isn't one yet:
	// domain.ErrConflict otherwise (a single conditional update).
	SetReply(ctx context.Context, id bson.ObjectID, reply domain.ReviewReply) error
	// AddHelpfulVote counts userID's vote once: voting again is a no-op
	// reporting added=false.
	AddHelpfulVote(ctx context.Context, id, userID bson.ObjectID) (added bool, err error)
	// VotedBy reports which of ids userID has marked helpful.
	VotedBy(ctx context.Context, userID bson.ObjectID, ids []bson.ObjectID) (map[bson.ObjectID]bool, error)
}

// ReviewRepository persists reviews and serves the aggregated summary a
// farm's public page needs. The unique index on reservationId (see
// infrastructure/mongodb.Bootstrap) is what actually enforces "one
// review per reservation" — Insert here returns domain.ErrConflict for
// that race rather than the usecase needing to check-then-insert.
type ReviewRepository interface {
	Insert(ctx context.Context, r *domain.Review) error
	ListByFarm(ctx context.Context, farmID bson.ObjectID) ([]*domain.Review, error)
	Summary(ctx context.Context, farmID bson.ObjectID) (ReviewSummary, error)
}
