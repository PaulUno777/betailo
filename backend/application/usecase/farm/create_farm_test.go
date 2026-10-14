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
	uc := &CreateFarm{Users: &fakeUsers{}, Farms: farms, Trust: trust, Clock: fakeClock{now: now}}
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
			uc := &CreateFarm{Users: &fakeUsers{}, Farms: newFakeFarms(), Trust: &fakeTrust{}, Clock: fakeClock{now: time.Now()}}
			_, err := uc.Execute(context.Background(), tt.in)
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("Execute() error = %v, want domain.ErrInvalid", err)
			}
		})
	}
}

// The platform default (MAX_FARMS_PER_ACCOUNT) bounds an account's
// farms; what staff set for one account wins; staff don't sell.
func TestCreateFarm_Execute_LimitPerAccount(t *testing.T) {
	ctx := context.Background()
	regular, raised, staff := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
	users := &fakeUsers{byID: map[bson.ObjectID]*domain.User{
		raised: {ID: raised, FarmLimit: 2},
		staff:  {ID: staff, StaffRole: domain.StaffSuperAdmin},
	}}
	uc := &CreateFarm{Users: users, Farms: newFakeFarms(), MaxFarms: 1, Trust: &fakeTrust{}, Clock: fakeClock{now: time.Now()}}
	open := func(owner bson.ObjectID) error {
		_, err := uc.Execute(ctx, CreateFarmInput{OwnerID: owner, Name: "Ferme", Location: "Douala"})
		return err
	}

	if err := open(regular); err != nil {
		t.Fatalf("first farm: %v", err)
	}
	err := open(regular)
	if code, params := domain.CodeOf(err); !errors.Is(err, domain.ErrConflict) || code != "farm.limit_reached" || params["max"] != 1 {
		t.Fatalf("second farm at the default limit: err = %v (code %q, params %v), want conflict farm.limit_reached max=1", err, code, params)
	}

	if err := errors.Join(open(raised), open(raised)); err != nil {
		t.Fatalf("an account staff raised to 2 should open two farms: %v", err)
	}
	if err := open(raised); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("third farm for an account limited to 2: err = %v, want conflict", err)
	}

	err = open(staff)
	if code, _ := domain.CodeOf(err); !errors.Is(err, domain.ErrForbidden) || code != "farm.staff_cannot_sell" {
		t.Fatalf("staff opening a farm: err = %v, want forbidden farm.staff_cannot_sell", err)
	}
}

func TestFarmLimitFor(t *testing.T) {
	tests := []struct {
		name               string
		account, env, want int
	}{
		{"nothing set falls back to the built-in default", 0, 0, domain.DefaultMaxFarmsPerAccount},
		{"platform default from the environment", 0, 3, 3},
		{"what staff set for the account wins", 5, 3, 5},
	}
	for _, tt := range tests {
		if got := FarmLimitFor(&domain.User{FarmLimit: tt.account}, tt.env); got != tt.want {
			t.Errorf("%s: FarmLimitFor = %d, want %d", tt.name, got, tt.want)
		}
	}
}
