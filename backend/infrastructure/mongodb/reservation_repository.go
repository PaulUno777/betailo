package mongodb

import (
	"context"
	"errors"
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
	ID              bson.ObjectID            `bson:"_id,omitempty"`
	ListingID       bson.ObjectID            `bson:"listingId"`
	BuyerID         bson.ObjectID            `bson:"buyerId"`
	Quantity        int                      `bson:"quantity"`
	UnitPrice       float64                  `bson:"unitPrice"`
	ServiceFee      float64                  `bson:"serviceFee,omitempty"`
	TotalAmount     float64                  `bson:"totalAmount"`
	Status          domain.ReservationStatus `bson:"status"`
	CreatedAt       time.Time                `bson:"createdAt"`
	Priority        bool                     `bson:"priority,omitempty"`
	PaymentDeadline *time.Time               `bson:"paymentDeadline,omitempty"`
	DeliveredAt     *time.Time               `bson:"deliveredAt,omitempty"`
	Payment         *paymentInfoDoc          `bson:"payment,omitempty"`
}

func fromDomainReservation(r *domain.Reservation) reservationDoc {
	doc := reservationDoc{
		ID:              r.ID,
		ListingID:       r.ListingID,
		BuyerID:         r.BuyerID,
		Quantity:        r.Quantity,
		UnitPrice:       r.UnitPrice,
		ServiceFee:      r.ServiceFee,
		TotalAmount:     r.TotalAmount,
		Status:          r.Status,
		CreatedAt:       r.CreatedAt,
		Priority:        r.Priority,
		PaymentDeadline: r.PaymentDeadline,
		DeliveredAt:     r.DeliveredAt,
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

func (d reservationDoc) toDomain() *domain.Reservation {
	r := &domain.Reservation{
		ID:              d.ID,
		ListingID:       d.ListingID,
		BuyerID:         d.BuyerID,
		Quantity:        d.Quantity,
		UnitPrice:       d.UnitPrice,
		ServiceFee:      d.ServiceFee,
		TotalAmount:     d.TotalAmount,
		Status:          d.Status,
		CreatedAt:       d.CreatedAt,
		Priority:        d.Priority,
		PaymentDeadline: d.PaymentDeadline,
		DeliveredAt:     d.DeliveredAt,
	}
	if d.Payment != nil {
		r.Payment = &domain.PaymentInfo{
			Status:     d.Payment.Status,
			Amount:     d.Payment.Amount,
			HeldAt:     d.Payment.HeldAt,
			ReleasedAt: d.Payment.ReleasedAt,
		}
	}
	return r
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

// FindByID implements ports.ReservationRepository.
func (r *ReservationRepository) FindByID(ctx context.Context, id bson.ObjectID) (*domain.Reservation, error) {
	var doc reservationDoc
	err := r.Coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

func (r *ReservationRepository) find(ctx context.Context, filter bson.M) ([]*domain.Reservation, error) {
	cur, err := r.Coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*domain.Reservation
	for cur.Next(ctx) {
		var doc reservationDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.toDomain())
	}
	return out, cur.Err()
}

// ListByBuyer implements ports.ReservationRepository.
func (r *ReservationRepository) ListByBuyer(ctx context.Context, buyerID bson.ObjectID) ([]*domain.Reservation, error) {
	return r.find(ctx, bson.M{"buyerId": buyerID})
}

// ListByListingIDs implements ports.ReservationRepository.
func (r *ReservationRepository) ListByListingIDs(ctx context.Context, listingIDs []bson.ObjectID) ([]*domain.Reservation, error) {
	if len(listingIDs) == 0 {
		return nil, nil
	}
	return r.find(ctx, bson.M{"listingId": bson.M{"$in": listingIDs}})
}

// Update implements ports.ReservationRepository.
func (r *ReservationRepository) Update(ctx context.Context, res *domain.Reservation) error {
	result, err := r.Coll.ReplaceOne(ctx, bson.M{"_id": res.ID}, fromDomainReservation(res))
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListUnpaidExpired implements ports.ReservationRepository.
func (r *ReservationRepository) ListUnpaidExpired(ctx context.Context, before time.Time) ([]*domain.Reservation, error) {
	return r.find(ctx, bson.M{
		"status":          domain.ReservationPending,
		"paymentDeadline": bson.M{"$lte": before},
	})
}

// ListDeliveredBefore implements ports.ReservationRepository.
func (r *ReservationRepository) ListDeliveredBefore(ctx context.Context, before time.Time) ([]*domain.Reservation, error) {
	return r.find(ctx, bson.M{
		"status":         domain.ReservationDelivered,
		"deliveredAt":    bson.M{"$lte": before},
		"payment.status": domain.PaymentHeld,
	})
}
