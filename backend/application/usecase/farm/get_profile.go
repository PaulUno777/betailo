package farm

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/farmstats"
)

// GetProfile builds a farm's public profile (see farmstats.Profile).
type GetProfile struct {
	Farms        ports.FarmRepository
	Users        ports.UserRepository
	Reviews      ports.ReviewRepository
	Cycles       ports.CycleStore
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
}

// Execute returns domain.ErrNotFound if the farm doesn't exist.
func (uc *GetProfile) Execute(ctx context.Context, farmID bson.ObjectID) (*farmstats.Profile, error) {
	f, err := uc.Farms.FindByID(ctx, farmID)
	if err != nil {
		return nil, err
	}
	return farmstats.BuildProfile(ctx, farmstats.ProfileSources{
		Sources: farmstats.Sources{Cycles: uc.Cycles, Listings: uc.Listings, Reservations: uc.Reservations},
		Users:   uc.Users, Reviews: uc.Reviews,
	}, f)
}
