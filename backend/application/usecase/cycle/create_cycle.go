// Package cycle implements CRUD for the domain pivot — Phase 2 of the
// roadmap. Like application/usecase/farm, every usecase here depends
// only on application/ports.
package cycle

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// CreateCycle is the usecase: validate input (dates, capacity), confirm
// the caller owns the farm, and persist a new cycle. No Clock
// dependency — unlike User/Reservation, Cycle has no CreatedAt field;
// StartsAt/EndsAt are caller-supplied business dates, not audit
// timestamps.
type CreateCycle struct {
	Farms  ports.FarmRepository
	Cycles ports.CycleStore
}

// CreateCycleInput is what a caller (the HTTP handler) supplies.
// OwnerID comes from the authenticated user's token.
type CreateCycleInput struct {
	FarmID   bson.ObjectID
	OwnerID  bson.ObjectID
	Species  string
	Type     string
	Capacity int
	StartsAt time.Time
	EndsAt   *time.Time
}

// Execute returns domain.ErrNotFound if the farm doesn't exist,
// domain.ErrForbidden if OwnerID isn't the farm's owner, and
// domain.ErrInvalid for a malformed input (missing species/type,
// non-positive capacity, a zero start date, or an end date not after
// the start date).
func (uc *CreateCycle) Execute(ctx context.Context, in CreateCycleInput) (*domain.Cycle, error) {
	f, err := uc.Farms.FindByID(ctx, in.FarmID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != in.OwnerID {
		return nil, fmt.Errorf("cycle: not the farm owner: %w", domain.ErrForbidden)
	}

	if in.Species == "" {
		return nil, fmt.Errorf("cycle: species is required: %w", domain.ErrInvalid)
	}
	if in.Type == "" {
		return nil, fmt.Errorf("cycle: type is required: %w", domain.ErrInvalid)
	}
	if in.Capacity <= 0 {
		return nil, fmt.Errorf("cycle: capacity must be positive: %w", domain.ErrInvalid)
	}
	if in.StartsAt.IsZero() {
		return nil, fmt.Errorf("cycle: startsAt is required: %w", domain.ErrInvalid)
	}
	if in.EndsAt != nil && !in.EndsAt.After(in.StartsAt) {
		return nil, fmt.Errorf("cycle: endsAt must be after startsAt: %w", domain.ErrInvalid)
	}

	c := &domain.Cycle{
		FarmID:   in.FarmID,
		Species:  in.Species,
		Type:     in.Type,
		Status:   domain.CycleStatusPlanned,
		Capacity: in.Capacity,
		// AvailableQuantity starts equal to Capacity — reservation is the
		// only thing that ever decrements it afterward (see
		// application/usecase/reservation.ReserveCycle).
		AvailableQuantity: in.Capacity,
		StartsAt:          in.StartsAt,
		EndsAt:            in.EndsAt,
	}

	if err := uc.Cycles.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}
