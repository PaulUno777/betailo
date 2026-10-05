package reservation

import (
	"context"
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/notify"
	"github.com/pauluno777/betailo/backend/domain"
)

// Parties resolves who is on each side of a reservation, for telling
// the other party what just happened. Buyers are on the reservation;
// the farmer is behind listing -> cycle -> farm.
type Parties struct {
	Listings ports.ListingStore
	Cycles   ports.CycleStore
	Farms    ports.FarmRepository
}

// farmerAndTitle returns the farm owner and the listing title.
func (p Parties) farmerAndTitle(ctx context.Context, res *domain.Reservation) (bson.ObjectID, string, error) {
	l, err := p.Listings.FindByID(ctx, res.ListingID)
	if err != nil {
		return bson.ObjectID{}, "", err
	}
	owner, err := farmOwner(ctx, p.Listings, p.Cycles, p.Farms, res.ListingID)
	if err != nil {
		return bson.ObjectID{}, "", err
	}
	return owner, l.Title, nil
}

// reservationParams are the values every reservation notification can
// interpolate.
func reservationParams(res *domain.Reservation, title string) map[string]string {
	return map[string]string{
		"ref":      res.Reference,
		"title":    title,
		"quantity": strconv.Itoa(res.Quantity),
		"amount":   strconv.FormatFloat(res.TotalAmount, 'f', 0, 64),
	}
}

// notifyFarmer tells the farmer behind res, best effort like notify.Send.
func (p Parties) notifyFarmer(ctx context.Context, outbox ports.NotificationRepository, clock ports.Clock, res *domain.Reservation, typ, fallback string) {
	if outbox == nil || p.Listings == nil {
		return
	}
	owner, title, err := p.farmerAndTitle(ctx, res)
	if err != nil {
		return
	}
	notify.Send(ctx, outbox, clock.Now(), notify.Notice{
		UserID: owner, Type: typ, Params: reservationParams(res, title),
		Link: domain.LinkFarmerReservations, Fallback: fmt.Sprintf(fallback, res.Reference),
	})
}

// notifyBuyer tells the buyer of res, best effort.
func (p Parties) notifyBuyer(ctx context.Context, outbox ports.NotificationRepository, clock ports.Clock, res *domain.Reservation, typ, fallback string, extra map[string]string) {
	if outbox == nil {
		return
	}
	title := ""
	if p.Listings != nil {
		if l, err := p.Listings.FindByID(ctx, res.ListingID); err == nil {
			title = l.Title
		}
	}
	params := reservationParams(res, title)
	for k, v := range extra {
		params[k] = v
	}
	notify.Send(ctx, outbox, clock.Now(), notify.Notice{
		UserID: res.BuyerID, Type: typ, Params: params,
		Link: domain.LinkBuyerReservations, Fallback: fmt.Sprintf(fallback, res.Reference),
	})
}
