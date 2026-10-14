package mongodb

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/pauluno777/betailo/backend/schemas"
)

// namespaceExists is the MongoDB server error code returned by `create`
// when the collection is already there — the one case Bootstrap treats
// as success rather than failure.
const namespaceExists = 48

// Bootstrap creates every collection the application needs and attaches
// its $jsonSchema validator (see the schemas package), then builds the
// indexes the domain relies on. It is idempotent: safe to run against a
// database that already has some or all collections, which matters both
// for local dev (docker compose restarts) and a shared Atlas cluster —
// every call at startup (see internal/app) re-applies the same state
// rather than assuming a fresh database.
func Bootstrap(ctx context.Context, database *mongo.Database) error {
	for _, name := range schemas.Names {
		if err := ensureCollection(ctx, database, name); err != nil {
			return fmt.Errorf("mongodb: bootstrap %s: %w", name, err)
		}
	}
	if err := ensureIndexes(ctx, database); err != nil {
		return fmt.Errorf("mongodb: bootstrap indexes: %w", err)
	}
	return nil
}

// ensureCollection creates the named collection with its validator, or
// — if it already exists — updates the validator in place via collMod
// so a schema change in the schemas package takes effect on restart
// without a manual migration step.
func ensureCollection(ctx context.Context, database *mongo.Database, name string) error {
	schema, err := schemas.Load(name)
	if err != nil {
		return err
	}
	validator := bson.M{"$jsonSchema": schema}

	err = database.CreateCollection(ctx, name, options.CreateCollection().
		SetValidator(validator).
		SetValidationLevel("moderate"))
	if err == nil {
		return nil
	}

	var cmdErr mongo.CommandError
	if !errors.As(err, &cmdErr) || cmdErr.Code != namespaceExists {
		return fmt.Errorf("create: %w", err)
	}

	result := database.RunCommand(ctx, bson.D{
		{Key: "collMod", Value: name},
		{Key: "validator", Value: validator},
		{Key: "validationLevel", Value: "moderate"},
	})
	if result.Err() != nil {
		return fmt.Errorf("collMod: %w", result.Err())
	}
	return nil
}

// ensureIndexes creates the indexes the domain depends on for
// correctness (users.email uniqueness) or for the query patterns the
// ER model implies (each *Id reference gets an index on the owning
// side). CreateMany is idempotent: an index that already exists with
// the same keys/options is a no-op.
func ensureIndexes(ctx context.Context, database *mongo.Database) error {
	type spec struct {
		collection string
		models     []mongo.IndexModel
	}

	specs := []spec{
		{"users", []mongo.IndexModel{
			{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		}},
		{"refresh_tokens", []mongo.IndexModel{
			{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "userId", Value: 1}}},
			{Keys: bson.D{{Key: "familyId", Value: 1}}},
			// TTL: Mongo's background thread deletes a document once
			// expiresAt is in the past — no cleanup worker needed,
			// unlike the reservation expiry sweep in internal/worker.
			{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		}},
		{"farms", []mongo.IndexModel{
			{Keys: bson.D{{Key: "ownerId", Value: 1}}},
			// The back office's review queue.
			{Keys: bson.D{{Key: "status", Value: 1}, {Key: "verification.submittedAt", Value: 1}}},
		}},
		{"cycles", []mongo.IndexModel{
			{Keys: bson.D{{Key: "farmId", Value: 1}}},
		}},
		{"listings", []mongo.IndexModel{
			{Keys: bson.D{{Key: "cycleId", Value: 1}}},
		}},
		{"reservations", []mongo.IndexModel{
			{Keys: bson.D{{Key: "listingId", Value: 1}}},
			{Keys: bson.D{{Key: "buyerId", Value: 1}}},
			// The worker's expiry sweep: unpaid reservations past their
			// deadline. Sparse-friendly since most reservations never set
			// paymentDeadline (paid-at-creation ones don't have one).
			{Keys: bson.D{{Key: "paymentDeadline", Value: 1}}},
		}},
		{"reviews", []mongo.IndexModel{
			// One review per reservation.
			{Keys: bson.D{{Key: "reservationId", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "farmId", Value: 1}}},
		}},
		{"reports", []mongo.IndexModel{
			{Keys: bson.D{{Key: "targetId", Value: 1}, {Key: "createdAt", Value: -1}}},
			{Keys: bson.D{{Key: "status", Value: 1}}},
		}},
		{"forum_posts", []mongo.IndexModel{
			{Keys: bson.D{{Key: "authorId", Value: 1}}},
			{Keys: bson.D{{Key: "category", Value: 1}}},
		}},
		{"reservations", []mongo.IndexModel{
			{Keys: bson.D{{Key: "reference", Value: 1}}, Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"reference": bson.M{"$type": "string"}})},
		}},
		{"cycles", []mongo.IndexModel{
			{Keys: bson.D{{Key: "reference", Value: 1}}, Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"reference": bson.M{"$type": "string"}})},
		}},
		{"farms", []mongo.IndexModel{
			{Keys: bson.D{{Key: "reference", Value: 1}}, Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"reference": bson.M{"$type": "string"}})},
		}},
		{"admin_actions", []mongo.IndexModel{
			{Keys: bson.D{{Key: "createdAt", Value: -1}}},
		}},
		{"favorites", []mongo.IndexModel{
			// One save per (user, listing); also serves "my favorites"
			// (userId prefix) and the per-page favorited lookup.
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "listingId", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
		}},
		{"notifications", []mongo.IndexModel{
			{Keys: bson.D{{Key: "status", Value: 1}}},
			{Keys: bson.D{{Key: "userId", Value: 1}}},
			// The bell: a user's newest first, and the unread count.
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
		}},
	}

	for _, s := range specs {
		if _, err := database.Collection(s.collection).Indexes().CreateMany(ctx, s.models); err != nil {
			return fmt.Errorf("%s: %w", s.collection, err)
		}
	}
	return nil
}
