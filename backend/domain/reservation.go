package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ReservationStatus is a Reservation's lifecycle state.
type ReservationStatus string

const (
	ReservationPending   ReservationStatus = "en_attente"
	ReservationConfirmed ReservationStatus = "confirmed"
	ReservationDelivered ReservationStatus = "livree"
	ReservationCancelled ReservationStatus = "annulee"
)

// Reservation engages quantity within a Cycle's listing, before the
// cycle ends — see "Reservation — la réponse à la rareté" in the
// architecture document. UnitPrice and TotalAmount are frozen at
// reservation time, like an invoice line: they never recompute if the
// listing's price changes afterwards.
type Reservation struct {
	ID          bson.ObjectID
	ListingID   bson.ObjectID
	BuyerID     bson.ObjectID
	Quantity    int
	UnitPrice   float64
	TotalAmount float64
	Status      ReservationStatus
	CreatedAt   time.Time
	Payment     *PaymentInfo
}
