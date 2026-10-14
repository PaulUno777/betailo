package admin

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// FarmReviewItem is a farm as the back office sees it: with its owner.
type FarmReviewItem struct {
	Farm  *domain.Farm
	Owner *domain.User
	// OwnerFarmCount is how many farms the owner has in total: the
	// queue tells a first farm from a third one.
	OwnerFarmCount int
}

// FarmQueue is one tab of the validation page.
type FarmQueue string

const (
	// FarmQueueTodo: waiting for a decision, oldest submission first.
	FarmQueueTodo FarmQueue = "todo"
	// FarmQueueResubmitted: waiting again after a refusal.
	FarmQueueResubmitted FarmQueue = "resubmitted"
	// FarmQueueDone: decided, latest decision first.
	FarmQueueDone FarmQueue = "done"
)

// ListFarmsForReview backs the validation queue. Queue picks a tab;
// Status, used when Queue is empty, lists one raw status (empty: every
// farm), oldest submission first.
type ListFarmsForReview struct {
	Farms ports.FarmRepository
	Users ports.UserRepository
}

type ListFarmsInput struct {
	Queue  FarmQueue
	Status domain.FarmStatus
}

func (uc *ListFarmsForReview) Execute(ctx context.Context, in ListFarmsInput) ([]FarmReviewItem, error) {
	var farms []*domain.Farm
	switch in.Queue {
	case "":
		list, err := uc.Farms.ListByStatus(ctx, in.Status)
		if err != nil {
			return nil, err
		}
		farms = list
	case FarmQueueTodo, FarmQueueResubmitted:
		list, err := uc.Farms.ListByStatus(ctx, domain.FarmPending)
		if err != nil {
			return nil, err
		}
		for _, f := range list {
			if in.Queue == FarmQueueTodo || f.Verification.Submissions > 1 {
				farms = append(farms, f)
			}
		}
	case FarmQueueDone:
		for _, status := range []domain.FarmStatus{domain.FarmApproved, domain.FarmRejected, domain.FarmSuspended} {
			list, err := uc.Farms.ListByStatus(ctx, status)
			if err != nil {
				return nil, err
			}
			farms = append(farms, list...)
		}
		sort.SliceStable(farms, func(i, j int) bool {
			return decidedAt(farms[i]).After(decidedAt(farms[j]))
		})
	default:
		return nil, fmt.Errorf("admin: unknown farm queue %q: %w", in.Queue, domain.ErrInvalid)
	}

	counts, err := farmCounts(ctx, uc.Farms)
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
		out = append(out, FarmReviewItem{Farm: f, Owner: owner, OwnerFarmCount: counts[f.OwnerID]})
	}
	return out, nil
}

func decidedAt(f *domain.Farm) time.Time {
	if f.Verification.ReviewedAt != nil {
		return *f.Verification.ReviewedAt
	}
	return f.CreatedAt
}

// FarmQueueCounts is the size of the two tabs staff still have to
// work through.
type FarmQueueCounts struct {
	Todo        int
	Resubmitted int
}

// CountFarmQueues backs the tab counters.
type CountFarmQueues struct {
	Farms ports.FarmRepository
}

func (uc *CountFarmQueues) Execute(ctx context.Context) (FarmQueueCounts, error) {
	pending, err := uc.Farms.ListByStatus(ctx, domain.FarmPending)
	if err != nil {
		return FarmQueueCounts{}, err
	}
	out := FarmQueueCounts{Todo: len(pending)}
	for _, f := range pending {
		if f.Verification.Submissions > 1 {
			out.Resubmitted++
		}
	}
	return out, nil
}

// FarmReviewDetail is everything staff need to decide on one farm,
// proof included as short-lived download links.
type FarmReviewDetail struct {
	FarmReviewItem
	IDDocumentURL     string
	IDDocumentBackURL string
	PhotoURLs         []string
	// OwnerFarmLimit is how many farms the owner may have.
	OwnerFarmLimit int
	// OwnerPurchases counts the owner's reservations as a buyer, and
	// OwnerReports the reports filed against their profile: what the
	// account did before asking to sell.
	OwnerPurchases int
	OwnerReports   int
	// ReviewerName is who decided, once someone did.
	ReviewerName string
	ReviewerRole domain.StaffRole
}

type GetFarmForReview struct {
	Farms        ports.FarmRepository
	Users        ports.UserRepository
	Reservations ports.ReservationRepository
	Reports      ports.ReportRepository
	Storage      ports.ObjectStorage
	// MaxFarms is the platform default (MAX_FARMS_PER_ACCOUNT).
	MaxFarms int
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
	detail := &FarmReviewDetail{
		FarmReviewItem: FarmReviewItem{Farm: f, Owner: owner, OwnerFarmCount: len(owned)},
		OwnerFarmLimit: owner.FarmLimit,
	}
	if detail.OwnerFarmLimit <= 0 {
		detail.OwnerFarmLimit = uc.MaxFarms
	}
	if detail.OwnerFarmLimit <= 0 {
		detail.OwnerFarmLimit = domain.DefaultMaxFarmsPerAccount
	}
	if uc.Reservations != nil {
		bought, err := uc.Reservations.ListByBuyer(ctx, owner.ID)
		if err != nil {
			return nil, err
		}
		detail.OwnerPurchases = len(bought)
	}
	if uc.Reports != nil {
		reported, err := uc.Reports.ListByTarget(ctx, owner.ID)
		if err != nil {
			return nil, err
		}
		detail.OwnerReports = len(reported)
	}
	if by := f.Verification.ReviewedBy; !by.IsZero() {
		if reviewer, err := uc.Users.FindByID(ctx, by); err == nil {
			detail.ReviewerName, detail.ReviewerRole = displayName(reviewer), reviewer.StaffRole
		}
	}

	if key := f.Verification.IDDocumentKey; key != "" {
		if detail.IDDocumentURL, err = uc.Storage.PresignDownload(ctx, key); err != nil {
			return nil, err
		}
	}
	if key := f.Verification.IDDocumentBackKey; key != "" {
		if detail.IDDocumentBackURL, err = uc.Storage.PresignDownload(ctx, key); err != nil {
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
