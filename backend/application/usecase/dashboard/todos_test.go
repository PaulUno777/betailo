package dashboard

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestTodosFor(t *testing.T) {
	now := time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)
	held := &domain.PaymentInfo{Status: domain.PaymentHeld}
	stale := &domain.Cycle{ID: bson.NewObjectID(), Reference: "CYC-0918", Status: domain.CycleStatusInProgress, StartsAt: now.AddDate(0, 0, -20),
		HealthLog: []domain.HealthEvent{{Type: "Mirage", Date: now.AddDate(0, 0, -9)}}}
	fresh := &domain.Cycle{ID: bson.NewObjectID(), Status: domain.CycleStatusInProgress, StartsAt: now.AddDate(0, 0, -20),
		HealthLog: []domain.HealthEvent{{Type: "Pesée", Date: now.AddDate(0, 0, -2)}}}
	noBuyers := &domain.Cycle{ID: bson.NewObjectID(), Status: domain.CycleStatusInProgress, StartsAt: now.AddDate(0, 0, -30)}
	staleListing, freshListing := bson.NewObjectID(), bson.NewObjectID()
	byListing := map[bson.ObjectID]*domain.Cycle{staleListing: stale, freshListing: fresh}

	todos := todosFor([]*domain.Cycle{stale, fresh, noBuyers}, []*domain.Reservation{
		{ListingID: staleListing, Status: domain.ReservationConfirmed, Payment: held},
		{ListingID: freshListing, Status: domain.ReservationConfirmed, Payment: held},
		{ListingID: freshListing, Status: domain.ReservationPending, Payment: &domain.PaymentInfo{Status: domain.PaymentPending}},
	}, byListing, now)

	want := []Todo{
		{Kind: TodoMarkDelivered, Count: 2},
		{Kind: TodoHealthLog, CycleID: stale.ID, CycleRef: "CYC-0918"},
		{Kind: TodoAwaitingPayment, Count: 1},
	}
	if len(todos) != len(want) {
		t.Fatalf("todos = %+v, want %+v", todos, want)
	}
	for i := range want {
		if todos[i] != want[i] {
			t.Errorf("todo %d = %+v, want %+v", i, todos[i], want[i])
		}
	}
}
