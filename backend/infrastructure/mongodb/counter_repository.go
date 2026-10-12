package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
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

// BackfillReferences gives every reservation and cycle created before
// references existed one, oldest first, from the same counters new ones
// use. Idempotent: it only touches documents without a reference, so
// running it on every startup is a no-op once done.
func BackfillReferences(ctx context.Context, database *mongo.Database) error {
	counters := &CounterRepository{Coll: database.Collection("counters")}
	for _, target := range []struct{ collection, prefix string }{
		{"cycles", domain.CycleRefPrefix},
		{"reservations", domain.ReservationRefPrefix},
	} {
		coll := database.Collection(target.collection)
		cur, err := coll.Find(ctx, bson.M{"reference": bson.M{"$exists": false}},
			options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}).SetProjection(bson.M{"_id": 1}))
		if err != nil {
			return err
		}
		var docs []struct {
			ID bson.ObjectID `bson:"_id"`
		}
		if err := cur.All(ctx, &docs); err != nil {
			return err
		}
		for _, d := range docs {
			n, err := counters.Next(ctx, target.prefix)
			if err != nil {
				return err
			}
			if _, err := coll.UpdateOne(ctx, bson.M{"_id": d.ID, "reference": bson.M{"$exists": false}},
				bson.M{"$set": bson.M{"reference": domain.FormatReference(target.prefix, n)}}); err != nil {
				return fmt.Errorf("backfill %s %s: %w", target.collection, d.ID.Hex(), err)
			}
		}
	}
	return nil
}
