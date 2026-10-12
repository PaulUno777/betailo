package ports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CycleDetails are a cycle's editable, non-status fields.
type CycleDetails struct {
	StartsAt          time.Time
	EndsAt            *time.Time
	MinPerReservation int
	MaxPerReservation int
}

// CycleEditor edits a cycle in place.
type CycleEditor interface {
	// UpdateDetails sets details and moves capacity and available
	// places together by capacityDelta, in one conditional update that
	// refuses (domain.ErrConflict) to take availability below zero, i.e.
	// to un-sell reserved places. domain.ErrNotFound for an unknown cycle.
	UpdateDetails(ctx context.Context, id bson.ObjectID, d CycleDetails, capacityDelta int) error
}
