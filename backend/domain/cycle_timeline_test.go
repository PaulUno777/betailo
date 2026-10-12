package domain

import (
	"testing"
	"time"
)

func TestCycleTimeline(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 9, d, 8, 0, 0, 0, time.UTC) }
	end := day(27)
	c := &Cycle{
		Status:   CycleStatusInProgress,
		StartsAt: day(2),
		EndsAt:   &end,
		HealthLog: []HealthEvent{
			{Type: "Éclosion prévue", Date: day(23)},
			{Type: "Mirage J+12", Description: "162 œufs fécondés (90 %)", Date: day(14)},
		},
	}

	steps := CycleTimeline(c, day(20))
	want := []struct {
		kind  TimelineStepKind
		title string
		state TimelineStepState
	}{
		{TimelineStart, "", TimelineDone},
		{TimelineHealth, "Mirage J+12", TimelineDone},
		{TimelineHealth, "Éclosion prévue", TimelineCurrent},
		{TimelineEnd, "", TimelineUpcoming},
	}
	if len(steps) != len(want) {
		t.Fatalf("len(steps) = %d, want %d", len(steps), len(want))
	}
	for i, w := range want {
		if steps[i].Kind != w.kind || steps[i].Title != w.title || steps[i].State != w.state {
			t.Errorf("step %d = %+v, want kind=%s title=%q state=%s", i, steps[i], w.kind, w.title, w.state)
		}
	}
	if steps[1].Detail != "162 œufs fécondés (90 %)" {
		t.Errorf("health detail lost: %q", steps[1].Detail)
	}

	c.Status = CycleStatusDone
	for _, s := range CycleTimeline(c, day(20)) {
		if s.State != TimelineDone {
			t.Fatalf("a done cycle should have every step done, got %+v", s)
		}
	}

	c.Status = CycleStatusCancelled
	for _, s := range CycleTimeline(c, day(20)) {
		if s.State == TimelineCurrent {
			t.Fatalf("a cancelled cycle has no current step, got %+v", s)
		}
	}
}
