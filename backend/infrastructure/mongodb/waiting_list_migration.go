package mongodb

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/domain"
)

// MigrateUnpaidToWaitingList converts reservations made under the old
// rule (an unpaid reservation held its places for 24h, marked by a
// paymentDeadline) to the waiting list, where unpaid reservations hold
// nothing: each gives its places back to its cycle and loses its
// deadline.
//
// Idempotent and safe on every startup: a reservation is claimed by
// atomically removing its deadline before its places are returned, so
// no reservation can return places twice. A crash between the two
// steps under-counts places (never over-sells), which a farmer can
// correct by editing the cycle's capacity.
func MigrateUnpaidToWaitingList(ctx context.Context, database *mongo.Database) (int, error) {
	reservations := database.Collection("reservations")
	listings := database.Collection("listings")
	cycles := database.Collection("cycles")

	migrated := 0
	for {
		var res struct {
			ID        bson.ObjectID `bson:"_id"`
			ListingID bson.ObjectID `bson:"listingId"`
			Quantity  int           `bson:"quantity"`
		}
		err := reservations.FindOneAndUpdate(ctx,
			bson.M{"status": domain.ReservationPending, "paymentDeadline": bson.M{"$exists": true}},
			bson.M{"$unset": bson.M{"paymentDeadline": ""}},
		).Decode(&res)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return migrated, nil
		}
		if err != nil {
			return migrated, err
		}

		var listing struct {
			CycleID *bson.ObjectID `bson:"cycleId"`
		}
		if err := listings.FindOne(ctx, bson.M{"_id": res.ListingID}).Decode(&listing); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				continue
			}
			return migrated, fmt.Errorf("waiting list migration: listing of %s: %w", res.ID.Hex(), err)
		}
		if listing.CycleID != nil {
			if _, err := cycles.UpdateOne(ctx, bson.M{"_id": *listing.CycleID},
				bson.M{"$inc": bson.M{"availableQuantity": res.Quantity}}); err != nil {
				return migrated, fmt.Errorf("waiting list migration: cycle of %s: %w", res.ID.Hex(), err)
			}
		}
		migrated++
	}
}
