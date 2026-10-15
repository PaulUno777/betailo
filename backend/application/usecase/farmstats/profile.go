package farmstats

import (
	"context"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Profile is a farm as a buyer sees it: the farm itself plus the trust
// signals the design shows next to it ("Ferme des Collines · 47 ventes
// · depuis 2024 · 92 CONFIANCE"). All are computed here, server-side,
// so no client ever stitches them together or recomputes them.
type Profile struct {
	Farm *domain.Farm

	TrustScore    float64
	Verified      bool
	SalesCount    int
	MemberSince   time.Time
	ReviewAverage float64
	ReviewCount   int
	// OwnerName is the farmer's public name and OwnerAvatarKey their
	// profile photo, shown next to the farm.
	OwnerName      string
	OwnerAvatarKey string
}

// ProfileSources is the read access BuildProfile needs.
type ProfileSources struct {
	Sources
	Users   ports.UserRepository
	Reviews ports.ReviewRepository
}

// BuildProfile assembles f's public Profile.
func BuildProfile(ctx context.Context, src ProfileSources, f *domain.Farm) (*Profile, error) {
	owner, err := src.Users.FindByID(ctx, f.OwnerID)
	if err != nil {
		return nil, err
	}
	reviews, err := src.Reviews.Summary(ctx, f.ID)
	if err != nil {
		return nil, err
	}
	sales, err := ReleasedSales(ctx, src.Sources, f.ID)
	if err != nil {
		return nil, err
	}
	return &Profile{
		Farm:           f,
		TrustScore:     owner.TrustScore,
		Verified:       f.Status.CanSell() && !f.Verification.AutoApproved,
		SalesCount:     sales,
		MemberSince:    owner.CreatedAt,
		ReviewAverage:  reviews.Average,
		ReviewCount:    reviews.Count,
		OwnerName:      domain.PublicName(owner.DisplayName),
		OwnerAvatarKey: owner.AvatarKey,
	}, nil
}
