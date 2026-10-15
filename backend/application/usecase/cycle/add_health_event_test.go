package cycle

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func setupHealthEvent(t *testing.T, status domain.CycleStatus) (*AddHealthEvent, *domain.Cycle, bson.ObjectID, time.Time) {
	t.Helper()
	owner := bson.NewObjectID()
	farm := &domain.Farm{ID: bson.NewObjectID(), OwnerID: owner}
	cycles := newFakeCycles()
	c := &domain.Cycle{FarmID: farm.ID, Species: "poussins", Status: status}
	if err := cycles.Create(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 14, 9, 0, 0, 0, time.UTC)
	return &AddHealthEvent{Farms: newFakeFarms(farm), Cycles: cycles, Clock: fakeClock{now: now}}, c, owner, now
}

func TestAddHealthEvent_AppendsTrimmedEntry(t *testing.T) {
	uc, c, owner, now := setupHealthEvent(t, domain.CycleStatusInProgress)

	got, err := uc.Execute(context.Background(), AddHealthEventInput{
		CycleID: c.ID, OwnerID: owner, Title: "  Mirage J+12 ", Detail: " 162 œufs fécondés ", Date: now,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if n := len(got.HealthLog); n != 1 {
		t.Fatalf("health log length = %d, want 1", n)
	}
	if e := got.HealthLog[0]; e.Type != "Mirage J+12" || e.Description != "162 œufs fécondés" {
		t.Errorf("entry = %+v, want trimmed title/detail", e)
	}
}

func TestAddHealthEvent_Rejections(t *testing.T) {
	tests := []struct {
		name    string
		status  domain.CycleStatus
		mutate  func(in *AddHealthEventInput, now time.Time)
		wantErr error
	}{
		{"not the owner", domain.CycleStatusInProgress, func(in *AddHealthEventInput, _ time.Time) { in.OwnerID = bson.NewObjectID() }, domain.ErrForbidden},
		{"empty title", domain.CycleStatusInProgress, func(in *AddHealthEventInput, _ time.Time) { in.Title = "  " }, domain.ErrInvalid},
		{"title too long", domain.CycleStatusInProgress, func(in *AddHealthEventInput, _ time.Time) { in.Title = strings.Repeat("a", 81) }, domain.ErrInvalid},
		{"detail too long", domain.CycleStatusInProgress, func(in *AddHealthEventInput, _ time.Time) { in.Detail = strings.Repeat("a", 281) }, domain.ErrInvalid},
		{"missing date", domain.CycleStatusInProgress, func(in *AddHealthEventInput, _ time.Time) { in.Date = time.Time{} }, domain.ErrInvalid},
		{"far future", domain.CycleStatusInProgress, func(in *AddHealthEventInput, now time.Time) { in.Date = now.Add(90 * 24 * time.Hour) }, domain.ErrInvalid},
		{"cancelled cycle", domain.CycleStatusCancelled, func(*AddHealthEventInput, time.Time) {}, domain.ErrInvalid},
		{"unknown cycle", domain.CycleStatusInProgress, func(in *AddHealthEventInput, _ time.Time) { in.CycleID = bson.NewObjectID() }, domain.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, c, owner, now := setupHealthEvent(t, tt.status)
			in := AddHealthEventInput{CycleID: c.ID, OwnerID: owner, Title: "Vaccination", Date: now}
			tt.mutate(&in, now)
			if _, err := uc.Execute(context.Background(), in); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddHealthEvent_FullLogIsConflict(t *testing.T) {
	uc, c, owner, now := setupHealthEvent(t, domain.CycleStatusInProgress)
	for range domain.MaxHealthLogEntries {
		if _, err := uc.Execute(context.Background(), AddHealthEventInput{CycleID: c.ID, OwnerID: owner, Title: "Pesée", Date: now}); err != nil {
			t.Fatal(err)
		}
	}
	_, err := uc.Execute(context.Background(), AddHealthEventInput{CycleID: c.ID, OwnerID: owner, Title: "Pesée", Date: now})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("Execute() on a full log: error = %v, want domain.ErrConflict", err)
	}
}
