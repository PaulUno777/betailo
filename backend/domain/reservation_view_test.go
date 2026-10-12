package domain

import (
	"testing"
	"time"
)

func states(steps []EscrowStep) []TimelineStepState {
	out := make([]TimelineStepState, len(steps))
	for i, s := range steps {
		out[i] = s.State
	}
	return out
}

func TestReservationGroupStepsAndAutoRelease(t *testing.T) {
	now := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	const D, C, U = TimelineDone, TimelineCurrent, TimelineUpcoming

	tests := []struct {
		name  string
		r     *Reservation
		group ReservationGroup
		steps []TimelineStepState
		auto  bool
	}{
		{"reserved, unpaid", &Reservation{Status: ReservationPending, Payment: &PaymentInfo{Status: PaymentPending}},
			GroupActive, []TimelineStepState{C, U, U, U}, false},
		{"paid, awaiting delivery", &Reservation{Status: ReservationConfirmed, Payment: &PaymentInfo{Status: PaymentHeld, HeldAt: &now}},
			GroupActive, []TimelineStepState{D, C, U, U}, false},
		{"delivered, awaiting confirmation", &Reservation{Status: ReservationDelivered, DeliveredAt: &now, Payment: &PaymentInfo{Status: PaymentHeld, HeldAt: &now}},
			GroupActive, []TimelineStepState{D, D, C, U}, true},
		{"released", &Reservation{Status: ReservationDelivered, DeliveredAt: &now, Payment: &PaymentInfo{Status: PaymentReleased, HeldAt: &now}},
			GroupDone, []TimelineStepState{D, D, D, D}, false},
		{"expired unpaid", &Reservation{Status: ReservationCancelled, Payment: &PaymentInfo{Status: PaymentFailed}},
			GroupCancelled, []TimelineStepState{U, U, U, U}, false},
		{"refunded after payment", &Reservation{Status: ReservationConfirmed, Payment: &PaymentInfo{Status: PaymentRefunded, HeldAt: &now}},
			GroupCancelled, []TimelineStepState{D, U, U, U}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if g := GroupOf(tt.r); g != tt.group {
				t.Errorf("GroupOf = %s, want %s", g, tt.group)
			}
			got := states(EscrowSteps(tt.r))
			for i := range got {
				if got[i] != tt.steps[i] {
					t.Fatalf("EscrowSteps = %v, want %v", got, tt.steps)
				}
			}
			at := AutoReleaseAt(tt.r, 48*time.Hour)
			if (at != nil) != tt.auto {
				t.Fatalf("AutoReleaseAt = %v, want set=%v", at, tt.auto)
			}
			if at != nil && !at.Equal(now.Add(48*time.Hour)) {
				t.Errorf("AutoReleaseAt = %v, want delivered+48h", at)
			}
		})
	}
}
