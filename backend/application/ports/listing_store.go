package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// ListingStore is the Listing aggregate's CRUD port, used by
// application/usecase/listing. It is distinct from ListingRepository —
// reservation's narrow, one-method slice of the same aggregate — so
// reservation's own tests can keep using a one-method fake; the real
// infrastructure/mongodb.ListingRepository type implements both.
type ListingStore interface {
	Create(ctx context.Context, l *domain.Listing) error

	// FindByID returns domain.ErrNotFound if no listing has this ID.
	FindByID(ctx context.Context, id bson.ObjectID) (*domain.Listing, error)

	// FindByCycleID returns domain.ErrNotFound if no listing references
	// this cycle — used by application/usecase/cycle.DeleteCycle to
	// enforce that a cycle with a listing can't be deleted out from
	// under it.
	FindByCycleID(ctx context.Context, cycleID bson.ObjectID) (*domain.Listing, error)

	Update(ctx context.Context, l *domain.Listing) error
}
