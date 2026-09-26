package farm

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestDeleteFarm_Execute_Success(t *testing.T) {
	farms := newFakeFarms()
	owner := bson.NewObjectID()
	f := &domain.Farm{OwnerID: owner, Name: "Ferme", Location: "Douala"}
	if err := farms.Create(context.Background(), f); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	cycles := &fakeCycleStore{byFarm: map[bson.ObjectID][]*domain.Cycle{}}

	uc := &DeleteFarm{Farms: farms, Cycles: cycles}
	if err := uc.Execute(context.Background(), f.ID, owner); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	if _, err := farms.FindByID(context.Background(), f.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Error("farm still exists after delete")
	}
}

func TestDeleteFarm_Execute_HasCycles(t *testing.T) {
	farms := newFakeFarms()
	owner := bson.NewObjectID()
	f := &domain.Farm{OwnerID: owner, Name: "Ferme", Location: "Douala"}
	if err := farms.Create(context.Background(), f); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	cycles := &fakeCycleStore{byFarm: map[bson.ObjectID][]*domain.Cycle{
		f.ID: {{ID: bson.NewObjectID(), FarmID: f.ID}},
	}}

	uc := &DeleteFarm{Farms: farms, Cycles: cycles}
	err := uc.Execute(context.Background(), f.ID, owner)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("Execute() error = %v, want domain.ErrConflict", err)
	}

	if _, err := farms.FindByID(context.Background(), f.ID); err != nil {
		t.Error("farm was deleted despite having cycles")
	}
}

func TestDeleteFarm_Execute_NotOwner(t *testing.T) {
	farms := newFakeFarms()
	f := &domain.Farm{OwnerID: bson.NewObjectID(), Name: "Ferme", Location: "Douala"}
	if err := farms.Create(context.Background(), f); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	cycles := &fakeCycleStore{byFarm: map[bson.ObjectID][]*domain.Cycle{}}

	uc := &DeleteFarm{Farms: farms, Cycles: cycles}
	err := uc.Execute(context.Background(), f.ID, bson.NewObjectID())
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Execute() error = %v, want domain.ErrForbidden", err)
	}
}
