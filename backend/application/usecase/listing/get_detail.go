package listing

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/farmstats"
	"github.com/pauluno777/betailo/backend/domain"
)

// detailTopReviews is how many recent reviews the listing page shows
// inline before "Voir les N avis".
const detailTopReviews = 2

// Detail is everything the listing page renders, assembled in one
// request: phones on slow networks shouldn't wait on a four-call
// waterfall (listing, then cycle, then farm and reviews).
type Detail struct {
	Listing  *domain.Listing
	Cycle    *domain.Cycle
	Timeline []domain.TimelineStep
	Farm     *farmstats.Profile

	ReviewSummary ports.ReviewSummary
	TopReviews    []*domain.Review

	// Viewer-specific, decided here rather than by the client.
	Favorited  bool
	IsOwner    bool
	CanReserve bool
}

// GetDetail builds a listing's Detail for a viewer (zero ID when
// anonymous). See visibleTo for who can see what.
type GetDetail struct {
	Listings     ports.ListingStore
	Cycles       ports.CycleStore
	Farms        ports.FarmRepository
	Users        ports.UserRepository
	Reviews      ports.ReviewRepository
	Reservations ports.ReservationRepository
	Favorites    ports.FavoriteRepository
	Clock        ports.Clock
}

func (uc *GetDetail) Execute(ctx context.Context, listingID, viewerID bson.ObjectID) (*Detail, error) {
	l, err := uc.Listings.FindByID(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if l.CycleID == nil {
		return nil, fmt.Errorf("listing: has no cycle: %w", domain.ErrNotFound)
	}
	c, err := uc.Cycles.FindByID(ctx, *l.CycleID)
	if err != nil {
		return nil, err
	}
	f, err := uc.Farms.FindByID(ctx, c.FarmID)
	if err != nil {
		return nil, err
	}
	isOwner := !viewerID.IsZero() && viewerID == f.OwnerID
	if err := visibleTo(l, isOwner); err != nil {
		return nil, err
	}

	profile, err := farmstats.BuildProfile(ctx, farmstats.ProfileSources{
		Sources: farmstats.Sources{Cycles: uc.Cycles, Listings: uc.Listings, Reservations: uc.Reservations},
		Users:   uc.Users, Reviews: uc.Reviews,
	}, f)
	if err != nil {
		return nil, err
	}
	summary, err := uc.Reviews.Summary(ctx, f.ID)
	if err != nil {
		return nil, err
	}
	reviews, err := uc.Reviews.ListByFarm(ctx, f.ID)
	if err != nil {
		return nil, err
	}
	// ObjectIDs embed creation time: newest first.
	slices.SortFunc(reviews, func(a, b *domain.Review) int { return strings.Compare(b.ID.Hex(), a.ID.Hex()) })
	if len(reviews) > detailTopReviews {
		reviews = reviews[:detailTopReviews]
	}

	favorited := false
	if !viewerID.IsZero() {
		saved, err := uc.Favorites.FavoritedAmong(ctx, viewerID, []bson.ObjectID{l.ID})
		if err != nil {
			return nil, err
		}
		favorited = saved[l.ID]
	}

	return &Detail{
		Listing:       l,
		Cycle:         c,
		Timeline:      domain.CycleTimeline(c, uc.Clock.Now()),
		Farm:          profile,
		ReviewSummary: summary,
		TopReviews:    reviews,
		Favorited:     favorited,
		IsOwner:       isOwner,
		CanReserve:    !isOwner && isPublic(l) && cycleOpenForReservation(c),
	}, nil
}

func isPublic(l *domain.Listing) bool {
	return l.Status == domain.ListingPublished && !l.Flagged
}

// visibleTo is the listing visibility rule: anyone can see a
// published, unflagged listing; drafts, unpublished and flagged ones
// exist only for their farm's owner (who edits and previews them).
// Hidden listings read as not found rather than forbidden, so their
// existence doesn't leak.
func visibleTo(l *domain.Listing, isOwner bool) error {
	if isPublic(l) || isOwner {
		return nil
	}
	return fmt.Errorf("listing: not visible: %w", domain.ErrNotFound)
}

func cycleOpenForReservation(c *domain.Cycle) bool {
	return (c.Status == domain.CycleStatusPlanned || c.Status == domain.CycleStatusInProgress) && c.AvailableQuantity > 0
}
