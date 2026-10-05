package ports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// FavoriteRepository persists saved listings. Add and Remove are
// idempotent: saving twice or un-saving something never saved is not an
// error, so a double tap on the heart can never surface one.
type FavoriteRepository interface {
	Add(ctx context.Context, userID, listingID bson.ObjectID, at time.Time) error
	Remove(ctx context.Context, userID, listingID bson.ObjectID) error
	// ListListingIDs returns the user's saved listing IDs, most
	// recently saved first.
	ListListingIDs(ctx context.Context, userID bson.ObjectID) ([]bson.ObjectID, error)
	// FavoritedAmong reports which of listingIDs the user has saved, in
	// one query, for flagging a page of search results.
	FavoritedAmong(ctx context.Context, userID bson.ObjectID, listingIDs []bson.ObjectID) (map[bson.ObjectID]bool, error)
}
