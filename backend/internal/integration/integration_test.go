//go:build integration

// Package integration exercises the critical usecases against a real
// MongoDB replica set (transactions need one), started per test run with
// testcontainers. The handler tests use in-memory fakes, which can't
// prove what matters here: that the atomic decrement plus multi-document
// transaction really prevents overselling under concurrency, and that
// escrow transitions persist correctly.
//
// Run with: go test -race -tags integration ./internal/integration/...
package integration

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	tcmongo "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/pauluno777/betailo/backend/application/usecase/reservation"
	"github.com/pauluno777/betailo/backend/application/usecase/review"
	"github.com/pauluno777/betailo/backend/application/usecase/trust"
	"github.com/pauluno777/betailo/backend/domain"
	"github.com/pauluno777/betailo/backend/infrastructure/clock"
	"github.com/pauluno777/betailo/backend/infrastructure/mongodb"
	"github.com/pauluno777/betailo/backend/infrastructure/payment"
	"github.com/pauluno777/betailo/backend/internal/worker"
)

var client *mongo.Client

func TestMain(m *testing.M) {
	ctx := context.Background()
	container, err := tcmongo.Run(ctx, "mongo:7", tcmongo.WithReplicaSet("rs0"))
	if err != nil {
		panic("start mongo container: " + err.Error())
	}
	uri, err := container.ConnectionString(ctx)
	if err != nil {
		panic(err)
	}
	client, err = mongo.Connect(options.Client().ApplyURI(uri).SetDirect(true))
	if err != nil {
		panic(err)
	}

	code := m.Run()

	_ = client.Disconnect(ctx)
	_ = container.Terminate(ctx)
	os.Exit(code)
}

// env is one isolated database with every repository the tests touch,
// bootstrapped exactly like production (validators + indexes).
type env struct {
	users        *mongodb.UserRepository
	farms        *mongodb.FarmRepository
	cycles       *mongodb.CycleRepository
	listings     *mongodb.ListingRepository
	reservations *mongodb.ReservationRepository
	reviews      *mongodb.ReviewRepository
	reports      *mongodb.ReportRepository
	notifs       *mongodb.NotificationRepository
	favorites    *mongodb.FavoriteRepository
	counters     *mongodb.CounterRepository
	db           *mongo.Database
	tx           *mongodb.TxManager
	trust        *trust.Recompute
}

func newEnv(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	db := client.Database("it_" + bson.NewObjectID().Hex())
	t.Cleanup(func() { _ = db.Drop(context.Background()) })
	if err := mongodb.Bootstrap(ctx, db); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	e := &env{
		users:        &mongodb.UserRepository{Coll: db.Collection("users")},
		farms:        &mongodb.FarmRepository{Coll: db.Collection("farms")},
		cycles:       &mongodb.CycleRepository{Coll: db.Collection("cycles")},
		listings:     &mongodb.ListingRepository{Coll: db.Collection("listings")},
		reservations: &mongodb.ReservationRepository{Coll: db.Collection("reservations")},
		reviews:      &mongodb.ReviewRepository{Coll: db.Collection("reviews")},
		reports:      &mongodb.ReportRepository{Coll: db.Collection("reports")},
		notifs:       &mongodb.NotificationRepository{Coll: db.Collection("notifications")},
		favorites:    &mongodb.FavoriteRepository{Coll: db.Collection("favorites")},
		counters:     &mongodb.CounterRepository{Coll: db.Collection("counters")},
		db:           db,
		tx:           &mongodb.TxManager{Client: client},
	}
	e.trust = &trust.Recompute{
		Users: e.users, Farms: e.farms, Cycles: e.cycles, Listings: e.listings,
		Reservations: e.reservations, Reviews: e.reviews, Reports: e.reports, Clock: clock.System{},
	}
	return e
}

