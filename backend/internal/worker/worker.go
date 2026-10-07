// Package worker is the one background goroutine the architecture
// document describes: same process, no separate service, ticking over
// two concrete responsibilities — advancing a cycle's status
// automatically (planned -> in_progress -> done) and running the
// reservation/escrow usecases that are time-based rather than
// request-based (closing a finished cycle's waiting list, escrow
// auto-release) — plus
// draining the notification outbox those usecases enqueue into.
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/reservation"
	"github.com/pauluno777/betailo/backend/domain"
)

// DefaultInterval is how often the worker ticks when Interval is left
// zero — frequent enough that a cycle's end or a 48h auto-release never
// drifts by more than a minute in practice, without
// hammering the database.
const DefaultInterval = time.Minute

// Worker owns every usecase its tick calls — composed once in
// internal/app, not re-built per tick.
type Worker struct {
	Cycles        ports.CycleStore
	CloseUnpaid   *reservation.CloseUnpaidForCycle
	AutoRelease   *reservation.AutoReleaseDeliveries
	Notifications ports.NotificationRepository
	Clock         ports.Clock
	Logger        *slog.Logger
	Interval      time.Duration
}

// Run blocks, ticking until ctx is cancelled — call it in its own
// goroutine (see internal/app.Application.Run).
func (w *Worker) Run(ctx context.Context) {
	interval := w.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	if err := w.advanceCycles(ctx); err != nil {
		w.Logger.Error("worker: advance cycles", slog.Any("err", err))
	}
	if n, err := w.AutoRelease.Execute(ctx); err != nil {
		w.Logger.Error("worker: auto-release deliveries", slog.Any("err", err))
	} else if n > 0 {
		w.Logger.Info("worker: auto-released deliveries", slog.Int("count", n))
	}
	if err := w.drainNotifications(ctx); err != nil {
		w.Logger.Error("worker: drain notifications", slog.Any("err", err))
	}
}

// advanceCycles is the cycle status machine the architecture document
// names explicitly: "demarrer un cycle a sa date de debut" and the
// symmetric transition into "termine" once it ends.
func (w *Worker) advanceCycles(ctx context.Context) error {
	now := w.Clock.Now()

	planned, err := w.Cycles.ListByStatus(ctx, domain.CycleStatusPlanned)
	if err != nil {
		return err
	}
	for _, c := range planned {
		if !c.StartsAt.After(now) {
			if err := w.Cycles.UpdateStatus(ctx, c.ID, domain.CycleStatusInProgress); err != nil {
				return err
			}
		}
	}

	inProgress, err := w.Cycles.ListByStatus(ctx, domain.CycleStatusInProgress)
	if err != nil {
		return err
	}
	for _, c := range inProgress {
		if c.EndsAt != nil && !c.EndsAt.After(now) {
			if err := w.Cycles.UpdateStatus(ctx, c.ID, domain.CycleStatusDone); err != nil {
				return err
			}
			if w.CloseUnpaid != nil {
				if n, err := w.CloseUnpaid.CloseUnpaid(ctx, c.ID); err != nil {
					return err
				} else if n > 0 {
					w.Logger.Info("worker: closed unpaid reservations of a finished cycle", slog.Int("count", n))
				}
			}
		}
	}
	return nil
}

// drainNotifications "sends" every pending notification — there is no
// real email/SMS provider behind this (see domain.Notification's doc
// comment), so sending means logging and marking it sent, an honest
// stand-in for a real channel plugged in later.
func (w *Worker) drainNotifications(ctx context.Context) error {
	pending, err := w.Notifications.ListPending(ctx)
	if err != nil {
		return err
	}
	for _, n := range pending {
		w.Logger.Info("notification",
			slog.String("userId", n.UserID.Hex()), slog.String("type", n.Type), slog.String("message", n.Message))
		if err := w.Notifications.MarkSent(ctx, n.ID, w.Clock.Now()); err != nil {
			return err
		}
	}
	return nil
}
