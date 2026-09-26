package domain

import "go.mongodb.org/mongo-driver/v2/bson"

// ListingStatus is a Listing's publication state. Values match
// schemas/listings.json's $jsonSchema enum exactly — do not change
// these string literals without updating that file too.
type ListingStatus string

const (
	ListingDraft       ListingStatus = "draft"
	ListingPublished   ListingStatus = "published"
	ListingUnpublished ListingStatus = "unpublished"
)

// Listing manages a marketplace listing, always attached to a Cycle.
//
// CycleID's Go type is a pointer to mirror the schema's technically
// optional reference, but application/usecase/listing.CreateListing
// never creates one without a cycle — see schemas/listings.json's
// comment on this field and ADR-0001.
type Listing struct {
	ID          bson.ObjectID
	CycleID     *bson.ObjectID
	Title       string
	Description string
	UnitPrice   float64
	Status      ListingStatus

	// Photos holds object-storage keys, never URLs — signed URLs are
	// generated on demand via application/ports.ObjectStorage.
	Photos []string
}