// seed creates a farmer, their farm, one in-progress cycle of the given
// capacity, and a published listing on it.
func (e *env) seed(t *testing.T, capacity int) (farmer *domain.User, cycle *domain.Cycle, listing *domain.Listing) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	farmer = &domain.User{ID: bson.NewObjectID(), Email: bson.NewObjectID().Hex() + "@example.test", PasswordHash: "x", Role: domain.RoleFarmer, DisplayName: "Ferme", CreatedAt: now}
	if err := e.users.Create(ctx, farmer); err != nil {
		t.Fatalf("create farmer: %v", err)
	}
	farm := &domain.Farm{ID: bson.NewObjectID(), OwnerID: farmer.ID, Name: "Ferme des Collines", Location: "Dschang", Species: []string{"poule"}, CreatedAt: now}
	if err := e.farms.Create(ctx, farm); err != nil {
		t.Fatalf("create farm: %v", err)
	}
	cycle = &domain.Cycle{
		ID: bson.NewObjectID(), FarmID: farm.ID, Species: "poule", Type: "couvee", Status: domain.CycleStatusInProgress,
		Capacity: capacity, AvailableQuantity: capacity, StartsAt: now.Add(24 * time.Hour),
	}
	if err := e.cycles.Create(ctx, cycle); err != nil {
		t.Fatalf("create cycle: %v", err)
	}
	listing = &domain.Listing{ID: bson.NewObjectID(), CycleID: &cycle.ID, Title: "Poussins Brahma", UnitPrice: 2500, Unit: "poussin", Status: domain.ListingPublished}
	if err := e.listings.Create(ctx, listing); err != nil {
		t.Fatalf("create listing: %v", err)
	}
	return farmer, cycle, listing
}

func (e *env) buyer(t *testing.T) *domain.User {
	t.Helper()
	u := &domain.User{ID: bson.NewObjectID(), Email: bson.NewObjectID().Hex() + "@example.test", PasswordHash: "x", Role: domain.RoleBuyer, DisplayName: "Acheteur", CreatedAt: time.Now().UTC()}
	if err := e.users.Create(context.Background(), u); err != nil {
		t.Fatalf("create buyer: %v", err)
	}
	return u
}

