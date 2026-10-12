package cycle

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// UpdateCycle edits a cycle's schedule, reservation bounds and capacity
// (status has its own state machine in UpdateCycleStatus). Capacity can
// grow freely but only shrink by places nobody has reserved: the change
// is applied atomically against the live availability, so a
// reservation landing at the same moment can't be oversold.
type UpdateCycle struct {
	Farms  ports.FarmRepository
	Cycles ports.CycleStore
	Editor ports.CycleEditor
}

type UpdateCycleInput struct {
	CycleID           bson.ObjectID
	OwnerID           bson.ObjectID
	StartsAt          time.Time
	EndsAt            *time.Time
	Capacity          int
	MinPerReservation int
	MaxPerReservation int
}

func (uc *UpdateCycle) Execute(ctx context.Context, in UpdateCycleInput) (*domain.Cycle, error) {
	c, err := uc.Cycles.FindByID(ctx, in.CycleID)
	if err != nil {
		return nil, err
	}
	f, err := uc.Farms.FindByID(ctx, c.FarmID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != in.OwnerID {
		return nil, fmt.Errorf("cycle: not the farm owner: %w", domain.ErrForbidden)
	}
	if c.Status == domain.CycleStatusDone || c.Status == domain.CycleStatusCancelled {
		return nil, fmt.Errorf("cycle: a closed cycle can't be edited: %w", domain.ErrConflict)
	}
	switch {
	case in.StartsAt.IsZero():
		return nil, fmt.Errorf("cycle: startsAt is required: %w", domain.ErrInvalid)
	case in.EndsAt != nil && !in.EndsAt.After(in.StartsAt):
		return nil, fmt.Errorf("cycle: endsAt must be after startsAt: %w", domain.ErrInvalid)
	case in.Capacity <= 0:
		return nil, fmt.Errorf("cycle: capacity must be positive: %w", domain.ErrInvalid)
	case in.MinPerReservation < 0 || in.MaxPerReservation < 0:
		return nil, fmt.Errorf("cycle: reservation bounds must not be negative: %w", domain.ErrInvalid)
	case in.MaxPerReservation > in.Capacity:
		return nil, fmt.Errorf("cycle: maxPerReservation must not exceed capacity: %w", domain.ErrInvalid)
	case in.MinPerReservation > 0 && in.MaxPerReservation > 0 && in.MinPerReservation > in.MaxPerReservation:
		return nil, fmt.Errorf("cycle: minPerReservation must not exceed maxPerReservation: %w", domain.ErrInvalid)
	}

	details := ports.CycleDetails{
		StartsAt: in.StartsAt, EndsAt: in.EndsAt,
		MinPerReservation: in.MinPerReservation, MaxPerReservation: in.MaxPerReservation,
	}
	if err := uc.Editor.UpdateDetails(ctx, c.ID, details, in.Capacity-c.Capacity); err != nil {
		return nil, err
	}
	return uc.Cycles.FindByID(ctx, c.ID)
}
