package farm

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestCreateFarm_Execute_Success(t *testing.T) {
	farms := newFakeFarms()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	trust := &fakeTrust{}
	uc := &CreateFarm{Farms: farms, Trust: trust, Clock: fakeClock{now: now}}
	ownerID := bson.NewObjectID()

	out, err := uc.Execute(context.Background(), CreateFarmInput{
		OwnerID:      ownerID,
		Name:         "  Ferme du Soleil  ",
		Location:     "Douala",
		Species:      []string{"poule", "porc"},
		Description:  "  Élevage familial depuis 2012.  ",
		ContactPhone: "6 90 12 34 56",
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
	if out.Farm.Description != "Élevage familial depuis 2012." {
		t.Errorf("Description = %q, want trimmed", out.Farm.Description)
	}
	if out.Farm.ContactPhone != "+237690123456" {
		t.Errorf("ContactPhone = %q, want E.164", out.Farm.ContactPhone)
	}
	if len(trust.calls) != 1 || trust.calls[0] != ownerID {
		t.Errorf("trust recompute calls = %v, want exactly the owner, for a baseline score", trust.calls)
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
		{"bad phone", CreateFarmInput{Name: "Ferme", Location: "Douala", ContactPhone: "12345"}},
		{"description too long", CreateFarmInput{Name: "Ferme", Location: "Douala", Description: strings.Repeat("é", domain.MaxFarmDescriptionLen+1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &CreateFarm{Farms: newFakeFarms(), Trust: &fakeTrust{}, Clock: fakeClock{now: time.Now()}}
			_, err := uc.Execute(context.Background(), tt.in)
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("Execute() error = %v, want domain.ErrInvalid", err)
			}
		})
	}
}
