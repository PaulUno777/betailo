package listing

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func newPublishedListingFixture(t *testing.T) (*fakeFarms, *fakeCycles, *fakeListings, bson.ObjectID, *domain.Listing) {
	t.Helper()
	farms, cycles, owner, c := newListingFixture(t)
	listings := newFakeListings()
	l := &domain.Listing{ID: bson.NewObjectID(), CycleID: &c.ID, Title: "Lot", UnitPrice: 10, Status: domain.ListingPublished}
	if err := listings.Create(context.Background(), l); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return farms, cycles, listings, owner, l
}

func TestUpdateListing_Execute_Success(t *testing.T) {
	farms, cycles, listings, owner, l := newPublishedListingFixture(t)
	uc := &UpdateListing{Farms: farms, Cycles: cycles, Listings: listings}

	got, err := uc.Execute(context.Background(), UpdateListingInput{
		ListingID: l.ID, OwnerID: owner, Title: "New title", UnitPrice: 20,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got.Title != "New title" || got.UnitPrice != 20 {
		t.Errorf("got = %+v, want updated fields", got)
	}
}

func TestUpdateListing_Execute_SellerDetails(t *testing.T) {
	farms, cycles, listings, owner, l := newPublishedListingFixture(t)
	l.Breed, l.AgeDays = "Brahma", 6
	uc := &UpdateListing{Farms: farms, Cycles: cycles, Listings: listings}

	// Omitted details stay as they were.
	got, err := uc.Execute(context.Background(), UpdateListingInput{ListingID: l.ID, OwnerID: owner, Title: "Lot", UnitPrice: 10})
	if err != nil || got.Breed != "Brahma" || got.AgeDays != 6 {
		t.Fatalf("Execute() = %+v, %v; want details kept", got, err)
	}

	breed, age, delivery := " Goliath ", 8, true
	got, err = uc.Execute(context.Background(), UpdateListingInput{
		ListingID: l.ID, OwnerID: owner, Title: "Lot", UnitPrice: 10, Breed: &breed, AgeDays: &age, DeliveryAvailable: &delivery,
	})
	if err != nil || got.Breed != "Goliath" || got.AgeDays != 8 || !got.DeliveryAvailable {
		t.Fatalf("Execute() = %+v, %v; want details updated", got, err)
	}

	bad := -1
	if _, err := uc.Execute(context.Background(), UpdateListingInput{ListingID: l.ID, OwnerID: owner, Title: "Lot", UnitPrice: 10, AgeDays: &bad}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("negative age: error = %v, want domain.ErrInvalid", err)
	}
}

func TestUpdateListing_Execute_NotOwner(t *testing.T) {
	farms, cycles, listings, _, l := newPublishedListingFixture(t)
	uc := &UpdateListing{Farms: farms, Cycles: cycles, Listings: listings}

	_, err := uc.Execute(context.Background(), UpdateListingInput{
		ListingID: l.ID, OwnerID: bson.NewObjectID(), Title: "New", UnitPrice: 20,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Execute() error = %v, want domain.ErrForbidden", err)
	}
}

func TestUnpublishListing_Execute_Success(t *testing.T) {
	farms, cycles, listings, owner, l := newPublishedListingFixture(t)
	uc := &UnpublishListing{Farms: farms, Cycles: cycles, Listings: listings}

	if err := uc.Execute(context.Background(), l.ID, owner); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	got, err := listings.FindByID(context.Background(), l.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if got.Status != domain.ListingUnpublished {
		t.Errorf("Status = %v, want %v", got.Status, domain.ListingUnpublished)
	}
}

func TestUnpublishListing_Execute_NotOwner(t *testing.T) {
	farms, cycles, listings, _, l := newPublishedListingFixture(t)
	uc := &UnpublishListing{Farms: farms, Cycles: cycles, Listings: listings}

	err := uc.Execute(context.Background(), l.ID, bson.NewObjectID())
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Execute() error = %v, want domain.ErrForbidden", err)
	}
}
