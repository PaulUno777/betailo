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
	// PayNow quotes a paid reservation, which needs places left now. A
	// waiting-list one (false) only has to fit the cycle's capacity.
	PayNow bool
}

type QuoteOutput struct {
	Amounts
	Quantity  int
	FeeRate   float64
	Min       int // 1 when the cycle sets no minimum
	Max       int // the tighter of the cycle's maximum and what's left (capacity when not paying now)
	Available int
}

func (uc *Quote) Execute(ctx context.Context, in QuoteInput) (*QuoteOutput, error) {
	ok, err := checkReservable(ctx, uc.Listings, uc.Cycles, uc.Farms, in.ListingID, in.BuyerID, in.Quantity)
	if err != nil {
		return nil, err
	}
	available := ok.cycle.AvailableQuantity
	limit := ok.cycle.Capacity
	if in.PayNow {
		limit = available
	}
	if in.Quantity > limit {
		return nil, domain.CodedWith("reservation.not_enough_places", map[string]any{"available": limit}, fmt.Errorf("reservation: only %d places: %w", limit, domain.ErrInsufficientCapacity))
	}
	minQ, maxQ := 1, limit
	if ok.cycle.MinPerReservation > 0 {
		minQ = ok.cycle.MinPerReservation
	}
	if ok.cycle.MaxPerReservation > 0 && ok.cycle.MaxPerReservation < maxQ {
		maxQ = ok.cycle.MaxPerReservation
	}
	return &QuoteOutput{
		Amounts:   priceOf(ok.listing.UnitPrice, in.Quantity),
		Quantity:  in.Quantity,
		FeeRate:   ServiceFeeRate,
		Min:       minQ,
		Max:       maxQ,
		Available: available,
	}, nil
}
