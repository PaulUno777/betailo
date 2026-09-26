package domain

import "go.mongodb.org/mongo-driver/v2/bson"

// Listing manages a marketplace listing, always attached to a Cycle.
//
// CycleID is an optional reference: a Cycle can exist before it is
// published (no Listing yet). Unlike a relational schema, nothing here
// technically prevents an orphaned Listing — that consistency is
// guaranteed by the usecase layer, not by the schema.
type Listing struct {
	ID          bson.ObjectID
	CycleID     *bson.ObjectID
	Title       string
	Description string
	UnitPrice   float64
	Status      string

	// Photos holds object-storage keys, never URLs — signed URLs are
	// generated on demand via application/ports.ObjectStorage.
	Photos []string
}
