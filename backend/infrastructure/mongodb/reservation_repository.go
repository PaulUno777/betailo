package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// paymentInfoDoc mirrors domain.PaymentInfo for BSON persistence — kept
// here, not on the domain type, so domain.PaymentInfo stays free of any
// driver-specific tag.
type paymentInfoDoc struct {
	Status     domain.PaymentStatus `bson:"status"`
	Amount     float64              `bson:"amount"`
	HeldAt     *time.Time           `bson:"heldAt,omitempty"`
	ReleasedAt *time.Time           `bson:"releasedAt,omitempty"`
}

// reservationDoc mirrors domain.Reservation for BSON persistence.
// domain.Reservation itself carries no bson tags (see domain/reservation.go);
// mapping happens only here, at the infrastructure boundary.
type reservationDoc struct {
	ID          bson.ObjectID            `bson:"_id,omitempty"`
	ListingID   bson.ObjectID            `bson:"listingId"`
	BuyerID     bson.ObjectID            `bson:"buyerId"`
	Quantity    int                      `bson:"quantity"`
	UnitPrice   float64                  `bson:"unitPrice"`
	TotalAmount float64                  `bson:"totalAmount"`
	Status      domain.ReservationStatus `bson:"status"`
	CreatedAt   time.Time                `bson:"createdAt"`
	Payment     *paymentInfoDoc          `bson:"payment,omitempty"`
}

func fromDomainReservation(r *domain.Reservation) reservationDoc {
	doc := reservationDoc{
		ID:          r.ID,
		ListingID:   r.ListingID,
		BuyerID:     r.BuyerID,
		Quantity:    r.Quantity,
		UnitPrice:   r.UnitPrice,
		TotalAmount: r.TotalAmount,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
	}
	if r.Payment != nil {
		doc.Payment = &paymentInfoDoc{
			Status:     r.Payment.Status,
			Amount:     r.Payment.Amount,
			HeldAt:     r.Payment.HeldAt,
			ReleasedAt: r.Payment.ReleasedAt,
		}
	}
	return doc
}

// ReservationRepository implements ports.ReservationRepository against
// the "reservations" collection.
type ReservationRepository struct {
	Coll *mongo.Collection
}

var _ ports.ReservationRepository = (*ReservationRepository)(nil)

// Insert implements ports.ReservationRepository.
func (r *ReservationRepository) Insert(ctx context.Context, res *domain.Reservation) error {
	_, err := r.Coll.InsertOne(ctx, fromDomainReservation(res))
	return err
}
