package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const verifyExistingSellersMigration = "verify-existing-sellers"

// VerifyExistingSellers marks as verified every seller who already had
// a published listing when publishing started to require a verified
// seller. Their listings were live under the old rule; without this
// they would stay live while their owner could no longer publish.
//
// Runs once: a marker in the `migrations` collection records it, so a
// later startup can never verify anyone on an admin's behalf. The
// marker is written last, so a crash midway re-runs the whole update,
// which is safe to repeat. Returns the sellers it verified, whose trust
// score the caller recomputes.
func VerifyExistingSellers(ctx context.Context, database *mongo.Database) ([]bson.ObjectID, error) {
	migrations := database.Collection("migrations")
	err := migrations.FindOne(ctx, bson.M{"_id": verifyExistingSellersMigration}).Err()
	if err == nil {
		return nil, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	var cycleIDs, farmIDs, ownerIDs []bson.ObjectID
	if err := database.Collection("listings").Distinct(ctx, "cycleId", bson.M{"status": "published"}).Decode(&cycleIDs); err != nil {
		return nil, err
	}
	if err := database.Collection("cycles").Distinct(ctx, "farmId", bson.M{"_id": bson.M{"$in": orEmpty(cycleIDs)}}).Decode(&farmIDs); err != nil {
		return nil, err
	}
	if err := database.Collection("farms").Distinct(ctx, "ownerId", bson.M{"_id": bson.M{"$in": orEmpty(farmIDs)}}).Decode(&ownerIDs); err != nil {
		return nil, err
	}

	users := database.Collection("users")
	unverified := bson.M{"_id": bson.M{"$in": orEmpty(ownerIDs)}, "isVerified": bson.M{"$ne": true}}
	var verified []bson.ObjectID
	if err := users.Distinct(ctx, "_id", unverified).Decode(&verified); err != nil {
		return nil, err
	}
	if len(verified) > 0 {
		if _, err := users.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": verified}}, bson.M{"$set": bson.M{"isVerified": true}}); err != nil {
			return nil, err
		}
	}
	if _, err := migrations.InsertOne(ctx, bson.M{"_id": verifyExistingSellersMigration, "ranAt": time.Now().UTC(), "verified": len(verified)}); err != nil {
		return nil, err
	}
	return verified, nil
}

// orEmpty keeps `$in` valid: it rejects null, which a nil slice encodes as.
func orEmpty(ids []bson.ObjectID) []bson.ObjectID {
	if ids == nil {
		return []bson.ObjectID{}
	}
	return ids
}
