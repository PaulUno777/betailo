package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CycleStatus is a Cycle's position in its lifecycle. Values match
// schemas/cycles.json's $jsonSchema enum exactly — do not change these
// string literals without updating that file too, or MongoDB will
// reject any document whose status isn't one of these four values.
type CycleStatus string

const (
	CycleStatusPlanned    CycleStatus = "planned"
	CycleStatusInProgress CycleStatus = "in_progress"
	CycleStatusDone       CycleStatus = "done"
	CycleStatusCancelled  CycleStatus = "cancelled"
)

// HealthEvent is one entry in a Cycle's embedded health log. It is
// always read and written together with its parent Cycle and stays
// bounded in size (no unlimited health record) — exactly the case
// MongoDB's document embedding targets, per the data model section of
// the "Concepts & Architecture" document.
type HealthEvent struct {
	Type        string
	Description string
	Date        time.Time
}

// Cycle is the domain pivot: a breeding/growth/laying cycle that
// listings, health logs and alerts attach to — it's the entity a buyer
// actually reserves, not a static "product". See the architecture
// document's overview of the domain concepts for why.
type Cycle struct {
	ID bson.ObjectID
	// Reference is the farmer-facing code ("CYC-0918"), assigned from a
	// counter at creation.
	Reference string
	FarmID    bson.ObjectID
	Species   string
	Type      string
	Status    CycleStatus
	Capacity  int

	// MinPerReservation and MaxPerReservation bound a single
	// reservation's quantity (0 means "no bound"). Enforced by
	// application/usecase/reservation.ReserveCycle, not by the schema —
	// same rationale as the listing/cycle "orphan" invariant in
	// ADR-0001: the service layer owns cross-field business rules, the
	// $jsonSchema validator only owns per-field shape.
	MinPerReservation int
	MaxPerReservation int

	// AvailableQuantity is maintained by atomic decrement
	// (findOneAndUpdate + $inc) on reservation, never recomputed by any
	// usecase — the only way to prevent overselling without an explicit
	// lock. See application/usecase/reservation.
	AvailableQuantity int

	StartsAt  time.Time
	EndsAt    *time.Time
	HealthLog []HealthEvent
}
