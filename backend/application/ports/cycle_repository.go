package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CycleRepository is the one write the reservation usecase needs from
// cycle persistence: an atomic decrement that fails closed
// (domain.ErrInsufficientCapacity) rather than overselling.
type CycleRepository interface {
	DecrementAvailability(ctx context.Context, cycleID bson.ObjectID, quantity int) error
}
