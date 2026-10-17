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

	// Unit is the pricing unit shown next to UnitPrice ("chick",
	// "trio", "sac"...) — free text set by the farmer, not an enum: the
	// marketplace spans too many species/units to close-list them.
	Unit   string
	Status ListingStatus

	// Photos holds object-storage keys, never URLs — signed URLs are
	// generated on demand via application/ports.ObjectStorage.
	Photos []string

	// Flagged is set true once the reporting threshold trips (see
	// application/usecase/report.CreateReport) and hides the listing
	// from the public marketplace until an admin resolves the reports
	// against it. Kept separate from Status so the publish/unpublish
	// state machine doesn't have to reason about moderation.
	Flagged bool

	// Breed ("Brahma herminé"), AgeDays and DeliveryAvailable describe
	// the animals as the sell wizard collects them. All optional. Age is
	// in days so day-old chicks can say so; the app shows it in days,
	// weeks, months or years.
	Breed             string
	AgeDays           int
	DeliveryAvailable bool
}
