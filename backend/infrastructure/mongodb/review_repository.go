package mongodb

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// reviewDoc mirrors domain.Review for BSON persistence.
type reviewDoc struct {
	ID            bson.ObjectID `bson:"_id,omitempty"`
	ReservationID bson.ObjectID `bson:"reservationId"`
	AuthorID      bson.ObjectID `bson:"authorId"`
	FarmID        bson.ObjectID `bson:"farmId"`
	Rating        int           `bson:"rating"`
	Comment       string        `bson:"comment,omitempty"`
}

func fromDomainReview(r *domain.Review) reviewDoc {
	return reviewDoc{
		ID: r.ID, ReservationID: r.ReservationID, AuthorID: r.AuthorID,
		FarmID: r.FarmID, Rating: r.Rating, Comment: r.Comment,
	}
}

func (d reviewDoc) toDomain() *domain.Review {
	return &domain.Review{
		ID: d.ID, ReservationID: d.ReservationID, AuthorID: d.AuthorID,
		FarmID: d.FarmID, Rating: d.Rating, Comment: d.Comment,
	}
}

// ReviewRepository implements ports.ReviewRepository against the
// "reviews" collection.
type ReviewRepository struct {
	Coll *mongo.Collection
}

var _ ports.ReviewRepository = (*ReviewRepository)(nil)

// Insert implements ports.ReviewRepository. The unique index on
// reservationId (see infrastructure/mongodb.Bootstrap) is what actually
// enforces "one review per reservation" — a race surfaces here as
// domain.ErrConflict rather than the usecase needing a separate
// check-then-insert.
func (r *ReviewRepository) Insert(ctx context.Context, rv *domain.Review) error {
	if rv.ID.IsZero() {
		rv.ID = bson.NewObjectID()
	}
	_, err := r.Coll.InsertOne(ctx, fromDomainReview(rv))
	if err == nil {
		return nil
	}
	var writeErr mongo.WriteException
	if errors.As(err, &writeErr) && writeErr.HasErrorCode(duplicateKeyCode) {
		return domain.ErrConflict
	}
	return err
}

// ListByFarm implements ports.ReviewRepository.
func (r *ReviewRepository) ListByFarm(ctx context.Context, farmID bson.ObjectID) ([]*domain.Review, error) {
	cur, err := r.Coll.Find(ctx, bson.M{"farmId": farmID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*domain.Review
	for cur.Next(ctx) {
		var doc reviewDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.toDomain())
	}
	return out, cur.Err()
}

// Summary implements ports.ReviewRepository via one aggregation: a
// single $group computes the average, count and 1-5 histogram in one
// round trip rather than pulling every review document into Go to
// average client-side.
func (r *ReviewRepository) Summary(ctx context.Context, farmID bson.ObjectID) (ports.ReviewSummary, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"farmId": farmID}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$rating",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cur, err := r.Coll.Aggregate(ctx, pipeline)
	if err != nil {
		return ports.ReviewSummary{}, err
	}
	defer cur.Close(ctx)

	summary := ports.ReviewSummary{Histogram: map[int]int{}}
	var weightedSum, total float64
	for cur.Next(ctx) {
		var row struct {
			Rating int `bson:"_id"`
			Count  int `bson:"count"`
		}
		if err := cur.Decode(&row); err != nil {
			return ports.ReviewSummary{}, err
		}
		summary.Histogram[row.Rating] = row.Count
		summary.Count += row.Count
		weightedSum += float64(row.Rating) * float64(row.Count)
		total += float64(row.Count)
	}
	if err := cur.Err(); err != nil {
		return ports.ReviewSummary{}, err
	}
	if total > 0 {
		summary.Average = weightedSum / total
	}
	return summary, nil
}
