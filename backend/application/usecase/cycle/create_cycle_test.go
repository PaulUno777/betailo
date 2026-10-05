package cycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestCreateCycle_Execute_Success(t *testing.T) {
	owner := bson.NewObjectID()
	f := &domain.Farm{OwnerID: owner, Name: "Ferme", Location: "Douala"}
	farms := newFakeFarms(f)
	cycles := newFakeCycles()
	uc := &CreateCycle{Farms: farms, Cycles: cycles, Sequencer: &fakeSequencer{}}

	starts := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	ends := starts.Add(30 * 24 * time.Hour)

	c, err := uc.Execute(context.Background(), CreateCycleInput{
		FarmID: f.ID, OwnerID: owner, Species: "poule", Type: "ponte",
		Capacity: 100, StartsAt: starts, EndsAt: &ends,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if c.Reference != "CYC-0001" {
		t.Errorf("Reference = %q, want CYC-0001", c.Reference)
	}
	if c.Status != domain.CycleStatusPlanned {
		t.Errorf("Status = %v, want %v", c.Status, domain.CycleStatusPlanned)
	}
	if c.AvailableQuantity != c.Capacity {
		t.Errorf("AvailableQuantity = %d, want equal to Capacity (%d)", c.AvailableQuantity, c.Capacity)
	}
	if c.ID.IsZero() {
		t.Error("ID is zero, want the repository to have assigned one")
	}
}

func TestCreateCycle_Execute_NotFarmOwner(t *testing.T) {
	f := &domain.Farm{OwnerID: bson.NewObjectID(), Name: "Ferme", Location: "Douala"}
	uc := &CreateCycle{Farms: newFakeFarms(f), Cycles: newFakeCycles()}

	_, err := uc.Execute(context.Background(), CreateCycleInput{
		FarmID: f.ID, OwnerID: bson.NewObjectID(), Species: "poule", Type: "ponte",
		Capacity: 10, StartsAt: time.Now(),
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Execute() error = %v, want domain.ErrForbidden", err)
	}
}

func TestCreateCycle_Execute_Validation(t *testing.T) {
	owner := bson.NewObjectID()
	f := &domain.Farm{ID: bson.NewObjectID(), OwnerID: owner, Name: "Ferme", Location: "Douala"}
	starts := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	before := starts.Add(-time.Hour)

	tests := []struct {
		name string
		in   CreateCycleInput
	}{
		{"missing species", CreateCycleInput{Type: "ponte", Capacity: 10, StartsAt: starts}},
		{"missing type", CreateCycleInput{Species: "poule", Capacity: 10, StartsAt: starts}},
		{"zero capacity", CreateCycleInput{Species: "poule", Type: "ponte", Capacity: 0, StartsAt: starts}},
		{"negative capacity", CreateCycleInput{Species: "poule", Type: "ponte", Capacity: -1, StartsAt: starts}},
		{"zero startsAt", CreateCycleInput{Species: "poule", Type: "ponte", Capacity: 10}},
		{"endsAt before startsAt", CreateCycleInput{Species: "poule", Type: "ponte", Capacity: 10, StartsAt: starts, EndsAt: &before}},
		{"endsAt equal startsAt", CreateCycleInput{Species: "poule", Type: "ponte", Capacity: 10, StartsAt: starts, EndsAt: &starts}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.in.FarmID = f.ID
			tt.in.OwnerID = owner
			uc := &CreateCycle{Farms: newFakeFarms(f), Cycles: newFakeCycles()}
			_, err := uc.Execute(context.Background(), tt.in)
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("Execute() error = %v, want domain.ErrInvalid", err)
			}
		})
	}
}
