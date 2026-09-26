package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ListingRepository is the read-only slice of listing persistence the
// reservation usecase needs: just enough to freeze a price and find the
// cycle to reserve against. Kept to the one method actually used —
// widen it only when a second real caller shows up (Interface
// Segregation Principle).
type ListingRepository interface {
	CycleAndPrice(ctx context.Context, listingID bson.ObjectID) (cycleID bson.ObjectID, unitPrice float64, err error)
}
