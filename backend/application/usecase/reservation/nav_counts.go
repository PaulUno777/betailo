package reservation

import (
	"context"
	"slices"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// NavCounts is the numbers on the navigation badges: how many
// reservations are waiting on this user right now, as a buyer (pay,
// confirm delivery) and, when they own a farm, as a seller (mark
// delivered). "Waiting on" is read from the same actionsFor rules the
// reservation cards render their buttons from, so a badge never counts
// something the user can't act on.
type NavCounts struct {
	Farms   ports.FarmRepository
	Buyer   *ListMyReservations
	ForFarm *ListReservationsByFarm
	// Cycles, when set, counts the seller's active cycles.
	Cycles ports.CycleStore
}

type NavCountsResult struct {
	Buyer  int
	Seller int
	// Cycles is how many of the seller's cycles are planned or running.
	Cycles int
}

// buyerDue and sellerDue are the actions that move a reservation
// forward; cancel, dispute, review and contact are optional.
var (
	buyerDue  = []Action{ActionPay, ActionConfirm}
	sellerDue = []Action{ActionMarkDelivered}
)

func (uc *NavCounts) Execute(ctx context.Context, userID bson.ObjectID) (*NavCountsResult, error) {
	mine, err := uc.Buyer.Execute(ctx, userID, domain.GroupActive)
	if err != nil {
		return nil, err
	}
	out := &NavCountsResult{Buyer: countDue(mine.Items, buyerDue)}

	farms, err := uc.Farms.ListByOwner(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, f := range farms {
		sold, err := uc.ForFarm.Execute(ctx, ListReservationsByFarmInput{FarmID: f.ID, OwnerID: userID, Group: domain.GroupActive})
		if err != nil {
			return nil, err
		}
		out.Seller += countDue(sold.Items, sellerDue)
		if uc.Cycles == nil {
			continue
		}
		cycles, err := uc.Cycles.ListByFarm(ctx, f.ID)
		if err != nil {
			return nil, err
		}
		for _, c := range cycles {
			if c.Status == domain.CycleStatusPlanned || c.Status == domain.CycleStatusInProgress {
				out.Cycles++
			}
		}
	}
	return out, nil
}

func countDue(views []View, due []Action) int {
	n := 0
	for _, v := range views {
		if slices.ContainsFunc(v.Actions, func(a Action) bool { return slices.Contains(due, a) }) {
			n++
		}
	}
	return n
}
