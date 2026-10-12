package listing

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// isVerifiedSeller reports whether the account may put listings live.
// Anyone can open a farm and prepare cycles and draft listings; only a
// seller an admin has verified gets them onto the marketplace.
func isVerifiedSeller(ctx context.Context, users ports.UserRepository, ownerID bson.ObjectID) (bool, error) {
	u, err := users.FindByID(ctx, ownerID)
	if err != nil {
		return false, err
	}
	return u.IsVerified, nil
}

func errSellerNotVerified() error {
	return domain.Coded("farm.not_verified", fmt.Errorf("listing: the seller isn't verified yet: %w", domain.ErrForbidden))
}