// TestReserveCycle_ConcurrentReservationsNeverOversell is the Phase 3
// DoD item: many buyers racing for the same cycle can never reserve
// more than its capacity, and the stored availability always equals
// capacity minus what was actually reserved.
func TestReserveCycle_ConcurrentReservationsNeverOversell(t *testing.T) {
	e := newEnv(t)
	const capacity, buyers, perBuyer = 50, 40, 3 // 120 requested for 50 places
	_, cycle, listing := e.seed(t, capacity)

	uc := &reservation.ReserveCycle{Listings: e.listings, Cycles: e.cycles, Farms: e.farms, Sequencer: e.counters, Reservations: e.reservations,
		Payments: payment.Sandbox{}, Tx: e.tx, Clock: clock.System{},
	}

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		rejected  int
		other     []error
	)
	start := make(chan struct{})
	for range buyers {
		buyerID := e.buyer(t).ID
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := uc.Execute(context.Background(), reservation.ReserveCycleInput{ListingID: listing.ID, BuyerID: buyerID, Quantity: perBuyer})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				succeeded++
			case errors.Is(err, domain.ErrInsufficientCapacity):
				rejected++
			default:
				other = append(other, err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if len(other) > 0 {
		t.Fatalf("unexpected errors: %v", other)
	}
	if want := capacity / perBuyer; succeeded != want {
		t.Errorf("succeeded = %d, want exactly %d (capacity %d / %d each)", succeeded, want, capacity, perBuyer)
	}
	if succeeded+rejected != buyers {
		t.Errorf("succeeded+rejected = %d, want %d", succeeded+rejected, buyers)
	}

	stored, err := e.cycles.FindByID(context.Background(), cycle.ID)
	if err != nil {
		t.Fatal(err)
	}
	reserved, err := e.reservations.ListByListingIDs(context.Background(), []bson.ObjectID{listing.ID})
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, r := range reserved {
		total += r.Quantity
	}
	if total > capacity {
		t.Fatalf("oversold: %d reserved for capacity %d", total, capacity)
	}
	if stored.AvailableQuantity != capacity-total {
		t.Errorf("availableQuantity = %d, want capacity-reserved = %d", stored.AvailableQuantity, capacity-total)
	}
	if len(reserved) != succeeded {
		t.Errorf("stored reservations = %d, want %d (a failed reserve must leave nothing behind)", len(reserved), succeeded)
	}
}

func TestEscrow_HeldThenReleasedOnConfirmation(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	farmer, _, listing := e.seed(t, 100)
	buyer := e.buyer(t)
	sys := clock.System{}

	reserve := &reservation.ReserveCycle{Listings: e.listings, Cycles: e.cycles, Farms: e.farms, Sequencer: e.counters, Reservations: e.reservations, Payments: payment.Sandbox{}, Tx: e.tx, Clock: sys}
	out, err := reserve.Execute(ctx, reservation.ReserveCycleInput{ListingID: listing.ID, BuyerID: buyer.ID, Quantity: 10, PayNow: true, PaymentMethod: "mtn_momo", PayerPhone: "690123456"})
	if err != nil || !out.PaymentCaptured {
		t.Fatalf("reserve+pay: captured=%v err=%v", out != nil && out.PaymentCaptured, err)
	}
	assertPayment(t, e, out.Reservation.ID, domain.ReservationConfirmed, domain.PaymentHeld)

	markDelivered := &reservation.MarkDelivered{Reservations: e.reservations, Listings: e.listings, Cycles: e.cycles, Farms: e.farms, Clock: sys}
	if _, err := markDelivered.Execute(ctx, reservation.MarkDeliveredInput{ReservationID: out.Reservation.ID, FarmOwnerID: farmer.ID}); err != nil {
		t.Fatalf("mark delivered: %v", err)
	}

	confirm := &reservation.ConfirmDelivery{Reservations: e.reservations, Reports: e.reports, Listings: e.listings, Cycles: e.cycles, Farms: e.farms, Trust: e.trust, Clock: sys}
	if _, err := confirm.Execute(ctx, reservation.ConfirmDeliveryInput{ReservationID: out.Reservation.ID, BuyerID: buyer.ID}); err != nil {
		t.Fatalf("confirm delivery: %v", err)
	}
	assertPayment(t, e, out.Reservation.ID, domain.ReservationDelivered, domain.PaymentReleased)
}

func TestEscrow_HeldThenRefundedByAdmin(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_, _, listing := e.seed(t, 100)
	buyer := e.buyer(t)
	sys := clock.System{}

	reserve := &reservation.ReserveCycle{Listings: e.listings, Cycles: e.cycles, Farms: e.farms, Sequencer: e.counters, Reservations: e.reservations, Payments: payment.Sandbox{}, Tx: e.tx, Clock: sys}
	out, err := reserve.Execute(ctx, reservation.ReserveCycleInput{ListingID: listing.ID, BuyerID: buyer.ID, Quantity: 5, PayNow: true, PaymentMethod: "orange_money", PayerPhone: "699123456"})
	if err != nil || !out.PaymentCaptured {
		t.Fatalf("reserve+pay: err=%v", err)
	}

	refund := &reservation.AdminRefundEscrow{Reservations: e.reservations, Payments: payment.Sandbox{}, Clock: sys}
	if _, err := refund.Execute(ctx, out.Reservation.ID); err != nil {
		t.Fatalf("refund: %v", err)
	}
	stored, err := e.reservations.FindByID(ctx, out.Reservation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Payment == nil || stored.Payment.Status != domain.PaymentRefunded {
		t.Fatalf("payment status = %+v, want %q", stored.Payment, domain.PaymentRefunded)
	}
}

func assertPayment(t *testing.T, e *env, id bson.ObjectID, wantStatus domain.ReservationStatus, wantPayment domain.PaymentStatus) {
	t.Helper()
	stored, err := e.reservations.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != wantStatus {
		t.Errorf("reservation status = %q, want %q", stored.Status, wantStatus)
	}
	if stored.Payment == nil || stored.Payment.Status != wantPayment {
		t.Errorf("payment = %+v, want status %q", stored.Payment, wantPayment)
	}
}

// TestTrust_RecomputedFromRealSignals is the Phase 4 DoD item: the
// score moves with actual persisted data (a released sale, then a
// verified 5-star review), not a constant.
func TestTrust_RecomputedFromRealSignals(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	farmer, _, listing := e.seed(t, 100)
	buyer := e.buyer(t)
	sys := clock.System{}

	score := func() float64 {
		t.Helper()
		u, err := e.users.FindByID(ctx, farmer.ID)
		if err != nil {
			t.Fatal(err)
		}
		return u.TrustScore
	}

	if err := e.trust.Execute(ctx, farmer.ID); err != nil {
		t.Fatal(err)
	}
	baseline := score()

	reserve := &reservation.ReserveCycle{Listings: e.listings, Cycles: e.cycles, Farms: e.farms, Sequencer: e.counters, Reservations: e.reservations, Payments: payment.Sandbox{}, Tx: e.tx, Clock: sys}
	out, err := reserve.Execute(ctx, reservation.ReserveCycleInput{ListingID: listing.ID, BuyerID: buyer.ID, Quantity: 10, PayNow: true, PaymentMethod: "mtn_momo", PayerPhone: "690123456"})
	if err != nil {
		t.Fatal(err)
	}
	markDelivered := &reservation.MarkDelivered{Reservations: e.reservations, Listings: e.listings, Cycles: e.cycles, Farms: e.farms, Clock: sys}
	if _, err := markDelivered.Execute(ctx, reservation.MarkDeliveredInput{ReservationID: out.Reservation.ID, FarmOwnerID: farmer.ID}); err != nil {
		t.Fatal(err)
	}
	confirm := &reservation.ConfirmDelivery{Reservations: e.reservations, Reports: e.reports, Listings: e.listings, Cycles: e.cycles, Farms: e.farms, Trust: e.trust, Clock: sys}
	if _, err := confirm.Execute(ctx, reservation.ConfirmDeliveryInput{ReservationID: out.Reservation.ID, BuyerID: buyer.ID}); err != nil {
		t.Fatal(err)
	}
	afterSale := score()
	if afterSale <= baseline {
		t.Fatalf("score after a released sale = %v, want > baseline %v", afterSale, baseline)
	}

	createReview := &review.CreateReview{Reservations: e.reservations, Listings: e.listings, Cycles: e.cycles, Farms: e.farms, Reviews: e.reviews, Trust: e.trust}
	if _, err := createReview.Execute(ctx, review.CreateReviewInput{ReservationID: out.Reservation.ID, AuthorID: buyer.ID, Rating: 5, Comment: "Parfait"}); err != nil {
		t.Fatalf("create review: %v", err)
	}
	if afterReview := score(); afterReview <= afterSale {
		t.Fatalf("score after a 5-star review = %v, want > %v", afterReview, afterSale)
	}
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

// TestWorker_TickAdvancesCyclesExpiresUnpaidAndDrainsOutbox runs the
// real worker loop (Phase 5 DoD): it advances cycle statuses, expires
// an unpaid reservation past its deadline (restoring stock), drains the
// notification that expiry enqueued, and stops promptly on cancel.
func TestWorker_TickAdvancesCyclesExpiresUnpaidAndDrainsOutbox(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_, cycle, listing := e.seed(t, 100)
	buyer := e.buyer(t)

	// A reservation made 25h ago is past the 24h unpaid grace period.
	past := fixedClock{t: now.Add(-reservation.UnpaidReservationGrace - time.Hour)}
	reserve := &reservation.ReserveCycle{Listings: e.listings, Cycles: e.cycles, Farms: e.farms, Sequencer: e.counters, Reservations: e.reservations, Payments: payment.Sandbox{}, Tx: e.tx, Clock: past}
	out, err := reserve.Execute(ctx, reservation.ReserveCycleInput{ListingID: listing.ID, BuyerID: buyer.ID, Quantity: 30})
	if err != nil {
		t.Fatal(err)
	}

	// One planned cycle that should have started, one in-progress cycle
	// that should have ended.
	farmID := cycle.FarmID
	toStart := &domain.Cycle{ID: bson.NewObjectID(), FarmID: farmID, Species: "porc", Type: "engraissement", Status: domain.CycleStatusPlanned, Capacity: 10, AvailableQuantity: 10, StartsAt: now.Add(-time.Hour)}
	ended := now.Add(-time.Minute)
	toEnd := &domain.Cycle{ID: bson.NewObjectID(), FarmID: farmID, Species: "porc", Type: "engraissement", Status: domain.CycleStatusInProgress, Capacity: 10, AvailableQuantity: 10, StartsAt: now.Add(-48 * time.Hour), EndsAt: &ended}
	for _, c := range []*domain.Cycle{toStart, toEnd} {
		if err := e.cycles.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
	}

	sys := clock.System{}
	w := &worker.Worker{
		Cycles:       e.cycles,
		ExpireUnpaid: &reservation.ExpireUnpaidReservations{Reservations: e.reservations, Listings: e.listings, Cycles: e.cycles, Tx: e.tx, Notifications: e.notifs, Clock: sys},
		AutoRelease: &reservation.AutoReleaseDeliveries{
			Reservations: e.reservations, Reports: e.reports, Listings: e.listings, Cycles: e.cycles,
			Farms: e.farms, Trust: e.trust, Notifications: e.notifs, Clock: sys,
		},
		Notifications: e.notifs,
		Clock:         sys,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Interval:      50 * time.Millisecond,
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { w.Run(runCtx); close(done) }()

	deadline := time.Now().Add(10 * time.Second)
	for {
		res, err := e.reservations.FindByID(ctx, out.Reservation.ID)
		if err != nil {
			t.Fatal(err)
		}
		pending, err := e.notifs.ListPending(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status == domain.ReservationCancelled && len(pending) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("worker didn't expire the reservation and drain the outbox in time: status=%q pending=%d", res.Status, len(pending))
		}
		time.Sleep(50 * time.Millisecond)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker.Run didn't return after cancel")
	}

	stored, err := e.cycles.FindByID(ctx, cycle.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.AvailableQuantity != stored.Capacity {
		t.Errorf("availableQuantity = %d, want %d restored after expiry", stored.AvailableQuantity, stored.Capacity)
	}
	for c, want := range map[bson.ObjectID]domain.CycleStatus{toStart.ID: domain.CycleStatusInProgress, toEnd.ID: domain.CycleStatusDone} {
		got, err := e.cycles.FindByID(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != want {
			t.Errorf("cycle %s status = %q, want %q", c.Hex(), got.Status, want)
		}
	}
}

// The health-log cap must hold under concurrency: it's one conditional
// update, so racing appends can never push the log past the limit.
func TestCycleRepository_AppendHealthEventCapIsAtomic(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_, cycle, _ := e.seed(t, 10)
	const limit = 5

	var wg sync.WaitGroup
	var mu sync.Mutex
	var ok, full int
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := e.cycles.AppendHealthEvent(ctx, cycle.ID, domain.HealthEvent{Type: "Pesée", Date: time.Now()}, limit)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case errors.Is(err, domain.ErrConflict):
				full++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	stored, err := e.cycles.FindByID(ctx, cycle.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ok != limit || full != 20-limit || len(stored.HealthLog) != limit {
		t.Fatalf("ok=%d full=%d stored=%d, want exactly %d appended", ok, full, len(stored.HealthLog), limit)
	}
	if err := e.cycles.AppendHealthEvent(ctx, bson.NewObjectID(), domain.HealthEvent{Type: "x", Date: time.Now()}, limit); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown cycle: error = %v, want ErrNotFound", err)
	}
}

func TestFavoriteRepository_IdempotentAndOrdered(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	user := bson.NewObjectID()
	a, b := bson.NewObjectID(), bson.NewObjectID()
	t0 := time.Now().UTC()

	for _, step := range []struct {
		id bson.ObjectID
		at time.Time
	}{{a, t0}, {b, t0.Add(time.Minute)}, {a, t0.Add(2 * time.Minute)}} { // re-saving a keeps its first timestamp
		if err := e.favorites.Add(ctx, user, step.id, step.at); err != nil {
			t.Fatal(err)
		}
	}
	ids, err := e.favorites.ListListingIDs(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != b || ids[1] != a {
		t.Fatalf("ids = %v, want [b a] (most recent first, no duplicate)", ids)
	}

	saved, err := e.favorites.FavoritedAmong(ctx, user, []bson.ObjectID{a, bson.NewObjectID()})
	if err != nil || !saved[a] || len(saved) != 1 {
		t.Fatalf("FavoritedAmong = %v, %v; want only a", saved, err)
	}

	if err := e.favorites.Remove(ctx, user, a); err != nil {
		t.Fatal(err)
	}
	if err := e.favorites.Remove(ctx, user, a); err != nil {
		t.Fatalf("removing twice should be a no-op, got %v", err)
	}
}

// References must stay unique under concurrent creation, and the
// startup backfill must give pre-existing documents one.
func TestCounters_ConcurrentReferencesAreUniqueAndBackfillFillsGaps(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := map[int64]bool{}
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := e.counters.Next(ctx, "RSV")
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if seen[n] {
				t.Errorf("duplicate counter value %d", n)
			}
			seen[n] = true
		}()
	}
	wg.Wait()
	if len(seen) != 50 {
		t.Fatalf("got %d distinct values, want 50", len(seen))
	}

	_, cycle, _ := e.seed(t, 10) // seeded straight through the repository: no reference
	if err := mongodb.BackfillReferences(ctx, e.db); err != nil {
		t.Fatal(err)
	}
	got, err := e.cycles.FindByID(ctx, cycle.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Reference != "CYC-0001" {
		t.Fatalf("backfilled reference = %q, want CYC-0001", got.Reference)
	}
	if err := mongodb.BackfillReferences(ctx, e.db); err != nil {
		t.Fatal(err)
	}
	if again, _ := e.cycles.FindByID(ctx, cycle.ID); again.Reference != "CYC-0001" {
		t.Fatalf("backfill isn't idempotent: %q", again.Reference)
	}
}

func TestNotificationInbox_NewestFirstUnreadAndScopedMarkRead(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	alice, bob := bson.NewObjectID(), bson.NewObjectID()
	t0 := time.Now().UTC()
	for i, u := range []bson.ObjectID{alice, alice, bob} {
		if err := e.notifs.Enqueue(ctx, &domain.Notification{
			UserID: u, Type: domain.NotifReservationPaid, Message: "x", Status: domain.NotificationPending,
			CreatedAt: t0.Add(time.Duration(i) * time.Minute), Params: map[string]string{"ref": "RSV-000" + string(rune('1'+i))},
		}); err != nil {
			t.Fatal(err)
		}
	}

	items, err := e.notifs.ListByUser(ctx, alice, 10)
	if err != nil || len(items) != 2 || items[0].Params["ref"] != "RSV-0002" {
		t.Fatalf("ListByUser = %+v, %v; want alice's two, newest first", items, err)
	}
	if err := e.notifs.MarkRead(ctx, bob, items[0].ID, t0); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("marking someone else's notification: %v, want ErrNotFound", err)
	}
	if err := e.notifs.MarkRead(ctx, alice, items[0].ID, t0); err != nil {
		t.Fatal(err)
	}
	if n, _ := e.notifs.CountUnread(ctx, alice); n != 1 {
		t.Fatalf("alice unread = %d, want 1", n)
	}
	if err := e.notifs.MarkAllRead(ctx, alice, t0); err != nil {
		t.Fatal(err)
	}
	if n, _ := e.notifs.CountUnread(ctx, alice); n != 0 {
		t.Fatalf("alice unread after read-all = %d, want 0", n)
	}
	if n, _ := e.notifs.CountUnread(ctx, bob); n != 1 {
		t.Fatalf("bob unread = %d, want 1 (read-all is per user)", n)
	}
}
