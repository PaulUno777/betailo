package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// CycleStore is the Cycle aggregate's CRUD port, used by
// application/usecase/cycle and by application/usecase/listing to
// verify a cycle exists (and belongs to the right farm) before creating
// a listing against it. It is distinct from CycleRepository —
// reservation's narrow, one-method slice of the same aggregate — so
// reservation's own tests can keep using a one-method fake; the real
// infrastructure/mongodb.CycleRepository type implements both.
type CycleStore interface {
	Create(ctx context.Context, c *domain.Cycle) error

	// FindByID returns domain.ErrNotFound if no cycle has this ID.
	FindByID(ctx context.Context, id bson.ObjectID) (*domain.Cycle, error)

	ListByFarm(ctx context.Context, farmID bson.ObjectID) ([]*domain.Cycle, error)

	// ListByStatus returns every cycle in the given status, across every
	// farm — internal/worker's sweep for automatic status transitions
	// (planned -> in_progress -> done) needs a global view, unlike every
	// other caller of this port which scopes to one farm.
	ListByStatus(ctx context.Context, status domain.CycleStatus) ([]*domain.Cycle, error)

	// UpdateStatus is deliberately narrower than a general update: a
	// cycle's status is the one field usecases change after creation —
	// see application/usecase/cycle.UpdateCycleStatus's state machine.
	UpdateStatus(ctx context.Context, id bson.ObjectID, status domain.CycleStatus) error

	// Delete removes the cycle unconditionally — the "no existing
	// listing" invariant is checked by the usecase, not here, for the
	// same reason as FarmRepository.Delete. See
	// application/usecase/cycle.DeleteCycle.
	Delete(ctx context.Context, id bson.ObjectID) error
}
