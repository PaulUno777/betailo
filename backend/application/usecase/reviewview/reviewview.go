// Package reviewview decorates reviews for display, shared by the
// reviews page and the listing page's top reviews.
package reviewview

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Item is a review as shown to buyers: the review, who wrote it (public
// name only), which reservation it verifies, and whether the viewer
// already found it helpful.
type Item struct {
	Review         *domain.Review
	AuthorName     string
	ReservationRef string
	VotedByViewer  bool
	// ByViewer is true on the viewer's own review: they can't vote on it.
	ByViewer bool
}

// Sources is the read access Build needs.
type Sources struct {
	Users        ports.UserRepository
	Reservations ports.ReservationRepository
	Store        ports.ReviewStore // optional: without it, no viewer votes
}

// Build decorates reviews for viewerID (zero when anonymous). Names and
// references are decoration: a missing user or reservation never hides
// the review itself.
func Build(ctx context.Context, src Sources, reviews []*domain.Review, viewerID bson.ObjectID) ([]Item, error) {
	voted := map[bson.ObjectID]bool{}
	if src.Store != nil && !viewerID.IsZero() {
		ids := make([]bson.ObjectID, 0, len(reviews))
		for _, r := range reviews {
			ids = append(ids, r.ID)
		}
		v, err := src.Store.VotedBy(ctx, viewerID, ids)
		if err != nil {
			return nil, err
		}
		voted = v
	}
	items := make([]Item, 0, len(reviews))
	for _, r := range reviews {
		it := Item{Review: r, VotedByViewer: voted[r.ID], ByViewer: !viewerID.IsZero() && r.AuthorID == viewerID}
		if u, err := src.Users.FindByID(ctx, r.AuthorID); err == nil {
			it.AuthorName = domain.PublicName(u.DisplayName)
		}
		if res, err := src.Reservations.FindByID(ctx, r.ReservationID); err == nil {
			it.ReservationRef = res.Reference
		}
		items = append(items, it)
	}
	return items, nil
}
