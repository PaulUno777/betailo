package reservation

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ConfirmDelivery is the buyer's escrow release: "Confirmer la
// réception" in the design. internal/worker performs the same release
// automatically 48h after DeliveredAt if the buyer never calls this and
// no dispute is open — see its doc comment.
type ConfirmDelivery struct {
	Reservations  ports.ReservationRepository
	Reports       ports.ReportRepository
	Listings      ports.ListingStore
	Cycles        ports.CycleStore
	Farms         ports.FarmRepository
	Trust         ports.TrustRecomputer
	Clock         ports.Clock
	Notifications ports.NotificationRepository
}

type ConfirmDeliveryInput struct {
	ReservationID bson.ObjectID
	BuyerID       bson.ObjectID
}

// Execute returns domain.ErrForbidden if the caller isn't the
// reservation's buyer, and domain.ErrConflict if it isn't in the
// delivered/held state yet, or has an open dispute against it.
func (uc *ConfirmDelivery) Execute(ctx context.Context, in ConfirmDeliveryInput) (*domain.Reservation, error) {
	res, err := uc.Reservations.FindByID(ctx, in.ReservationID)
	if err != nil {
		return nil, err
	}
	if res.BuyerID != in.BuyerID {
		return nil, fmt.Errorf("reservation: not the buyer: %w", domain.ErrForbidden)
	}
	if res.Status != domain.ReservationDelivered || res.Payment == nil || res.Payment.Status != domain.PaymentHeld {
		return nil, fmt.Errorf("reservation: not awaiting delivery confirmation: %w", domain.ErrConflict)
	}

	disputed, err := hasOpenDispute(ctx, uc.Reports, res.ID)
	if err != nil {
		return nil, err
	}
	if disputed {
		return nil, fmt.Errorf("reservation: an open dispute must be resolved first: %w", domain.ErrConflict)
	}

	release(res, uc.Clock)
	if err := uc.Reservations.Update(ctx, res); err != nil {
		return nil, err
	}

	// Trust score recomputation is best-effort telemetry derived from
	// this release, not part of its correctness — a failure here must
	// never undo (or block reporting) a payment that already released.
	if ownerID, err := farmOwner(ctx, uc.Listings, uc.Cycles, uc.Farms, res.ListingID); err == nil {
		_ = uc.Trust.Execute(ctx, ownerID)
	}
	Parties{Listings: uc.Listings, Cycles: uc.Cycles, Farms: uc.Farms}.notifyFarmer(ctx, uc.Notifications, uc.Clock, res,
		domain.NotifEscrowReleased, "Séquestre libéré pour %s.")

	return res, nil
}

// release mutates res in place to the released-escrow state — shared
// between ConfirmDelivery (buyer-triggered) and internal/worker's
// auto-release sweep so the two paths can never disagree on what
// "released" means.
func release(res *domain.Reservation, clock ports.Clock) {
	now := clock.Now()
	res.Payment.Status = domain.PaymentReleased
	res.Payment.ReleasedAt = &now
}

// hasOpenDispute reports whether any open/under_review report targets
// this reservation — the dispute flow rides the generic reporting
// mechanism (domain.ReportTargetReservation) rather than a bespoke
// entity, per the architecture document's "forum shares Trust's
// reporting mechanism" precedent.
func hasOpenDispute(ctx context.Context, reports ports.ReportRepository, reservationID bson.ObjectID) (bool, error) {
	rs, err := reports.ListByTarget(ctx, reservationID)
	if err != nil {
		return false, err
	}
	for _, r := range rs {
		if r.Status == domain.ReportOpen || r.Status == domain.ReportUnderReview {
			return true, nil
		}
	}
	return false, nil
}
