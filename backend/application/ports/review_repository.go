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
