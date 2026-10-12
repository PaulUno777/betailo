package cycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestGetCycle_Execute(t *testing.T) {
	cycles := newFakeCycles()
	c := &domain.Cycle{FarmID: bson.NewObjectID(), Species: "poule"}
	if err := cycles.Create(context.Background(), c); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	uc := &GetCycle{Cycles: cycles, Clock: fakeClock{now: time.Now()}}
	got, err := uc.Execute(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got.Cycle.Species != "poule" {
		t.Errorf("Species = %q, want %q", got.Cycle.Species, "poule")
	}
	if len(got.Timeline) == 0 || got.Timeline[0].Kind != domain.TimelineStart {
		t.Errorf("Timeline = %+v, want it to open with the start step", got.Timeline)
	}

	_, err = uc.Execute(context.Background(), bson.NewObjectID())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Execute() with unknown ID: error = %v, want domain.ErrNotFound", err)
	}
}

func TestListCyclesByFarm_Execute(t *testing.T) {
	cycles := newFakeCycles()
	farmA, farmB := bson.NewObjectID(), bson.NewObjectID()
	for _, c := range []*domain.Cycle{
		{FarmID: farmA, Species: "poule"},
		{FarmID: farmA, Species: "porc"},
		{FarmID: farmB, Species: "chèvre"},
	} {
		if err := cycles.Create(context.Background(), c); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	uc := &ListCyclesByFarm{Cycles: cycles}
	got, err := uc.Execute(context.Background(), farmA)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d cycles, want 2 (only farmA's)", len(got))
	}
}
