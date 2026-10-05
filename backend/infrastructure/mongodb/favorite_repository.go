package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/pauluno777/betailo/backend/application/ports"
)

type favoriteDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	UserID    bson.ObjectID `bson:"userId"`
	ListingID bson.ObjectID `bson:"listingId"`
	CreatedAt time.Time     `bson:"createdAt"`
}

// FavoriteRepository implements ports.FavoriteRepository against the
// "favorites" collection (unique on userId+listingId, see Bootstrap).
type FavoriteRepository struct {
	Coll *mongo.Collection
}

var _ ports.FavoriteRepository = (*FavoriteRepository)(nil)

// Add implements ports.FavoriteRepository as an upsert, so saving an
// already-saved listing keeps its original CreatedAt and order.
func (r *FavoriteRepository) Add(ctx context.Context, userID, listingID bson.ObjectID, at time.Time) error {
	_, err := r.Coll.UpdateOne(ctx,
		bson.M{"userId": userID, "listingId": listingID},
		bson.M{"$setOnInsert": favoriteDoc{UserID: userID, ListingID: listingID, CreatedAt: at}},
		options.UpdateOne().SetUpsert(true),
	)
	if mongo.IsDuplicateKeyError(err) {
		// Two concurrent upserts raced on the unique index; the other
		// one already saved it, which is the outcome we wanted.
		return nil
	}
	return err
}

// Remove implements ports.FavoriteRepository.
func (r *FavoriteRepository) Remove(ctx context.Context, userID, listingID bson.ObjectID) error {
	_, err := r.Coll.DeleteOne(ctx, bson.M{"userId": userID, "listingId": listingID})
	return err
}

// ListListingIDs implements ports.FavoriteRepository.
func (r *FavoriteRepository) ListListingIDs(ctx context.Context, userID bson.ObjectID) ([]bson.ObjectID, error) {
	cur, err := r.Coll.Find(ctx, bson.M{"userId": userID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetProjection(bson.M{"listingId": 1}))
	if err != nil {
		return nil, err
	}
	var docs []favoriteDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	ids := make([]bson.ObjectID, 0, len(docs))
	for _, d := range docs {
		ids = append(ids, d.ListingID)
	}
	return ids, nil
}

// FavoritedAmong implements ports.FavoriteRepository.
func (r *FavoriteRepository) FavoritedAmong(ctx context.Context, userID bson.ObjectID, listingIDs []bson.ObjectID) (map[bson.ObjectID]bool, error) {
	out := make(map[bson.ObjectID]bool, len(listingIDs))
	if len(listingIDs) == 0 {
		return out, nil
	}
	cur, err := r.Coll.Find(ctx, bson.M{"userId": userID, "listingId": bson.M{"$in": listingIDs}},
		options.Find().SetProjection(bson.M{"listingId": 1}))
	if err != nil {
		return nil, err
	}
	var docs []favoriteDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	for _, d := range docs {
		out[d.ListingID] = true
	}
	return out, nil
}
