package reservation

import (
	"context"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// AutoReleaseGrace is how long a delivered reservation waits for the
// buyer's explicit ConfirmDelivery before internal/worker releases
// escrow on their behalf — the design's "Sans action de votre part, le
// séquestre sera libéré dans 48 h."
const AutoReleaseGrace = 48 * time.Hour

// AutoReleaseDeliveries is internal/worker's sweep: every delivered
// reservation past AutoReleaseGrace with no open dispute gets its
// escrow released exactly the way ConfirmDelivery would — see release
// and hasOpenDispute, shared by both paths so they can never disagree
// on what "released" means.
type AutoReleaseDeliveries struct {
	Reservations  ports.ReservationRepository
	Reports       ports.ReportRepository
	Listings      ports.ListingStore
	Cycles        ports.CycleStore
	Farms         ports.FarmRepository
	Trust         ports.TrustRecomputer
	Notifications ports.NotificationRepository
	Clock         ports.Clock
}

// Execute returns the number of reservations released.
func (uc *AutoReleaseDeliveries) Execute(ctx context.Context) (int, error) {
	deadline := uc.Clock.Now().Add(-AutoReleaseGrace)
	candidates, err := uc.Reservations.ListDeliveredBefore(ctx, deadline)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, res := range candidates {
		disputed, err := hasOpenDispute(ctx, uc.Reports, res.ID)
		if err != nil {
			return count, err
		}
		if disputed {
			continue
		}

		release(res, uc.Clock)
		if err := uc.Reservations.Update(ctx, res); err != nil {
			return count, err
		}

		if ownerID, err := farmOwner(ctx, uc.Listings, uc.Cycles, uc.Farms, res.ListingID); err == nil {
			_ = uc.Trust.Execute(ctx, ownerID)
		}
		parties := Parties{Listings: uc.Listings, Cycles: uc.Cycles, Farms: uc.Farms}
		parties.notifyBuyer(ctx, uc.Notifications, uc.Clock, res, domain.NotifEscrowAutoReleased,
			"Le séquestre de %s a été libéré automatiquement, faute de confirmation ou de litige.", nil)
		parties.notifyFarmer(ctx, uc.Notifications, uc.Clock, res, domain.NotifEscrowReleased, "Séquestre libéré pour %s.")
		count++
	}
	return count, nil
}
