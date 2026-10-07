package ports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// ReservationRepository persists and queries reservations. Widened past
// Insert-only once payment, delivery, dispute and listing/history
// usecases became real callers.
type ReservationRepository interface {
	Insert(ctx context.Context, r *domain.Reservation) error

	// FindByID returns domain.ErrNotFound if no reservation has this ID.
	FindByID(ctx context.Context, id bson.ObjectID) (*domain.Reservation, error)

	ListByBuyer(ctx context.Context, buyerID bson.ObjectID) ([]*domain.Reservation, error)

	// ListByListingIDs backs a farmer's "my reservations" view: every
	// reservation against any of their own listings.
	ListByListingIDs(ctx context.Context, listingIDs []bson.ObjectID) ([]*domain.Reservation, error)

	// Update persists the full reservation document — payment,
	// delivery and status all move together often enough (see
	// application/usecase/reservation) that one whole-document replace
	// is simpler and safer than several narrow setters that could drift
	// out of sync with each other.
	Update(ctx context.Context, r *domain.Reservation) error

	// ListDeliveredBefore returns every delivered reservation whose
	// DeliveredAt is before the given time and whose payment is still
	// held — internal/worker's auto-release sweep.
	ListDeliveredBefore(ctx context.Context, before time.Time) ([]*domain.Reservation, error)
}
