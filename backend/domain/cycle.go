package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CycleStatus is a Cycle's position in its lifecycle.
type CycleStatus string

const (
	CycleStatusPlanned    CycleStatus = "planifie"
	CycleStatusInProgress CycleStatus = "en_cours"
	CycleStatusDone       CycleStatus = "termine"
	CycleStatusCancelled  CycleStatus = "annule"
)

// HealthEvent is one entry in a Cycle's embedded health log. It is
// always read and written together with its parent Cycle and stays
// bounded in size (no unlimited health record) — exactly the case
// MongoDB's document embedding targets, per the "Modèle de données"
// section of the "Concepts & Architecture" document.
type HealthEvent struct {
	Type        string
	Description string
	Date        time.Time
}

// Cycle is the domain pivot: a breeding/growth/laying cycle that
// listings, health logs and alerts attach to — see the "Cycle — le
// concept pivot" section of the architecture document for why this,
// and not a static "product", is the entity a buyer actually reserves.
type Cycle struct {
	ID       bson.ObjectID
	FarmID   bson.ObjectID
	Species  string
	Type     string
	Status   CycleStatus
	Capacity int

	// AvailableQuantity is maintained by atomic decrement
	// (findOneAndUpdate + $inc) on reservation, never recomputed by any
	// usecase — the only way to prevent overselling without an explicit
	// lock. See application/usecase/reservation.
	AvailableQuantity int

	StartsAt  time.Time
	EndsAt    *time.Time
	HealthLog []HealthEvent
}
