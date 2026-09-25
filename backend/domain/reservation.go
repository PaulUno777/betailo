package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ReservationStatus is a Reservation's lifecycle state. Values match
// schemas/reservations.json's $jsonSchema enum exactly — do not change
// these string literals without updating that file too, or MongoDB
// will reject any document whose status isn't one of these four
// values.
type ReservationStatus string

const (
	ReservationPending   ReservationStatus = "pending"
	ReservationConfirmed ReservationStatus = "confirmed"
	ReservationDelivered ReservationStatus = "delivered"
	ReservationCancelled ReservationStatus = "cancelled"
)

// Reservation engages quantity within a Cycle's listing, before the
// cycle ends — it's the response to scarcity: see the architecture
// document's domain-concepts overview for why overselling has to be
// prevented at the database level, not just in application code.
// UnitPrice and TotalAmount are frozen at reservation time, like an
// invoice line: they never recompute if the listing's price changes
// afterwards.
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
