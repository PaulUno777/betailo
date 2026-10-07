package cycle

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// allowedCycleTransitions is the cycle's state machine: planned can
// start or be cancelled; in_progress can finish or be cancelled; done
// and cancelled are terminal. This is a real validation rule beyond
// dates/quantities — a cycle can't jump from planned straight to done,
// or leave a terminal state.
var allowedCycleTransitions = map[domain.CycleStatus]map[domain.CycleStatus]bool{
	domain.CycleStatusPlanned: {
		domain.CycleStatusInProgress: true,
		domain.CycleStatusCancelled:  true,
	},
	domain.CycleStatusInProgress: {
		domain.CycleStatusDone:      true,
		domain.CycleStatusCancelled: true,
	},
}

// nextStatusOrder fixes the order NextStatuses lists transitions in, so
// clients render the forward step before cancellation.
var nextStatusOrder = []domain.CycleStatus{
	domain.CycleStatusInProgress,
	domain.CycleStatusDone,
	domain.CycleStatusCancelled,
}

// NextStatuses lists what a cycle in status s may move to, in a stable
// order. Clients show these as the cycle's actions instead of keeping
// their own copy of the state machine.
func NextStatuses(s domain.CycleStatus) []domain.CycleStatus {
	out := []domain.CycleStatus{}
	for _, next := range nextStatusOrder {
		if allowedCycleTransitions[s][next] {
			out = append(out, next)
		}
	}
	return out
}

// UpdateCycleStatus is the usecase: transition a cycle's status,
// enforcing the state machine above, after confirming the caller owns
// the cycle's farm.
type UpdateCycleStatus struct {
	Farms  ports.FarmRepository
	Cycles ports.CycleStore
	// Unpaid, when set, empties the cycle's waiting list once the cycle
	// closes (done or cancelled).
	Unpaid UnpaidCloser
}

// UnpaidCloser cancels a closed cycle's still-unpaid reservations
// (reservation.CloseUnpaidForCycle).
type UnpaidCloser interface {
	CloseUnpaid(ctx context.Context, cycleID bson.ObjectID) (int, error)
}

// UpdateCycleStatusInput is what a caller (the HTTP handler) supplies.
type UpdateCycleStatusInput struct {
	CycleID bson.ObjectID
	OwnerID bson.ObjectID
	Status  domain.CycleStatus
}

// Execute returns domain.ErrNotFound if the cycle or its farm doesn't
// exist, domain.ErrForbidden if OwnerID isn't the farm's owner, and
// domain.ErrInvalid if the requested transition isn't allowed from the
// cycle's current status.
func (uc *UpdateCycleStatus) Execute(ctx context.Context, in UpdateCycleStatusInput) (*domain.Cycle, error) {
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

	if !allowedCycleTransitions[c.Status][in.Status] {
		return nil, domain.Coded("cycle.invalid_transition", fmt.Errorf("cycle: cannot transition from %q to %q: %w", c.Status, in.Status, domain.ErrInvalid))
	}

	if err := uc.Cycles.UpdateStatus(ctx, in.CycleID, in.Status); err != nil {
		return nil, err
	}
	c.Status = in.Status
	if uc.Unpaid != nil && (c.Status == domain.CycleStatusDone || c.Status == domain.CycleStatusCancelled) {
		if _, err := uc.Unpaid.CloseUnpaid(ctx, c.ID); err != nil {
			return nil, err
		}
	}
	return c, nil
}
