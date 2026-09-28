package reservation

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// AdminReleaseEscrow is an admin's manual override of ConfirmDelivery —
// resolving a dispute in the buyer's favor... in the farmer's, rather:
// it releases the held payment exactly like a normal confirmation
// would, via the same release helper, so the two paths can never
// disagree on what "released" means.
type AdminReleaseEscrow struct {
	Reservations ports.ReservationRepository
	Listings     ports.ListingStore
	Cycles       ports.CycleStore
	Farms        ports.FarmRepository
	Trust        ports.TrustRecomputer
	Clock        ports.Clock
}

// Execute returns domain.ErrConflict if the reservation has no held
// payment to release.
func (uc *AdminReleaseEscrow) Execute(ctx context.Context, reservationID bson.ObjectID) (*domain.Reservation, error) {
	res, err := uc.Reservations.FindByID(ctx, reservationID)
	if err != nil {
		return nil, err
	}
	if res.Payment == nil || res.Payment.Status != domain.PaymentHeld {
		return nil, fmt.Errorf("reservation: no held payment to release: %w", domain.ErrConflict)
	}

	release(res, uc.Clock)
	if err := uc.Reservations.Update(ctx, res); err != nil {
		return nil, err
	}
	if ownerID, err := farmOwner(ctx, uc.Listings, uc.Cycles, uc.Farms, res.ListingID); err == nil {
		_ = uc.Trust.Execute(ctx, ownerID)
	}
	return res, nil
}

// AdminRefundEscrow is a dispute resolved in the buyer's favor — refunds
// the held payment and cancels the reservation.
type AdminRefundEscrow struct {
	Reservations ports.ReservationRepository
	Payments     ports.PaymentProvider
	Clock        ports.Clock
}

// Execute returns domain.ErrConflict if the reservation has no held
// payment to refund.
func (uc *AdminRefundEscrow) Execute(ctx context.Context, reservationID bson.ObjectID) (*domain.Reservation, error) {
	res, err := uc.Reservations.FindByID(ctx, reservationID)
	if err != nil {
		return nil, err
	}
	if res.Payment == nil || res.Payment.Status != domain.PaymentHeld {
		return nil, fmt.Errorf("reservation: no held payment to refund: %w", domain.ErrConflict)
	}

	if err := uc.Payments.Refund(ctx, reservationID.Hex(), res.TotalAmount); err != nil {
		return nil, err
	}
	res.Payment.Status = domain.PaymentRefunded
	res.Status = domain.ReservationCancelled
	if err := uc.Reservations.Update(ctx, res); err != nil {
		return nil, err
	}
	return res, nil
}
