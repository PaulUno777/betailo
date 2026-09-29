package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CycleRepository is the one write the reservation usecase needs from
// cycle persistence: an atomic decrement that fails closed
// (domain.ErrInsufficientCapacity) rather than overselling. See
// CycleStore for the fuller port application/usecase/cycle depends on —
// both are implemented by the same infrastructure/mongodb.CycleRepository
// concrete type; keeping this one narrow is what lets reservation's own
// tests use a one-method fake.
type CycleRepository interface {
	DecrementAvailability(ctx context.Context, cycleID bson.ObjectID, quantity int) error
}
