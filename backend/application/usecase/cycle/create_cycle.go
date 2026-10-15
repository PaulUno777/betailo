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
	"github.com/pauluno777/betailo/backend/application/usecase/catalog"
	"github.com/pauluno777/betailo/backend/domain"
)

// CreateCycle is the usecase: validate input (dates, capacity), confirm
// the caller owns the farm, and persist a new cycle. No Clock
// dependency — unlike User/Reservation, Cycle has no CreatedAt field;
// StartsAt/EndsAt are caller-supplied business dates, not audit
// timestamps.
type CreateCycle struct {
	Farms     ports.FarmRepository
	Cycles    ports.CycleStore
	Sequencer ports.Sequencer
	// Catalog says which species can be raised and in which cycle type.
	Catalog ports.CatalogSource
}

// CreateCycleInput is what a caller (the HTTP handler) supplies.
// OwnerID comes from the authenticated user's token.
type CreateCycleInput struct {
	FarmID  bson.ObjectID
	OwnerID bson.ObjectID
	// Species is a catalog sub-category key; the cycle's type is the
	// one the catalog gives it, whatever Type says.
	Species  string
	Type     string
	Capacity int
	StartsAt time.Time
	EndsAt   *time.Time

	// MinPerReservation/MaxPerReservation bound a single buyer's
	// reservation quantity — 0 means "no bound". Enforced here at
	// creation and by application/usecase/reservation.ReserveCycle
	// against every reservation attempt.
	MinPerReservation int
	MaxPerReservation int
}

// Execute returns domain.ErrNotFound if the farm doesn't exist,
// domain.ErrForbidden if OwnerID isn't the farm's owner, and
// domain.ErrInvalid for a malformed input (a species the catalog
// doesn't offer,
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

	cat, err := catalog.Of(ctx, uc.Catalog)
	if err != nil {
		return nil, err
	}
	sp, err := cat.SpeciesByKey(in.Species)
	if err != nil {
		return nil, err
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
	if in.MinPerReservation < 0 || in.MaxPerReservation < 0 {
		return nil, fmt.Errorf("cycle: minPerReservation/maxPerReservation must not be negative: %w", domain.ErrInvalid)
	}
	if in.MaxPerReservation > 0 && in.MaxPerReservation > in.Capacity {
		return nil, fmt.Errorf("cycle: maxPerReservation must not exceed capacity: %w", domain.ErrInvalid)
	}
	if in.MinPerReservation > 0 && in.MaxPerReservation > 0 && in.MinPerReservation > in.MaxPerReservation {
		return nil, fmt.Errorf("cycle: minPerReservation must not exceed maxPerReservation: %w", domain.ErrInvalid)
	}

	c := &domain.Cycle{
		FarmID:   in.FarmID,
		Species:  sp.Key,
		Type:     sp.CycleType,
		Status:   domain.CycleStatusPlanned,
		Capacity: in.Capacity,
		// AvailableQuantity starts equal to Capacity — reservation is the
		// only thing that ever decrements it afterward (see
		// application/usecase/reservation.ReserveCycle).
		AvailableQuantity: in.Capacity,
		StartsAt:          in.StartsAt,
		EndsAt:            in.EndsAt,
		MinPerReservation: in.MinPerReservation,
		MaxPerReservation: in.MaxPerReservation,
	}

	seq, err := uc.Sequencer.Next(ctx, domain.CycleRefPrefix)
	if err != nil {
		return nil, err
	}
	c.Reference = domain.FormatReference(domain.CycleRefPrefix, seq)

	if err := uc.Cycles.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}
