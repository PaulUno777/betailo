package review

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/reviewview"
	"github.com/pauluno777/betailo/backend/domain"
)

const (
	defaultPageSize = 10
	maxPageSize     = 50
)

// ListByFarm is a farm's reviews page: summary (histogram + per-criteria
// averages) and one filtered page of reviews, newest first.
type ListByFarm struct {
	Reviews      ports.ReviewRepository
	Store        ports.ReviewStore
	Users        ports.UserRepository
	Reservations ports.ReservationRepository
}

type ListByFarmInput struct {
	FarmID   bson.ObjectID
	ViewerID bson.ObjectID // zero when anonymous
	Filter   ports.ReviewFilter
	Page     int
	PageSize int
}

type ListByFarmOutput struct {
	Items   []reviewview.Item
	Total   int
	Summary ports.ReviewSummary
}

func (uc *ListByFarm) Execute(ctx context.Context, in ListByFarmInput) (*ListByFarmOutput, error) {
	switch in.Filter {
	case ports.ReviewFilterAll, ports.ReviewFilterPhotos, ports.ReviewFilterFive, ports.ReviewFilterCritical:
	default:
		return nil, fmt.Errorf("review: unknown filter %q: %w", in.Filter, domain.ErrInvalid)
	}
	page, size := max(in.Page, 1), in.PageSize
	if size <= 0 {
		size = defaultPageSize
	}
	size = min(size, maxPageSize)

	reviews, total, err := uc.Store.ListPage(ctx, in.FarmID, in.Filter, page, size)
	if err != nil {
		return nil, err
	}
	summary, err := uc.Reviews.Summary(ctx, in.FarmID)
	if err != nil {
		return nil, err
	}
	items, err := reviewview.Build(ctx, reviewview.Sources{Users: uc.Users, Reservations: uc.Reservations, Store: uc.Store}, reviews, in.ViewerID)
	if err != nil {
		return nil, err
	}
	return &ListByFarmOutput{Items: items, Total: total, Summary: summary}, nil
}
