package farm

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestUpdateFarm_Execute_Success(t *testing.T) {
	farms := newFakeFarms()
	owner := bson.NewObjectID()
	f := &domain.Farm{OwnerID: owner, Name: "Old", Location: "Douala"}
	if err := farms.Create(context.Background(), f); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	uc := &UpdateFarm{Farms: farms}
	got, err := uc.Execute(context.Background(), UpdateFarmInput{
		FarmID: f.ID, OwnerID: owner, Name: "New", Location: "Yaoundé", Species: []string{"poule"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got.Name != "New" || got.Location != "Yaoundé" {
		t.Errorf("got = %+v, want updated fields", got)
	}
}

func TestUpdateFarm_Execute_NotOwner(t *testing.T) {
	farms := newFakeFarms()
	f := &domain.Farm{OwnerID: bson.NewObjectID(), Name: "Old", Location: "Douala"}
	if err := farms.Create(context.Background(), f); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	uc := &UpdateFarm{Farms: farms}
	_, err := uc.Execute(context.Background(), UpdateFarmInput{
		FarmID: f.ID, OwnerID: bson.NewObjectID(), Name: "New", Location: "Yaoundé",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Execute() error = %v, want domain.ErrForbidden", err)
	}
}

func TestUpdateFarm_Execute_NotFound(t *testing.T) {
	uc := &UpdateFarm{Farms: newFakeFarms()}
	_, err := uc.Execute(context.Background(), UpdateFarmInput{
		FarmID: bson.NewObjectID(), OwnerID: bson.NewObjectID(), Name: "New", Location: "Yaoundé",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Execute() error = %v, want domain.ErrNotFound", err)
	}
}
