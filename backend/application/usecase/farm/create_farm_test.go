package farm

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestCreateFarm_Execute_Success(t *testing.T) {
	farms := newFakeFarms()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	uc := &CreateFarm{Farms: farms, Clock: fakeClock{now: now}}
	ownerID := bson.NewObjectID()

	out, err := uc.Execute(context.Background(), CreateFarmInput{
		OwnerID:  ownerID,
		Name:     "  Ferme du Soleil  ",
		Location: "Douala",
		Species:  []string{"poule", "porc"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if out.Farm.Name != "Ferme du Soleil" {
		t.Errorf("Name = %q, want trimmed", out.Farm.Name)
	}
	if out.Farm.OwnerID != ownerID {
		t.Errorf("OwnerID = %v, want %v", out.Farm.OwnerID, ownerID)
	}
	if !out.Farm.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", out.Farm.CreatedAt, now)
	}
	if out.Farm.ID.IsZero() {
		t.Error("ID is zero, want the repository to have assigned one")
	}
}

func TestCreateFarm_Execute_Validation(t *testing.T) {
	tests := []struct {
		name string
		in   CreateFarmInput
	}{
		{"empty name", CreateFarmInput{Name: "", Location: "Douala"}},
		{"blank name", CreateFarmInput{Name: "   ", Location: "Douala"}},
		{"empty location", CreateFarmInput{Name: "Ferme", Location: ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &CreateFarm{Farms: newFakeFarms(), Clock: fakeClock{now: time.Now()}}
			_, err := uc.Execute(context.Background(), tt.in)
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("Execute() error = %v, want domain.ErrInvalid", err)
			}
		})
	}
}
