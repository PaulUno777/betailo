package reservation

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// farmOwner resolves listing -> cycle -> farm and returns the farm's
// owner ID — the chain every "is this farmer allowed to act on this
// reservation" check needs, since a reservation only references a
// listing directly.
func farmOwner(ctx context.Context, listings ports.ListingStore, cycles ports.CycleStore, farms ports.FarmRepository, listingID bson.ObjectID) (ownerID bson.ObjectID, err error) {
	l, err := listings.FindByID(ctx, listingID)
	if err != nil {
		return bson.ObjectID{}, err
	}
	if l.CycleID == nil {
		return bson.ObjectID{}, fmt.Errorf("reservation: listing has no cycle: %w", domain.ErrInvalid)
	}
	c, err := cycles.FindByID(ctx, *l.CycleID)
	if err != nil {
		return bson.ObjectID{}, err
	}
	f, err := farms.FindByID(ctx, c.FarmID)
	if err != nil {
		return bson.ObjectID{}, err
	}
	return f.OwnerID, nil
}
