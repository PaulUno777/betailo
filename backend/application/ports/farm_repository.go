package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// FarmRepository is the Farm aggregate's persistence port, used by
// application/usecase/farm's CRUD usecases and by
// application/usecase/cycle to verify farm ownership before creating a
// cycle.
type FarmRepository interface {
	Create(ctx context.Context, f *domain.Farm) error

	// FindByID returns domain.ErrNotFound if no farm has this ID.
	FindByID(ctx context.Context, id bson.ObjectID) (*domain.Farm, error)

	ListByOwner(ctx context.Context, ownerID bson.ObjectID) ([]*domain.Farm, error)

	// ListByStatus feeds the back office's review queue, oldest
	// submission first; an empty status lists every farm.
	ListByStatus(ctx context.Context, status domain.FarmStatus) ([]*domain.Farm, error)

	Update(ctx context.Context, f *domain.Farm) error

	// Delete removes the farm unconditionally — the "no existing
	// cycles" invariant is checked by the usecase, not here, since
	// enforcing it would mean this repository reaching into the cycles
	// collection it doesn't own. See
	// application/usecase/farm.DeleteFarm.
	Delete(ctx context.Context, id bson.ObjectID) error
}
