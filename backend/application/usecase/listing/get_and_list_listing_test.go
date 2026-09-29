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
	got, err := uc.Execute(context.Background(), l.ID)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got.Title != "Lot" {
		t.Errorf("Title = %q, want %q", got.Title, "Lot")
	}

	_, err = uc.Execute(context.Background(), bson.NewObjectID())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Execute() with unknown ID: error = %v, want domain.ErrNotFound", err)
	}
}

func TestListPublishedListings_Execute(t *testing.T) {
	listings := newFakeListings()
	for _, l := range []*domain.Listing{
		{Title: "Published 1", Status: domain.ListingPublished},
		{Title: "Published 2", Status: domain.ListingPublished},
		{Title: "Unpublished", Status: domain.ListingUnpublished},
	} {
		if err := listings.Create(context.Background(), l); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	uc := &ListPublishedListings{Listings: listings}
	got, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d listings, want 2 (only published)", len(got))
	}
}
