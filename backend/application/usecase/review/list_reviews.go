package review

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ListByFarm backs a farm's public reviews section — the reviews
// themselves plus the aggregated summary (RatingSummary in the design).
type ListByFarm struct {
	Reviews ports.ReviewRepository
}

type ListByFarmOutput struct {
	Reviews []*domain.Review
	Summary ports.ReviewSummary
}

func (uc *ListByFarm) Execute(ctx context.Context, farmID bson.ObjectID) (*ListByFarmOutput, error) {
	reviews, err := uc.Reviews.ListByFarm(ctx, farmID)
	if err != nil {
		return nil, err
	}
	summary, err := uc.Reviews.Summary(ctx, farmID)
	if err != nil {
		return nil, err
	}
	return &ListByFarmOutput{Reviews: reviews, Summary: summary}, nil
}
