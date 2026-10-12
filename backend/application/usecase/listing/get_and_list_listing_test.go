package listing

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestGetListing_Execute(t *testing.T) {
	listings := newFakeListings()
	l := &domain.Listing{Title: "Lot", Status: domain.ListingPublished}
	if err := listings.Create(context.Background(), l); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	uc := &GetListing{Listings: listings}
	got, err := uc.Execute(context.Background(), l.ID, bson.ObjectID{})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got.Title != "Lot" {
		t.Errorf("Title = %q, want %q", got.Title, "Lot")
	}

	_, err = uc.Execute(context.Background(), bson.NewObjectID(), bson.ObjectID{})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Execute() with unknown ID: error = %v, want domain.ErrNotFound", err)
	}
}

// Drafts, unpublished and flagged listings exist only for their owner;
// to anyone else they read as not found.
func TestGetListing_Execute_HiddenListingsOnlyForOwner(t *testing.T) {
	owner := bson.NewObjectID()
	farm := &domain.Farm{ID: bson.NewObjectID(), OwnerID: owner}
	cycle := &domain.Cycle{ID: bson.NewObjectID(), FarmID: farm.ID}
	listings := newFakeListings()
	uc := &GetListing{Listings: listings, Cycles: newFakeCycles(cycle), Farms: newFakeFarms(farm)}

	for _, l := range []*domain.Listing{
		{CycleID: &cycle.ID, Title: "Brouillon", Status: domain.ListingDraft},
		{CycleID: &cycle.ID, Title: "Retirée", Status: domain.ListingUnpublished},
		{CycleID: &cycle.ID, Title: "Signalée", Status: domain.ListingPublished, Flagged: true},
	} {
		if err := listings.Create(context.Background(), l); err != nil {
			t.Fatal(err)
		}
		if _, err := uc.Execute(context.Background(), l.ID, bson.ObjectID{}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s, anonymous: error = %v, want ErrNotFound", l.Title, err)
		}
		if _, err := uc.Execute(context.Background(), l.ID, bson.NewObjectID()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s, another user: error = %v, want ErrNotFound", l.Title, err)
		}
		if _, err := uc.Execute(context.Background(), l.ID, owner); err != nil {
			t.Errorf("%s, owner: error = %v, want nil", l.Title, err)
		}
	}
}
