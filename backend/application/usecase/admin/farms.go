package admin

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// FarmReviewItem is a farm as the back office sees it: with its owner.
type FarmReviewItem struct {
	Farm  *domain.Farm
	Owner *domain.User
}

// ListFarmsForReview backs the validation queue: farms in one status
// (pending by default at the HTTP layer), oldest submission first.
type ListFarmsForReview struct {
	Farms ports.FarmRepository
	Users ports.UserRepository
}

func (uc *ListFarmsForReview) Execute(ctx context.Context, status domain.FarmStatus) ([]FarmReviewItem, error) {
	farms, err := uc.Farms.ListByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	owners := map[bson.ObjectID]*domain.User{}
	out := make([]FarmReviewItem, 0, len(farms))
	for _, f := range farms {
		owner, seen := owners[f.OwnerID]
		if !seen {
			// An owner that can't be read doesn't hide the farm from
			// the queue; it shows without one.
			owner, _ = uc.Users.FindByID(ctx, f.OwnerID)
			owners[f.OwnerID] = owner
		}
		out = append(out, FarmReviewItem{Farm: f, Owner: owner})
	}
	return out, nil
}

// FarmReviewDetail is everything staff need to decide on one farm,
// proof included as short-lived download links.
type FarmReviewDetail struct {
	FarmReviewItem
	IDDocumentURL string
	PhotoURLs     []string
	// OwnerFarmCount is how many farms the owner has in total.
	OwnerFarmCount int
}

type GetFarmForReview struct {
	Farms   ports.FarmRepository
	Users   ports.UserRepository
	Storage ports.ObjectStorage
}

func (uc *GetFarmForReview) Execute(ctx context.Context, farmID bson.ObjectID) (*FarmReviewDetail, error) {
	f, err := uc.Farms.FindByID(ctx, farmID)
	if err != nil {
		return nil, err
	}
	owner, err := uc.Users.FindByID(ctx, f.OwnerID)
	if err != nil {
		return nil, err
	}
	owned, err := uc.Farms.ListByOwner(ctx, f.OwnerID)
	if err != nil {
		return nil, err
	}
	detail := &FarmReviewDetail{FarmReviewItem: FarmReviewItem{Farm: f, Owner: owner}, OwnerFarmCount: len(owned)}
	if key := f.Verification.IDDocumentKey; key != "" {
		if detail.IDDocumentURL, err = uc.Storage.PresignDownload(ctx, key); err != nil {
			return nil, err
		}
	}
	for _, key := range f.Verification.PhotoKeys {
		url, err := uc.Storage.PresignDownload(ctx, key)
		if err != nil {
			return nil, err
		}
		detail.PhotoURLs = append(detail.PhotoURLs, url)
	}
	return detail, nil
}
