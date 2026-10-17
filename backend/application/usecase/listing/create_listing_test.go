package listing

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func newListingFixture(t *testing.T) (*fakeFarms, *fakeCycles, bson.ObjectID, *domain.Cycle) {
	t.Helper()
	owner := bson.NewObjectID()
	f := &domain.Farm{ID: bson.NewObjectID(), OwnerID: owner, Name: "Ferme", Location: "Douala", Status: domain.FarmApproved}
	c := &domain.Cycle{ID: bson.NewObjectID(), FarmID: f.ID, Species: "chicks", Status: domain.CycleStatusPlanned, Capacity: 10, AvailableQuantity: 10}
	return newFakeFarms(f), newFakeCycles(c), owner, c
}

func TestCreateListing_Execute_Success(t *testing.T) {
	farms, cycles, owner, c := newListingFixture(t)
	listings := newFakeListings()
	uc := &CreateListing{Farms: farms, Cycles: cycles, Listings: listings}

	l, err := uc.Execute(context.Background(), CreateListingInput{
		CycleID: c.ID, OwnerID: owner, Title: "  Lot de poules  ", UnitPrice: 15.5,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if l.Title != "Lot de poules" {
		t.Errorf("Title = %q, want trimmed", l.Title)
	}
	if l.CycleID == nil || *l.CycleID != c.ID {
		t.Errorf("CycleID = %v, want %v", l.CycleID, c.ID)
	}
	if l.Status != domain.ListingPublished {
		t.Errorf("Status = %v, want %v", l.Status, domain.ListingPublished)
	}
	if l.ID.IsZero() {
		t.Error("ID is zero, want the repository to have assigned one")
	}
}

func TestCreateListing_Execute_NotFarmOwner(t *testing.T) {
	_, cycles, _, c := newListingFixture(t)
	farms := newFakeFarms(&domain.Farm{ID: c.FarmID, OwnerID: bson.NewObjectID(), Name: "Ferme", Location: "Douala"})
	uc := &CreateListing{Farms: farms, Cycles: cycles, Listings: newFakeListings()}

	_, err := uc.Execute(context.Background(), CreateListingInput{
		CycleID: c.ID, OwnerID: bson.NewObjectID(), Title: "Lot", UnitPrice: 10,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Execute() error = %v, want domain.ErrForbidden", err)
	}
}

func TestCreateListing_Execute_UnknownCycle(t *testing.T) {
	farms, _, owner, _ := newListingFixture(t)
	uc := &CreateListing{Farms: farms, Cycles: newFakeCycles(), Listings: newFakeListings()}

	_, err := uc.Execute(context.Background(), CreateListingInput{
		CycleID: bson.NewObjectID(), OwnerID: owner, Title: "Lot", UnitPrice: 10,
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Execute() error = %v, want domain.ErrNotFound", err)
	}
}

func TestCreateListing_Execute_Validation(t *testing.T) {
	tests := []struct {
		name  string
		title string
		price float64
	}{
		{"empty title", "", 10},
		{"blank title", "   ", 10},
		{"negative price", "Lot", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			farms, cycles, owner, c := newListingFixture(t)
			uc := &CreateListing{Farms: farms, Cycles: cycles, Listings: newFakeListings()}

			_, err := uc.Execute(context.Background(), CreateListingInput{
				CycleID: c.ID, OwnerID: owner, Title: tt.title, UnitPrice: tt.price,
			})
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("Execute() error = %v, want domain.ErrInvalid", err)
			}
		})
	}
}

// A farm staff haven't approved can prepare a listing, but it stays a
// draft: only an approved farm's listings reach the marketplace.
func TestCreateListing_Execute_UnapprovedFarmGetsADraft(t *testing.T) {
	farms, cycles, owner, c := newListingFixture(t)
	farms.byID[c.FarmID].Status = domain.FarmPending
	uc := &CreateListing{Farms: farms, Cycles: cycles, Listings: newFakeListings()}

	l, err := uc.Execute(context.Background(), CreateListingInput{CycleID: c.ID, OwnerID: owner, Title: "Lot", UnitPrice: 10})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if l.Status != domain.ListingDraft {
		t.Errorf("Status = %v, want %v", l.Status, domain.ListingDraft)
	}
}

func TestPublish_Execute_RequiresAnApprovedFarm(t *testing.T) {
	farms, cycles, owner, c := newListingFixture(t)
	listings := newFakeListings()
	draft := &domain.Listing{ID: bson.NewObjectID(), CycleID: &c.ID, Title: "Lot", UnitPrice: 10, Status: domain.ListingDraft, Photos: []string{"listings/a.jpg"}}
	listings.byID[draft.ID] = draft
	uc := &Publish{Farms: farms, Cycles: cycles, Listings: listings}

	for _, status := range []domain.FarmStatus{domain.FarmDraft, domain.FarmPending, domain.FarmRejected, domain.FarmSuspended} {
		farms.byID[c.FarmID].Status = status
		_, err := uc.Execute(context.Background(), draft.ID, owner)
		if code, _ := domain.CodeOf(err); !errors.Is(err, domain.ErrForbidden) || code != "farm.not_approved" {
			t.Fatalf("%s farm: Execute() error = %v, want forbidden farm.not_approved", status, err)
		}
		if draft.Status != domain.ListingDraft {
			t.Errorf("%s farm: Status = %v, want it left a draft", status, draft.Status)
		}
	}

	farms.byID[c.FarmID].Status = domain.FarmApproved
	l, err := uc.Execute(context.Background(), draft.ID, owner)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if l.Status != domain.ListingPublished {
		t.Errorf("Status = %v, want %v", l.Status, domain.ListingPublished)
	}
}
