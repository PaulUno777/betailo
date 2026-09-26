package farm

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestGetFarm_Execute(t *testing.T) {
	farms := newFakeFarms()
	created := &domain.Farm{OwnerID: bson.NewObjectID(), Name: "Ferme", Location: "Douala"}
	if err := farms.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	uc := &GetFarm{Farms: farms}

	got, err := uc.Execute(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got.Name != "Ferme" {
		t.Errorf("Name = %q, want %q", got.Name, "Ferme")
	}

	_, err = uc.Execute(context.Background(), bson.NewObjectID())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Execute() with unknown ID: error = %v, want domain.ErrNotFound", err)
	}
}

func TestListMyFarms_Execute(t *testing.T) {
	farms := newFakeFarms()
	ownerA, ownerB := bson.NewObjectID(), bson.NewObjectID()
	now := time.Now()
	for _, f := range []*domain.Farm{
		{OwnerID: ownerA, Name: "A1", Location: "Douala", CreatedAt: now},
		{OwnerID: ownerA, Name: "A2", Location: "Yaoundé", CreatedAt: now},
		{OwnerID: ownerB, Name: "B1", Location: "Bafoussam", CreatedAt: now},
	} {
		if err := farms.Create(context.Background(), f); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	uc := &ListMyFarms{Farms: farms}

	got, err := uc.Execute(context.Background(), ownerA)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d farms, want 2 (only ownerA's)", len(got))
	}
}
