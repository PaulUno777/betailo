package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CatalogChange is one operation waiting in the changeset, with who
// queued it and when.
type CatalogChange struct {
	ID bson.ObjectID
	Op CatalogOp
	By bson.ObjectID
	At time.Time
}

// CatalogChangeset is the reference data's pending changes, shared by
// the super admins: operations accumulate (each checked against the
// cumulative result when it is queued) and are applied, in order, to
// the version in force when published. Rev counts its writes: a write
// based on an older Rev is refused, so nobody overwrites another's
// changes unseen.
type CatalogChangeset struct {
	Rev       int
	Changes   []CatalogChange
	UpdatedAt time.Time
}
