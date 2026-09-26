package cycle

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func newCycleFixture(t *testing.T, status domain.CycleStatus) (*fakeFarms, *fakeCycles, bson.ObjectID, *domain.Cycle) {
	t.Helper()
	owner := bson.NewObjectID()
	f := &domain.Farm{OwnerID: owner, Name: "Ferme", Location: "Douala"}
	farms := newFakeFarms(f)
	cycles := newFakeCycles()
	c := &domain.Cycle{FarmID: f.ID, Species: "poule", Status: status}
	if err := cycles.Create(context.Background(), c); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return farms, cycles, owner, c
}

func TestUpdateCycleStatus_Execute_AllowedTransitions(t *testing.T) {
	tests := []struct {
		from, to domain.CycleStatus
	}{
		{domain.CycleStatusPlanned, domain.CycleStatusInProgress},
		{domain.CycleStatusPlanned, domain.CycleStatusCancelled},
		{domain.CycleStatusInProgress, domain.CycleStatusDone},
		{domain.CycleStatusInProgress, domain.CycleStatusCancelled},
	}
	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			farms, cycles, owner, c := newCycleFixture(t, tt.from)
			uc := &UpdateCycleStatus{Farms: farms, Cycles: cycles}

			got, err := uc.Execute(context.Background(), UpdateCycleStatusInput{
				CycleID: c.ID, OwnerID: owner, Status: tt.to,
			})
			if err != nil {
				t.Fatalf("Execute() error = %v, want nil", err)
			}
			if got.Status != tt.to {
				t.Errorf("Status = %v, want %v", got.Status, tt.to)
			}
		})
	}
}

func TestUpdateCycleStatus_Execute_RejectsInvalidTransitions(t *testing.T) {
	tests := []struct {
		from, to domain.CycleStatus
	}{
		{domain.CycleStatusPlanned, domain.CycleStatusDone},      // can't skip in_progress
		{domain.CycleStatusDone, domain.CycleStatusInProgress},   // terminal state
		{domain.CycleStatusCancelled, domain.CycleStatusPlanned}, // terminal state
		{domain.CycleStatusPlanned, domain.CycleStatusPlanned},   // no-op transition
	}
	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			farms, cycles, owner, c := newCycleFixture(t, tt.from)
			uc := &UpdateCycleStatus{Farms: farms, Cycles: cycles}

			_, err := uc.Execute(context.Background(), UpdateCycleStatusInput{
				CycleID: c.ID, OwnerID: owner, Status: tt.to,
			})
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("Execute() error = %v, want domain.ErrInvalid", err)
			}
		})
	}
}

func TestUpdateCycleStatus_Execute_NotFarmOwner(t *testing.T) {
	farms, cycles, _, c := newCycleFixture(t, domain.CycleStatusPlanned)
	uc := &UpdateCycleStatus{Farms: farms, Cycles: cycles}

	_, err := uc.Execute(context.Background(), UpdateCycleStatusInput{
		CycleID: c.ID, OwnerID: bson.NewObjectID(), Status: domain.CycleStatusInProgress,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Execute() error = %v, want domain.ErrForbidden", err)
	}
}
