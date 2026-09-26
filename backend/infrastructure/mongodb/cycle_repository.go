package mongodb

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// CycleRepository implements ports.CycleRepository against the
// "cycles" collection.
type CycleRepository struct {
	Coll *mongo.Collection
}

var _ ports.CycleRepository = (*CycleRepository)(nil)

// DecrementAvailability implements ports.CycleRepository via the atomic
// findOneAndUpdate + $inc described in the sequence diagram: the
// condition (availableQuantity >= quantity) and the decrement happen in
// a single document-level operation, which MongoDB guarantees is
// atomic — no explicit lock needed even under concurrent reservations.
func (r *CycleRepository) DecrementAvailability(ctx context.Context, cycleID bson.ObjectID, quantity int) error {
	filter := bson.M{"_id": cycleID, "availableQuantity": bson.M{"$gte": quantity}}
	update := bson.M{"$inc": bson.M{"availableQuantity": -quantity}}

	if err := r.Coll.FindOneAndUpdate(ctx, filter, update).Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.ErrInsufficientCapacity
		}
		return err
	}
	return nil
}
