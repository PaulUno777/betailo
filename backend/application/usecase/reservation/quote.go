package reservation

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Quote previews a reservation without making it: the checkout's price
// breakdown (with the service fee the backend owns) and the quantity
// bounds, refused for exactly the reasons the reservation itself would
// be refused.
type Quote struct {
	Listings ports.ListingStore
	Cycles   ports.CycleRepository
	Farms    ports.FarmRepository
}

type QuoteInput struct {
	ListingID bson.ObjectID
	BuyerID   bson.ObjectID
	Quantity  int
}

type QuoteOutput struct {
	Amounts
	Quantity    int
	FeeRate     float64
	Min         int // 1 when the cycle sets no minimum
	Max         int // the tighter of the cycle's maximum and what's left
	Available   int
	UnpaidHours int // how long an unpaid reservation holds its places
}

func (uc *Quote) Execute(ctx context.Context, in QuoteInput) (*QuoteOutput, error) {
	ok, err := checkReservable(ctx, uc.Listings, uc.Cycles, uc.Farms, in.ListingID, in.BuyerID, in.Quantity)
	if err != nil {
		return nil, err
	}
	available := ok.cycle.AvailableQuantity
	if in.Quantity > available {
		return nil, fmt.Errorf("reservation: only %d places left: %w", available, domain.ErrInsufficientCapacity)
	}
	minQ, maxQ := 1, available
	if ok.cycle.MinPerReservation > 0 {
		minQ = ok.cycle.MinPerReservation
	}
	if ok.cycle.MaxPerReservation > 0 && ok.cycle.MaxPerReservation < maxQ {
		maxQ = ok.cycle.MaxPerReservation
	}
	return &QuoteOutput{
		Amounts:     priceOf(ok.listing.UnitPrice, in.Quantity),
		Quantity:    in.Quantity,
		FeeRate:     ServiceFeeRate,
		Min:         minQ,
		Max:         maxQ,
		Available:   available,
		UnpaidHours: int(UnpaidReservationGrace.Hours()),
	}, nil
}
