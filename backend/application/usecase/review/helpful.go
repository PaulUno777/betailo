package review

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// MarkHelpful records the viewer finding a review useful ("Utile · 12").
// Idempotent per user; nobody can upvote their own review.
type MarkHelpful struct {
	Store ports.ReviewStore
}

// Execute returns the review's new helpful count.
func (uc *MarkHelpful) Execute(ctx context.Context, reviewID, userID bson.ObjectID) (int, error) {
	rv, err := uc.Store.FindByID(ctx, reviewID)
	if err != nil {
		return 0, err
	}
	if rv.AuthorID == userID {
		return 0, domain.Coded("review.own_review", fmt.Errorf("review: you can't mark your own review helpful: %w", domain.ErrForbidden))
	}
	added, err := uc.Store.AddHelpfulVote(ctx, reviewID, userID)
	if err != nil {
		return 0, err
	}
	if added {
		return rv.HelpfulCount + 1, nil
	}
	return rv.HelpfulCount, nil
}
