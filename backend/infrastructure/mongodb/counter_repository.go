package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// CounterRepository implements ports.Sequencer against the "counters"
// collection: one document per key, bumped with an atomic upsert.
type CounterRepository struct {
	Coll *mongo.Collection
}

var _ ports.Sequencer = (*CounterRepository)(nil)

// Next implements ports.Sequencer.
func (r *CounterRepository) Next(ctx context.Context, key string) (int64, error) {
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
