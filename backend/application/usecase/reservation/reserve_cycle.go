// Package reservation implements the one usecase in this codebase with
// real business logic today: reserving quantity within a cycle's
// listing and preventing overselling via a MongoDB multi-document
// transaction — see "1. Réservation d'un cycle, sans survente" in the
// "Concepts & Architecture" document for the sequence diagram this
// package implements.
//
// ReserveCycle depends only on application/ports interfaces
// (ListingRepository, CycleRepository, ReservationRepository,
// TxManager), never on *mongo.Client or *mongo.Collection — the
// Dependency Inversion Principle applied at the one boundary that
// actually needs it. That keeps the business rule ("check-and-decrement
// and insert happen atomically") testable with fakes and independent of
// the driver; infrastructure/mongo is the only layer that imports
// mongo-driver. internal/app wires the two together once, at startup.
package reservation

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ReserveCycle is the usecase: read the listing, then run the
// availability check-and-decrement and the reservation insert inside
// one TxManager so they succeed or fail together.
type ReserveCycle struct {
	Listings     ports.ListingRepository
	Cycles       ports.CycleRepository
	Reservations ports.ReservationRepository
	Tx           ports.TxManager
}

// ReserveCycleInput is what a caller (an HTTP handler, once one exists)
// supplies.
type ReserveCycleInput struct {
	ListingID bson.ObjectID
	BuyerID   bson.ObjectID
	Quantity  int
}

// ReserveCycleOutput wraps the usecase's result.
type ReserveCycleOutput struct {
	Reservation *domain.Reservation
}

// Execute engages Quantity units of the cycle behind ListingID for
// BuyerID. The price is read from the listing before the unit of work
// opens and frozen onto the reservation; the availability check inside
// it means a concurrent reservation on the same cycle either sees the
// updated (lower) availableQuantity or aborts with
// domain.ErrInsufficientCapacity — never both succeeding on the same
// unit.
func (uc *ReserveCycle) Execute(ctx context.Context, in ReserveCycleInput) (*ReserveCycleOutput, error) {
	if in.Quantity <= 0 {
		return nil, errors.New("reservation: quantity must be positive")
	}

	cycleID, unitPrice, err := uc.Listings.CycleAndPrice(ctx, in.ListingID)
	if err != nil {
		return nil, err
	}

	var res *domain.Reservation
	err = uc.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.Cycles.DecrementAvailability(ctx, cycleID, in.Quantity); err != nil {
			return err
		}

		res = &domain.Reservation{
			ID:          bson.NewObjectID(),
			ListingID:   in.ListingID,
			BuyerID:     in.BuyerID,
			Quantity:    in.Quantity,
			UnitPrice:   unitPrice,
			TotalAmount: unitPrice * float64(in.Quantity),
			Status:      domain.ReservationConfirmed,
			CreatedAt:   time.Now().UTC(),
		}

		return uc.Reservations.Insert(ctx, res)
	})
	if err != nil {
		return nil, err
	}

	return &ReserveCycleOutput{Reservation: res}, nil
}
