package mongodb

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// listingRefDoc is the minimal "listings" document shape this
// repository reads — a persistence-only projection, not the full
// listing. It never leaves this file; domain.Listing carries no bson
// tags at all, by design (see domain/listing.go).
type listingRefDoc struct {
	CycleID   *bson.ObjectID `bson:"cycleId"`
	UnitPrice float64        `bson:"unitPrice"`
}

// ListingRepository implements ports.ListingRepository against the
// "listings" collection. It does not depend on any other module's
// repository, so reservation and listing stay decoupled at the module
// boundary (see "les services ne se parlent pas directement entre
// modules" in the architecture document).
type ListingRepository struct {
	Coll *mongo.Collection
}

var _ ports.ListingRepository = (*ListingRepository)(nil)

// CycleAndPrice implements ports.ListingRepository.
func (r *ListingRepository) CycleAndPrice(ctx context.Context, listingID bson.ObjectID) (bson.ObjectID, float64, error) {
	var doc listingRefDoc
	if err := r.Coll.FindOne(ctx, bson.M{"_id": listingID}).Decode(&doc); err != nil {
		return bson.ObjectID{}, 0, err
	}
	if doc.CycleID == nil {
		return bson.ObjectID{}, 0, errors.New("mongodb: listing has no cycle")
	}
	return *doc.CycleID, doc.UnitPrice, nil
}
