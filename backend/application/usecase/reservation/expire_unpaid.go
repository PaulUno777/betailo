package reservation

import (
	"context"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ExpireUnpaidReservations is internal/worker's sweep: every "réserver
// sans payer" reservation past its PaymentDeadline is cancelled and its
// quantity is restored to the cycle — the automatic status transition
// the architecture document's worker section names explicitly
// ("expirer une reservation non payee apres un delai").
type ExpireUnpaidReservations struct {
	Reservations  ports.ReservationRepository
	Listings      ports.ListingRepository
	Cycles        ports.CycleRepository
	Notifications ports.NotificationRepository
	Clock         ports.Clock
}

// Execute returns the number of reservations expired, or the first
// error encountered — a partial sweep on error is fine, since the next
// tick retries whatever is left.
func (uc *ExpireUnpaidReservations) Execute(ctx context.Context) (int, error) {
	expired, err := uc.Reservations.ListUnpaidExpired(ctx, uc.Clock.Now())
	if err != nil {
		return 0, err
	}

	count := 0
	for _, res := range expired {
		if err := uc.expireOne(ctx, res); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (uc *ExpireUnpaidReservations) expireOne(ctx context.Context, res *domain.Reservation) error {
	cycleID, _, err := uc.Listings.CycleAndPrice(ctx, res.ListingID)
	if err != nil {
		return err
	}
	if err := uc.Cycles.IncrementAvailability(ctx, cycleID, res.Quantity); err != nil {
		return err
	}

	res.Status = domain.ReservationCancelled
	if res.Payment != nil {
		res.Payment.Status = domain.PaymentFailed
	}
	if err := uc.Reservations.Update(ctx, res); err != nil {
		return err
	}

	return uc.Notifications.Enqueue(ctx, &domain.Notification{
		UserID:    res.BuyerID,
		Type:      "reservation_expired",
		Message:   "Votre réservation non payée a expiré et la place a été libérée.",
		Status:    domain.NotificationPending,
		CreatedAt: uc.Clock.Now(),
	})
}
