package reservation

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// CloseUnpaidForCycle empties a closed cycle's waiting list: when a
// cycle ends (done) or is cancelled, every reservation on it that was
// never paid is cancelled and its buyer told. Unpaid reservations hold
// no places and have no deadline, so this is the only thing that closes
// them besides the buyer cancelling or paying.
type CloseUnpaidForCycle struct {
	Listings      ports.ListingStore
	Reservations  ports.ReservationRepository
	Notifications ports.NotificationRepository
	Clock         ports.Clock
}

// CloseUnpaid returns how many reservations it cancelled. It is safe to
// call again on the same cycle: only still-pending ones are touched.
func (uc *CloseUnpaidForCycle) CloseUnpaid(ctx context.Context, cycleID bson.ObjectID) (int, error) {
	l, err := uc.Listings.FindByCycleID(ctx, cycleID)
	if errors.Is(err, domain.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	all, err := uc.Reservations.ListByListingIDs(ctx, []bson.ObjectID{l.ID})
	if err != nil {
		return 0, err
	}
	closed := 0
	for _, res := range all {
		if res.Status != domain.ReservationPending || (res.Payment != nil && res.Payment.Status != domain.PaymentPending) {
			continue
		}
		res.Status = domain.ReservationCancelled
		if res.Payment != nil {
			res.Payment.Status = domain.PaymentFailed
		}
		if err := uc.Reservations.Update(ctx, res); err != nil {
			return closed, err
		}
		closed++
		Parties{}.notifyBuyer(ctx, uc.Notifications, uc.Clock, res, domain.NotifReservationExpired,
			"Le lot est clos : votre réservation %s non payée a été annulée.", map[string]string{"listing": l.Title})
	}
	return closed, nil
}
