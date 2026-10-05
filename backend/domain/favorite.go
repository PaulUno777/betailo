package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Favorite is a buyer's saved listing (the heart on a listing card).
// One per (user, listing) pair, enforced by a unique index.
type Favorite struct {
	UserID    bson.ObjectID
	ListingID bson.ObjectID
	CreatedAt time.Time
}
