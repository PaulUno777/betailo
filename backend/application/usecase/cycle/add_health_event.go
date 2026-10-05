package cycle

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

const (
	maxHealthTitleLen  = 80
	maxHealthDetailLen = 280
	// healthEventFutureSlack lets a farmer log something happening
	// later today (or schedule "Éclosion prévue") without allowing
	// arbitrary future dates.
	healthEventFutureSlack = 60 * 24 * time.Hour
)

// AddHealthEvent appends one entry to a cycle's health log, the
// "carnet sanitaire" buyers see as the cycle's timeline ("Mirage J+12 ·
// 162 œufs fécondés"). Only the farm's owner can add to it.
type AddHealthEvent struct {
	Farms  ports.FarmRepository
	Cycles ports.CycleStore
	Clock  ports.Clock
}

type AddHealthEventInput struct {
	CycleID bson.ObjectID
	OwnerID bson.ObjectID
	Title   string // short, e.g. "Vaccination Newcastle"
	Detail  string // optional, e.g. "180 sujets, rappel à J+21"
	Date    time.Time
}

// Execute returns domain.ErrNotFound for an unknown cycle or farm,
// domain.ErrForbidden if OwnerID doesn't own the farm, domain.ErrInvalid
// for a bad title/detail/date or a cancelled cycle, and
// domain.ErrConflict once the log holds domain.MaxHealthLogEntries.
func (uc *AddHealthEvent) Execute(ctx context.Context, in AddHealthEventInput) (*domain.Cycle, error) {
	title := strings.TrimSpace(in.Title)
	detail := strings.TrimSpace(in.Detail)
	switch {
	case title == "":
		return nil, fmt.Errorf("cycle: health event title is required: %w", domain.ErrInvalid)
	case utf8.RuneCountInString(title) > maxHealthTitleLen:
		return nil, fmt.Errorf("cycle: health event title must be at most %d characters: %w", maxHealthTitleLen, domain.ErrInvalid)
	case utf8.RuneCountInString(detail) > maxHealthDetailLen:
		return nil, fmt.Errorf("cycle: health event detail must be at most %d characters: %w", maxHealthDetailLen, domain.ErrInvalid)
	case in.Date.IsZero():
		return nil, fmt.Errorf("cycle: health event date is required: %w", domain.ErrInvalid)
	}

	c, err := uc.Cycles.FindByID(ctx, in.CycleID)
	if err != nil {
		return nil, err
	}
	f, err := uc.Farms.FindByID(ctx, c.FarmID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != in.OwnerID {
		return nil, fmt.Errorf("cycle: not the farm owner: %w", domain.ErrForbidden)
	}
	if c.Status == domain.CycleStatusCancelled {
		return nil, fmt.Errorf("cycle: cannot log health events on a cancelled cycle: %w", domain.ErrInvalid)
	}
	if in.Date.After(uc.Clock.Now().Add(healthEventFutureSlack)) {
		return nil, fmt.Errorf("cycle: health event date is too far in the future: %w", domain.ErrInvalid)
	}

	event := domain.HealthEvent{Type: title, Description: detail, Date: in.Date.UTC()}
	if err := uc.Cycles.AppendHealthEvent(ctx, c.ID, event, domain.MaxHealthLogEntries); err != nil {
		return nil, err
	}
	c.HealthLog = append(c.HealthLog, event)
	return c, nil
}
