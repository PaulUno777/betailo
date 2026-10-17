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
	f := &domain.Farm{OwnerID: owner, Name: "Ferme", Location: "Douala", Species: []string{"poussins"}}
	farms := newFakeFarms(f)
	cycles := newFakeCycles()
	uc := &CreateCycle{Farms: farms, Cycles: cycles, Sequencer: &fakeSequencer{}}

	starts := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	ends := starts.Add(30 * 24 * time.Hour)

	c, err := uc.Execute(context.Background(), CreateCycleInput{
		FarmID: f.ID, OwnerID: owner, Species: "poussins", Type: "ponte",
		Capacity: 100, StartsAt: starts, EndsAt: &ends,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if c.Reference != "CYC-2610-0001" {
		t.Errorf("Reference = %q, want CYC-2610-0001", c.Reference)
	}
	if c.Type != "couvee" {
		t.Errorf("Type = %q, want couvee: the catalog's cycle type for poussins, not the one sent", c.Type)
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
	f := &domain.Farm{OwnerID: bson.NewObjectID(), Name: "Ferme", Location: "Douala", Species: []string{"poussins"}}
	uc := &CreateCycle{Farms: newFakeFarms(f), Cycles: newFakeCycles()}

	_, err := uc.Execute(context.Background(), CreateCycleInput{
		FarmID: f.ID, OwnerID: bson.NewObjectID(), Species: "poussins", Type: "ponte",
		Capacity: 10, StartsAt: time.Now(),
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Execute() error = %v, want domain.ErrForbidden", err)
	}
}

func TestCreateCycle_Execute_Validation(t *testing.T) {
	owner := bson.NewObjectID()
	f := &domain.Farm{ID: bson.NewObjectID(), OwnerID: owner, Name: "Ferme", Location: "Douala", Species: []string{"poussins"}}
	starts := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	before := starts.Add(-time.Hour)

	tests := []struct {
		name string
		in   CreateCycleInput
	}{
		{"missing species", CreateCycleInput{Type: "ponte", Capacity: 10, StartsAt: starts}},
		{"a species the catalog doesn't offer", CreateCycleInput{Species: "licornes", Capacity: 10, StartsAt: starts}},
		{"a species of an inactive category", CreateCycleInput{Species: "veaux", Capacity: 10, StartsAt: starts}},
		{"zero capacity", CreateCycleInput{Species: "poussins", Type: "ponte", Capacity: 0, StartsAt: starts}},
		{"negative capacity", CreateCycleInput{Species: "poussins", Type: "ponte", Capacity: -1, StartsAt: starts}},
		{"zero startsAt", CreateCycleInput{Species: "poussins", Type: "ponte", Capacity: 10}},
		{"endsAt before startsAt", CreateCycleInput{Species: "poussins", Type: "ponte", Capacity: 10, StartsAt: starts, EndsAt: &before}},
		{"endsAt equal startsAt", CreateCycleInput{Species: "poussins", Type: "ponte", Capacity: 10, StartsAt: starts, EndsAt: &starts}},
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

// A farm reviewed for its poultry can't open a pig cycle until its
// owner declares pigs.
func TestCreateCycle_Execute_SpeciesNotDeclared(t *testing.T) {
	owner := bson.NewObjectID()
	f := &domain.Farm{OwnerID: owner, Name: "Ferme", Location: "Douala", Species: []string{"poussins"}}
	uc := &CreateCycle{Farms: newFakeFarms(f), Cycles: newFakeCycles(), Sequencer: &fakeSequencer{}}

	_, err := uc.Execute(context.Background(), CreateCycleInput{
		FarmID: f.ID, OwnerID: owner, Species: "porcs", Capacity: 10, StartsAt: time.Now(),
	})
	if code, _ := domain.CodeOf(err); !errors.Is(err, domain.ErrInvalid) || code != "farm.species_not_declared" {
		t.Fatalf("Execute() error = %v, want farm.species_not_declared", err)
	}
}
