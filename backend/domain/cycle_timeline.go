package domain

import (
	"sort"
	"time"
)

// TimelineStepKind says what a timeline step marks: the cycle's start,
// one health-log entry, or its (planned) end.
type TimelineStepKind string

const (
	TimelineStart  TimelineStepKind = "start"
	TimelineHealth TimelineStepKind = "health"
	TimelineEnd    TimelineStepKind = "end"
)

// TimelineStepState places a step relative to now. Exactly one step is
// "current" while the cycle runs: the next one still ahead.
type TimelineStepState string

const (
	TimelineDone     TimelineStepState = "done"
	TimelineCurrent  TimelineStepState = "current"
	TimelineUpcoming TimelineStepState = "upcoming"
)

// TimelineStep is one row of the "Déroulé du cycle" the buyer sees.
// Title and Detail are only set for health entries; the client labels
// start and end steps itself (they're localized, health entries are
// the farmer's own words).
type TimelineStep struct {
	Kind   TimelineStepKind
	Date   time.Time
	Title  string
	Detail string
	State  TimelineStepState
}

// MaxHealthLogEntries bounds a cycle's embedded health log. The log is
// embedded precisely because it stays small (see HealthEvent).
const MaxHealthLogEntries = 60

// CycleTimeline builds the cycle's public timeline at now: its start,
// every health-log entry in date order, then its end when one is set.
// A finished cycle has every step done; otherwise steps up to now are
// done, the first one after now is current, and the rest are upcoming.
func CycleTimeline(c *Cycle, now time.Time) []TimelineStep {
	steps := make([]TimelineStep, 0, len(c.HealthLog)+2)
	steps = append(steps, TimelineStep{Kind: TimelineStart, Date: c.StartsAt})

	log := append([]HealthEvent(nil), c.HealthLog...)
	sort.SliceStable(log, func(i, j int) bool { return log[i].Date.Before(log[j].Date) })
	for _, e := range log {
		steps = append(steps, TimelineStep{Kind: TimelineHealth, Date: e.Date, Title: e.Type, Detail: e.Description})
	}
	if c.EndsAt != nil {
		steps = append(steps, TimelineStep{Kind: TimelineEnd, Date: *c.EndsAt})
	}

	currentAssigned := false
	for i := range steps {
		switch {
		case c.Status == CycleStatusDone || !steps[i].Date.After(now):
			steps[i].State = TimelineDone
		case !currentAssigned && c.Status != CycleStatusCancelled:
			steps[i].State = TimelineCurrent
			currentAssigned = true
		default:
			steps[i].State = TimelineUpcoming
		}
	}
	return steps
}
