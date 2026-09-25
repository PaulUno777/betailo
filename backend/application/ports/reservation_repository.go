package ports

import (
	"context"

	"github.com/pauluno777/betailo/backend/domain"
)

// ReservationRepository persists a Reservation. Kept to the one method
// the reservation usecase needs — widen it only when a second real
// caller shows up.
type ReservationRepository interface {
	Insert(ctx context.Context, r *domain.Reservation) error
}
