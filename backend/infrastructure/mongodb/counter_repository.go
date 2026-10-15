package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// CounterRepository implements ports.Sequencer against the "counters"
// collection: one document per prefix and month ("RSV-2610"), bumped
// with an atomic upsert.
type CounterRepository struct {
	Coll *mongo.Collection
	// Now is the clock that picks the month; nil means time.Now.
	Now func() time.Time
}

var _ ports.Sequencer = (*CounterRepository)(nil)

// NextReference implements ports.Sequencer.
func (r *CounterRepository) NextReference(ctx context.Context, prefix string) (string, error) {
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	period := domain.ReferencePeriod(prefix, now())
	n, err := r.next(ctx, period)
	if err != nil {
		return "", err
	}
	return domain.FormatReference(period, n), nil
}

func (r *CounterRepository) next(ctx context.Context, key string) (int64, error) {
	var doc struct {
		Seq int64 `bson:"seq"`
	}
	err := r.Coll.FindOneAndUpdate(ctx,
		bson.M{"_id": key},
		bson.M{"$inc": bson.M{"seq": int64(1)}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		return 0, fmt.Errorf("counter %s: %w", key, err)
	}
	return doc.Seq, nil
}
