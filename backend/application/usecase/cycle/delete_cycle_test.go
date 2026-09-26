package cycle

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestDeleteCycle_Execute_Success(t *testing.T) {
	farms, cycles, owner, c := newCycleFixture(t, domain.CycleStatusPlanned)
	listings := &fakeListings{byCycle: map[bson.ObjectID]*domain.Listing{}}
	uc := &DeleteCycle{Farms: farms, Cycles: cycles, Listings: listings}

	if err := uc.Execute(context.Background(), c.ID, owner); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if _, err := cycles.FindByID(context.Background(), c.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Error("cycle still exists after delete")
	}
}

func TestDeleteCycle_Execute_HasListing(t *testing.T) {
	farms, cycles, owner, c := newCycleFixture(t, domain.CycleStatusPlanned)
	listings := &fakeListings{byCycle: map[bson.ObjectID]*domain.Listing{
		c.ID: {ID: bson.NewObjectID(), CycleID: &c.ID},
	}}
	uc := &DeleteCycle{Farms: farms, Cycles: cycles, Listings: listings}

	err := uc.Execute(context.Background(), c.ID, owner)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("Execute() error = %v, want domain.ErrConflict", err)
	}
	if _, err := cycles.FindByID(context.Background(), c.ID); err != nil {
		t.Error("cycle was deleted despite having a listing")
	}
}

func TestDeleteCycle_Execute_NotFarmOwner(t *testing.T) {
	farms, cycles, _, c := newCycleFixture(t, domain.CycleStatusPlanned)
	listings := &fakeListings{byCycle: map[bson.ObjectID]*domain.Listing{}}
	uc := &DeleteCycle{Farms: farms, Cycles: cycles, Listings: listings}

	err := uc.Execute(context.Background(), c.ID, bson.NewObjectID())
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Execute() error = %v, want domain.ErrForbidden", err)
	}
}
