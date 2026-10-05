package domain

import "time"

// ReservationGroup is the tab a reservation belongs under: "En cours",
// "Terminées", "Annulées".
type ReservationGroup string

const (
	GroupActive    ReservationGroup = "active"
	GroupDone      ReservationGroup = "done"
	GroupCancelled ReservationGroup = "cancelled"
)

// GroupOf files a reservation under its tab. Escrow released means done;
// cancelled, expired or refunded means cancelled; anything else is still
// in progress.
func GroupOf(r *Reservation) ReservationGroup {
	switch {
	case r.Status == ReservationCancelled:
		return GroupCancelled
	case r.Payment != nil && r.Payment.Status == PaymentRefunded:
		return GroupCancelled
	case r.Payment != nil && r.Payment.Status == PaymentReleased:
		return GroupDone
	default:
		return GroupActive
	}
}

// EscrowStepKey names one stage of a reservation's progress bar.
type EscrowStepKey string

const (
	StepPaid      EscrowStepKey = "paid"
	StepDelivered EscrowStepKey = "delivered"
	StepConfirmed EscrowStepKey = "confirmed" // the buyer's confirmation, or the auto-release standing in for it
	StepReleased  EscrowStepKey = "released"
)

// EscrowStep is one segment of the progress bar, in the same
// done/current/upcoming vocabulary as the cycle timeline.
type EscrowStep struct {
	Key   EscrowStepKey
	State TimelineStepState
}

// EscrowSteps lays out paid → delivered → confirmed → released for a
// reservation, marking each step done, the next one current, and the
// rest upcoming. A cancelled or refunded reservation has no current
// step: its progress just stops where it was.
func EscrowSteps(r *Reservation) []EscrowStep {
	paid := r.Payment != nil && (r.Payment.Status == PaymentHeld || r.Payment.Status == PaymentReleased ||
		(r.Payment.Status == PaymentRefunded && r.Payment.HeldAt != nil))
	released := r.Payment != nil && r.Payment.Status == PaymentReleased
	done := []bool{paid, r.DeliveredAt != nil, released, released}
	keys := []EscrowStepKey{StepPaid, StepDelivered, StepConfirmed, StepReleased}

	stopped := GroupOf(r) == GroupCancelled
	steps := make([]EscrowStep, len(keys))
	currentAssigned := false
	for i, k := range keys {
		switch {
		case done[i]:
			steps[i] = EscrowStep{Key: k, State: TimelineDone}
		case !currentAssigned && !stopped:
			steps[i] = EscrowStep{Key: k, State: TimelineCurrent}
			currentAssigned = true
		default:
			steps[i] = EscrowStep{Key: k, State: TimelineUpcoming}
		}
	}
	return steps
}

// AutoReleaseAt is when escrow releases on its own if the buyer neither
// confirms nor disputes a delivered reservation: DeliveredAt + grace.
// Nil when that clock isn't running.
func AutoReleaseAt(r *Reservation, grace time.Duration) *time.Time {
	if r.DeliveredAt == nil || r.Payment == nil || r.Payment.Status != PaymentHeld {
		return nil
	}
	at := r.DeliveredAt.Add(grace)
	return &at
}
