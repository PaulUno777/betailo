package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// CycleRepository is the reservation usecase's slice of cycle
// persistence: read a cycle (to validate min/max before deciding
// whether to reserve at all) and the atomic increment/decrement pair
// that adjusts availableQuantity without ever overselling
// (domain.ErrInsufficientCapacity on failure). See CycleStore for the
// fuller port application/usecase/cycle depends on — both are
// implemented by the same infrastructure/mongodb.CycleRepository
// concrete type; keeping this one narrow is what lets reservation's own
// tests use a small fake instead of the full CRUD surface.
type CycleRepository interface {
	FindByID(ctx context.Context, id bson.ObjectID) (*domain.Cycle, error)
	DecrementAvailability(ctx context.Context, cycleID bson.ObjectID, quantity int) error
	IncrementAvailability(ctx context.Context, cycleID bson.ObjectID, quantity int) error
}
