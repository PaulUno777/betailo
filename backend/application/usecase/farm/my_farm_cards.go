package farm

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/farmstats"
	"github.com/pauluno777/betailo/backend/domain"
)

// FarmCard is one farm as its owner's "Mes fermes" page shows it: the
// farm, a photo of it, and the figures that say how it is doing.
type FarmCard struct {
	Farm *domain.Farm
	// CoverKey is the first photo the owner sent with the farm's proof.
	// Private: only the owner and staff ever get a URL for it.
	CoverKey      string
	ActiveCycles  int
	SalesCount    int
	TrustScore    float64
	ReviewAverage float64
	ReviewCount   int
}

// MyFarmCards lists the caller's farms with those figures.
type MyFarmCards struct {
	Farms        ports.FarmRepository
	Users        ports.UserRepository
	Reviews      ports.ReviewRepository
	Cycles       ports.CycleStore
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
}

func (uc *MyFarmCards) Execute(ctx context.Context, ownerID bson.ObjectID) ([]FarmCard, error) {
	farms, err := uc.Farms.ListByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	src := farmstats.ProfileSources{
		Sources: farmstats.Sources{Cycles: uc.Cycles, Listings: uc.Listings, Reservations: uc.Reservations},
		Users:   uc.Users, Reviews: uc.Reviews,
	}
	cards := make([]FarmCard, 0, len(farms))
	for _, f := range farms {
		p, err := farmstats.BuildProfile(ctx, src, f)
		if err != nil {
			return nil, err
		}
		cycles, err := uc.Cycles.ListByFarm(ctx, f.ID)
		if err != nil {
			return nil, err
		}
		active := 0
		for _, c := range cycles {
			if c.Status == domain.CycleStatusPlanned || c.Status == domain.CycleStatusInProgress {
				active++
			}
		}
		card := FarmCard{
			Farm: f, ActiveCycles: active, SalesCount: p.SalesCount, TrustScore: p.TrustScore,
			ReviewAverage: p.ReviewAverage, ReviewCount: p.ReviewCount,
		}
		if len(f.Verification.PhotoKeys) > 0 {
			card.CoverKey = f.Verification.PhotoKeys[0]
		}
		cards = append(cards, card)
	}
	return cards, nil
}
