package reservation

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// OpenDispute is "Ouvrir un litige": the buyer flags a held or
// delivered reservation as disputed. It rides the generic reporting
// mechanism (domain.ReportTargetReservation) — see hasOpenDispute, which
// ConfirmDelivery and internal/worker's auto-release sweep both consult
// before releasing escrow. An admin resolves the dispute via
// application/usecase/admin's escrow release/refund actions.
type OpenDispute struct {
	Reservations ports.ReservationRepository
	Reports      ports.ReportRepository
	Clock        ports.Clock
}

type OpenDisputeInput struct {
	ReservationID bson.ObjectID
	BuyerID       bson.ObjectID
	Reason        string
}

// Execute returns domain.ErrForbidden if the caller isn't the
// reservation's buyer, and domain.ErrConflict if there's no held
// payment to dispute.
func (uc *OpenDispute) Execute(ctx context.Context, in OpenDisputeInput) (*domain.Report, error) {
	res, err := uc.Reservations.FindByID(ctx, in.ReservationID)
	if err != nil {
		return nil, err
	}
	if res.BuyerID != in.BuyerID {
		return nil, fmt.Errorf("reservation: not the buyer: %w", domain.ErrForbidden)
	}
	if res.Payment == nil || res.Payment.Status != domain.PaymentHeld {
		return nil, fmt.Errorf("reservation: no held payment to dispute: %w", domain.ErrConflict)
	}
	if in.Reason == "" {
		return nil, fmt.Errorf("reservation: a reason is required: %w", domain.ErrInvalid)
	}

	report := &domain.Report{
		ID:         bson.NewObjectID(),
		ReporterID: in.BuyerID,
		TargetID:   res.ID,
		TargetType: domain.ReportTargetReservation,
		Reason:     in.Reason,
		Status:     domain.ReportOpen,
		CreatedAt:  uc.Clock.Now(),
	}
	if err := uc.Reports.Insert(ctx, report); err != nil {
		return nil, err
	}
	return report, nil
}
