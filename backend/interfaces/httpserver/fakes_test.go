package httpserver

import (
	"context"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// fakeFarmRepo, fakeCycleStore and fakeListingStore are in-memory
// doubles for ports.FarmRepository, ports.CycleStore and
// ports.ListingStore — used the same way fakeUserRepo is in
// router_test.go.

type fakeFarmRepo struct {
	byID map[bson.ObjectID]*domain.Farm
	// awaitReview leaves a new farm a draft, as in production. Off by
	// default: most tests are about what a seller does once staff
	// approved their farm, so a new farm is approved on the spot.
	awaitReview bool
}

func newFakeFarmRepo() *fakeFarmRepo {
	return &fakeFarmRepo{byID: map[bson.ObjectID]*domain.Farm{}}
}

func (f *fakeFarmRepo) Create(_ context.Context, farm *domain.Farm) error {
	if farm.ID.IsZero() {
		farm.ID = bson.NewObjectID()
	}
	if !f.awaitReview && farm.Status == domain.FarmDraft {
		farm.Status = domain.FarmApproved
	}
	f.byID[farm.ID] = farm
	return nil
}

func (f *fakeFarmRepo) ListByStatus(_ context.Context, status domain.FarmStatus) ([]*domain.Farm, error) {
	var out []*domain.Farm
	for _, farm := range f.byID {
		if status == "" || farm.Status == status {
			out = append(out, farm)
		}
	}
	return out, nil
}

func (f *fakeFarmRepo) FindByID(_ context.Context, id bson.ObjectID) (*domain.Farm, error) {
	farm, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return farm, nil
}

func (f *fakeFarmRepo) ListByOwner(_ context.Context, ownerID bson.ObjectID) ([]*domain.Farm, error) {
	var out []*domain.Farm
	for _, farm := range f.byID {
		if farm.OwnerID == ownerID {
			out = append(out, farm)
		}
	}
	return out, nil
}

func (f *fakeFarmRepo) Update(_ context.Context, farm *domain.Farm) error {
	if _, ok := f.byID[farm.ID]; !ok {
		return domain.ErrNotFound
	}
	f.byID[farm.ID] = farm
	return nil
}

func (f *fakeFarmRepo) Delete(_ context.Context, id bson.ObjectID) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

type fakeCycleStore struct {
	byID map[bson.ObjectID]*domain.Cycle
}

func newFakeCycleStore() *fakeCycleStore {
	return &fakeCycleStore{byID: map[bson.ObjectID]*domain.Cycle{}}
}

func (f *fakeCycleStore) Create(_ context.Context, c *domain.Cycle) error {
	if c.ID.IsZero() {
		c.ID = bson.NewObjectID()
	}
	f.byID[c.ID] = c
	return nil
}

func (f *fakeCycleStore) FindByID(_ context.Context, id bson.ObjectID) (*domain.Cycle, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

func (f *fakeCycleStore) ListByFarm(_ context.Context, farmID bson.ObjectID) ([]*domain.Cycle, error) {
	var out []*domain.Cycle
	for _, c := range f.byID {
		if c.FarmID == farmID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeCycleStore) ListByStatus(_ context.Context, status domain.CycleStatus) ([]*domain.Cycle, error) {
	var out []*domain.Cycle
	for _, c := range f.byID {
		if c.Status == status {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeCycleStore) UpdateStatus(_ context.Context, id bson.ObjectID, status domain.CycleStatus) error {
	c, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	c.Status = status
	return nil
}

func (f *fakeCycleStore) AppendHealthEvent(_ context.Context, id bson.ObjectID, e domain.HealthEvent, maxEntries int) error {
	c, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	if len(c.HealthLog) >= maxEntries {
		return domain.ErrConflict
	}
	// Store a fresh copy, like a database would, rather than mutating
	// the *Cycle a caller may still hold from FindByID.
	updated := *c
	updated.HealthLog = append(append([]domain.HealthEvent(nil), c.HealthLog...), e)
	f.byID[id] = &updated
	return nil
}

func (f *fakeCycleStore) Delete(_ context.Context, id bson.ObjectID) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

// DecrementAvailability/IncrementAvailability implement
// ports.CycleRepository (reservation's narrower slice) — the same
// concrete type satisfies both that and ports.CycleStore, exactly like
// infrastructure/mongodb.CycleRepository does.
func (f *fakeCycleStore) DecrementAvailability(_ context.Context, cycleID bson.ObjectID, quantity int) error {
	c, ok := f.byID[cycleID]
	if !ok {
		return domain.ErrNotFound
	}
	if c.AvailableQuantity < quantity {
		return domain.ErrInsufficientCapacity
	}
	c.AvailableQuantity -= quantity
	return nil
}

func (f *fakeCycleStore) IncrementAvailability(_ context.Context, cycleID bson.ObjectID, quantity int) error {
	c, ok := f.byID[cycleID]
	if !ok {
		return domain.ErrNotFound
	}
	c.AvailableQuantity += quantity
	return nil
}

type fakeListingStore struct {
	byID map[bson.ObjectID]*domain.Listing
}

func newFakeListingStore() *fakeListingStore {
	return &fakeListingStore{byID: map[bson.ObjectID]*domain.Listing{}}
}

func (f *fakeListingStore) Create(_ context.Context, l *domain.Listing) error {
	if l.ID.IsZero() {
		l.ID = bson.NewObjectID()
	}
	f.byID[l.ID] = l
	return nil
}

func (f *fakeListingStore) FindByID(_ context.Context, id bson.ObjectID) (*domain.Listing, error) {
	l, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return l, nil
}

func (f *fakeListingStore) ListPublished(_ context.Context) ([]*domain.Listing, error) {
	var out []*domain.Listing
	for _, l := range f.byID {
		if l.Status == domain.ListingPublished {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f *fakeListingStore) FindByCycleID(_ context.Context, cycleID bson.ObjectID) (*domain.Listing, error) {
	for _, l := range f.byID {
		if l.CycleID != nil && *l.CycleID == cycleID {
			return l, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeListingStore) Update(_ context.Context, l *domain.Listing) error {
	if _, ok := f.byID[l.ID]; !ok {
		return domain.ErrNotFound
	}
	f.byID[l.ID] = l
	return nil
}

// CycleAndPrice implements ports.ListingRepository (reservation's
// narrower slice) — the same concrete type satisfies both that and
// ports.ListingStore, exactly like infrastructure/mongodb.ListingRepository
// does.
func (f *fakeListingStore) CycleAndPrice(_ context.Context, listingID bson.ObjectID) (bson.ObjectID, float64, error) {
	l, ok := f.byID[listingID]
	if !ok {
		return bson.ObjectID{}, 0, domain.ErrNotFound
	}
	if l.CycleID == nil {
		return bson.ObjectID{}, 0, domain.ErrInvalid
	}
	return *l.CycleID, l.UnitPrice, nil
}

// fakeListingSearch implements ports.ListingSearchRepository by joining
// the same in-memory stores the real Mongo aggregation joins
// (infrastructure/mongodb/listing_search.go) — so a router-level test
// can exercise real filter/sort behavior, not just "is it published".
// The one thing this can never stand in for is the aggregation query
// itself; that needs a real MongoDB instance (testcontainers), which
// this test suite doesn't have.
type fakeListingSearch struct {
	store   *fakeListingStore
	cycles  *fakeCycleStore
	farms   *fakeFarmRepo
	users   *fakeUserRepo
	reviews *fakeReviewRepo
}

func (f *fakeListingSearch) Search(_ context.Context, c ports.ListingSearchCriteria) (ports.ListingSearchResult, error) {
	var items []ports.ListingSearchItem
	for _, l := range f.store.byID {
		if l.Status != domain.ListingPublished || l.Flagged {
			continue
		}
		if len(c.IDs) > 0 && !slices.Contains(c.IDs, l.ID) {
			continue
		}
		if c.Query != "" && !containsFold(l.Title, c.Query) {
			continue
		}
		if c.MinPrice > 0 && l.UnitPrice < c.MinPrice {
			continue
		}
		if c.MaxPrice > 0 && l.UnitPrice > c.MaxPrice {
			continue
		}
		if l.CycleID == nil {
			continue
		}
		cyc, ok := f.cycles.byID[*l.CycleID]
		if !ok {
			continue
		}
		if len(c.Species) > 0 && !slices.Contains(c.Species, cyc.Species) {
			continue
		}
		if len(c.CycleTypes) > 0 && !slices.ContainsFunc(c.CycleTypes, func(t string) bool { return strings.EqualFold(t, cyc.Type) }) {
			continue
		}
		statuses := c.Statuses
		if len(statuses) == 0 && len(c.IDs) == 0 {
			statuses = []domain.CycleStatus{domain.CycleStatusPlanned, domain.CycleStatusInProgress}
		}
		if len(statuses) > 0 && !slices.Contains(statuses, cyc.Status) {
			continue
		}
		farmVal, ok := f.farms.byID[cyc.FarmID]
		if !ok {
			continue
		}
		if c.Country != "" && farmVal.Country != c.Country {
			continue
		}
		if c.Region != "" && farmVal.Region != c.Region {
			continue
		}
		if !farmVal.Status.CanSell() {
			continue
		}
		owner, _ := f.users.FindByID(context.Background(), farmVal.OwnerID)
		verified := !farmVal.Verification.AutoApproved
		if c.VerifiedOnly && !verified {
			continue
		}
		trustScore := 0.0
		if owner != nil {
			trustScore = owner.TrustScore
		}
		summary, _ := f.reviews.Summary(context.Background(), farmVal.ID)

		items = append(items, ports.ListingSearchItem{
			Listing: l, Species: cyc.Species, CycleType: cyc.Type, CycleStatus: cyc.Status,
			StartsAt: cyc.StartsAt, EndsAt: cyc.EndsAt, Capacity: cyc.Capacity, Available: cyc.AvailableQuantity,
			FarmID: farmVal.ID, FarmName: farmVal.Name, FarmLocation: farmVal.Location,
			FarmCountry: farmVal.Country, FarmRegion: farmVal.Region,
			FarmVerified: verified, FarmTrustScore: trustScore,
			ReviewAverage: summary.Average, ReviewCount: summary.Count,
		})
	}

	switch c.Sort {
	case "price_asc":
		sortItems(items, func(a, b ports.ListingSearchItem) bool { return a.Listing.UnitPrice < b.Listing.UnitPrice })
	case "price_desc":
		sortItems(items, func(a, b ports.ListingSearchItem) bool { return a.Listing.UnitPrice > b.Listing.UnitPrice })
	default:
		sortItems(items, func(a, b ports.ListingSearchItem) bool { return a.StartsAt.Before(b.StartsAt) })
	}

	return ports.ListingSearchResult{Items: items, Total: len(items)}, nil
}

func containsFold(haystack, needle string) bool {
	return len(needle) == 0 || bytesContainsFold(haystack, needle)
}

func bytesContainsFold(haystack, needle string) bool {
	h, n := []rune(haystack), []rune(needle)
	toLower := func(rs []rune) []rune {
		out := make([]rune, len(rs))
		for i, r := range rs {
			if r >= 'A' && r <= 'Z' {
				r += 'a' - 'A'
			}
			out[i] = r
		}
		return out
	}
	h, n = toLower(h), toLower(n)
	if len(n) > len(h) {
		return false
	}
	for i := 0; i+len(n) <= len(h); i++ {
		if string(h[i:i+len(n)]) == string(n) {
			return true
		}
	}
	return false
}

func sortItems(items []ports.ListingSearchItem, less func(a, b ports.ListingSearchItem) bool) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && less(items[j], items[j-1]); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

// fakeReservationRepo implements ports.ReservationRepository — the full
// surface, not reservation's own narrower fakes (fakeListings/
// fakeCycles in application/usecase/reservation's own tests), since
// router-level tests exercise every reservation/escrow endpoint through
// this one store.
type fakeReservationRepo struct {
	byID map[bson.ObjectID]*domain.Reservation
}

func newFakeReservationRepo() *fakeReservationRepo {
	return &fakeReservationRepo{byID: map[bson.ObjectID]*domain.Reservation{}}
}

func (f *fakeReservationRepo) Insert(_ context.Context, r *domain.Reservation) error {
	if r.ID.IsZero() {
		r.ID = bson.NewObjectID()
	}
	f.byID[r.ID] = r
	return nil
}

func (f *fakeReservationRepo) FindByID(_ context.Context, id bson.ObjectID) (*domain.Reservation, error) {
	r, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return r, nil
}

func (f *fakeReservationRepo) ListByBuyer(_ context.Context, buyerID bson.ObjectID) ([]*domain.Reservation, error) {
	var out []*domain.Reservation
	for _, r := range f.byID {
		if r.BuyerID == buyerID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeReservationRepo) ListByListingIDs(_ context.Context, listingIDs []bson.ObjectID) ([]*domain.Reservation, error) {
	set := map[bson.ObjectID]bool{}
	for _, id := range listingIDs {
		set[id] = true
	}
	var out []*domain.Reservation
	for _, r := range f.byID {
		if set[r.ListingID] {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeReservationRepo) Update(_ context.Context, r *domain.Reservation) error {
	if _, ok := f.byID[r.ID]; !ok {
		return domain.ErrNotFound
	}
	f.byID[r.ID] = r
	return nil
}

func (f *fakeReservationRepo) ListDeliveredBefore(_ context.Context, before time.Time) ([]*domain.Reservation, error) {
	var out []*domain.Reservation
	for _, r := range f.byID {
		if r.Status == domain.ReservationDelivered && r.DeliveredAt != nil && !r.DeliveredAt.After(before) &&
			r.Payment != nil && r.Payment.Status == domain.PaymentHeld {
			out = append(out, r)
		}
	}
	return out, nil
}

// fakePaymentProvider always succeeds — the sandbox provider's own
// behavior (see infrastructure/payment.Sandbox), reimplemented here so
// these tests don't depend on that package.
type fakePaymentProvider struct{}

func (fakePaymentProvider) Capture(_ context.Context, req ports.CaptureRequest) (string, error) {
	reference := req.Reference
	return "fake-ref-" + reference, nil
}

func (fakePaymentProvider) Refund(context.Context, string, float64) error { return nil }

// fakeObjectStorage returns deterministic fake URLs — no real bytes
// move anywhere in these tests, only the key/URL plumbing is exercised.
type fakeObjectStorage struct{}

func (fakeObjectStorage) PresignUpload(_ context.Context, key string, _ int64) (string, error) {
	return "https://fake-upload.example/" + key, nil
}

func (fakeObjectStorage) PresignDownload(_ context.Context, key string) (string, error) {
	return "https://fake-download.example/" + key, nil
}

// fakeReviewRepo implements ports.ReviewRepository.
type fakeReviewRepo struct {
	byID  map[bson.ObjectID]*domain.Review
	votes map[bson.ObjectID][]bson.ObjectID
}

func (f *fakeReviewRepo) ExistsForReservation(_ context.Context, reservationID bson.ObjectID) (bool, error) {
	for _, r := range f.byID {
		if r.ReservationID == reservationID {
			return true, nil
		}
	}
	return false, nil
}

func newFakeReviewRepo() *fakeReviewRepo {
	return &fakeReviewRepo{byID: map[bson.ObjectID]*domain.Review{}, votes: map[bson.ObjectID][]bson.ObjectID{}}
}

func (f *fakeReviewRepo) Insert(_ context.Context, r *domain.Review) error {
	for _, existing := range f.byID {
		if existing.ReservationID == r.ReservationID {
			return domain.ErrConflict
		}
	}
	if r.ID.IsZero() {
		r.ID = bson.NewObjectID()
	}
	f.byID[r.ID] = r
	return nil
}

func (f *fakeReviewRepo) ListByFarm(_ context.Context, farmID bson.ObjectID) ([]*domain.Review, error) {
	var out []*domain.Review
	for _, r := range f.byID {
		if r.FarmID == farmID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeReviewRepo) Summary(_ context.Context, farmID bson.ObjectID) (ports.ReviewSummary, error) {
	s := ports.ReviewSummary{Histogram: map[int]int{}}
	var sum float64
	for _, r := range f.byID {
		if r.FarmID != farmID {
			continue
		}
		s.Count++
		sum += float64(r.Rating)
		s.Histogram[r.Rating]++
	}
	if s.Count > 0 {
		s.Average = sum / float64(s.Count)
	}
	// Each criterion averages over the reviews that rated it, like the
	// Mongo aggregation.
	avg := func(get func(domain.ReviewCriteria) int) float64 {
		var total, n int
		for _, r := range f.byID {
			if v := get(r.Criteria); r.FarmID == farmID && v > 0 {
				total += v
				n++
			}
		}
		if n == 0 {
			return 0
		}
		return float64(total) / float64(n)
	}
	s.Criteria = ports.CriteriaAverages{
		Health:        avg(func(c domain.ReviewCriteria) int { return c.Health }),
		Conformity:    avg(func(c domain.ReviewCriteria) int { return c.Conformity }),
		Communication: avg(func(c domain.ReviewCriteria) int { return c.Communication }),
		Timeliness:    avg(func(c domain.ReviewCriteria) int { return c.Timeliness }),
	}
	return s, nil
}

// fakeReportRepo implements ports.ReportRepository.
type fakeReportRepo struct {
	byID map[bson.ObjectID]*domain.Report
}

func newFakeReportRepo() *fakeReportRepo {
	return &fakeReportRepo{byID: map[bson.ObjectID]*domain.Report{}}
}

func (f *fakeReportRepo) Insert(_ context.Context, r *domain.Report) error {
	if r.ID.IsZero() {
		r.ID = bson.NewObjectID()
	}
	f.byID[r.ID] = r
	return nil
}

func (f *fakeReportRepo) FindByID(_ context.Context, id bson.ObjectID) (*domain.Report, error) {
	r, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return r, nil
}

func (f *fakeReportRepo) CountRecentByTarget(_ context.Context, targetID bson.ObjectID, since time.Time) (int, error) {
	count := 0
	for _, r := range f.byID {
		if r.TargetID == targetID && !r.CreatedAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func (f *fakeReportRepo) ListByTarget(_ context.Context, targetID bson.ObjectID) ([]*domain.Report, error) {
	var out []*domain.Report
	for _, r := range f.byID {
		if r.TargetID == targetID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeReportRepo) List(_ context.Context, status domain.ReportStatus) ([]*domain.Report, error) {
	var out []*domain.Report
	for _, r := range f.byID {
		if status == "" || r.Status == status {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeReportRepo) Update(_ context.Context, r *domain.Report) error {
	if _, ok := f.byID[r.ID]; !ok {
		return domain.ErrNotFound
	}
	f.byID[r.ID] = r
	return nil
}

// fakeForumRepo implements ports.ForumRepository.
type fakeForumRepo struct {
	byID map[bson.ObjectID]*domain.ForumPost
}

func newFakeForumRepo() *fakeForumRepo {
	return &fakeForumRepo{byID: map[bson.ObjectID]*domain.ForumPost{}}
}

func (f *fakeForumRepo) Insert(_ context.Context, p *domain.ForumPost) error {
	if p.ID.IsZero() {
		p.ID = bson.NewObjectID()
	}
	f.byID[p.ID] = p
	return nil
}

func (f *fakeForumRepo) FindByID(_ context.Context, id bson.ObjectID) (*domain.ForumPost, error) {
	p, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (f *fakeForumRepo) List(_ context.Context, filter ports.ForumListFilter) ([]*domain.ForumPost, error) {
	var out []*domain.ForumPost
	for _, p := range f.byID {
		if filter.Category != "" && p.Category != filter.Category {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeForumRepo) AddReply(_ context.Context, postID bson.ObjectID, reply domain.ForumReply) error {
	p, ok := f.byID[postID]
	if !ok {
		return domain.ErrNotFound
	}
	p.Replies = append(p.Replies, reply)
	return nil
}

func (f *fakeForumRepo) SetBestReplyIndex(_ context.Context, postID bson.ObjectID, index int) error {
	p, ok := f.byID[postID]
	if !ok {
		return domain.ErrNotFound
	}
	p.BestReplyIndex = &index
	return nil
}

// fakeFavoriteRepo keeps saves in insertion order per user.
type fakeFavoriteRepo struct {
	byUser map[bson.ObjectID][]bson.ObjectID
}

func newFakeFavoriteRepo() *fakeFavoriteRepo {
	return &fakeFavoriteRepo{byUser: map[bson.ObjectID][]bson.ObjectID{}}
}

func (f *fakeFavoriteRepo) Add(_ context.Context, userID, listingID bson.ObjectID, _ time.Time) error {
	if !slices.Contains(f.byUser[userID], listingID) {
		f.byUser[userID] = append(f.byUser[userID], listingID)
	}
	return nil
}

func (f *fakeFavoriteRepo) Remove(_ context.Context, userID, listingID bson.ObjectID) error {
	f.byUser[userID] = slices.DeleteFunc(f.byUser[userID], func(id bson.ObjectID) bool { return id == listingID })
	return nil
}

func (f *fakeFavoriteRepo) ListListingIDs(_ context.Context, userID bson.ObjectID) ([]bson.ObjectID, error) {
	ids := slices.Clone(f.byUser[userID])
	slices.Reverse(ids) // most recently saved first
	return ids, nil
}

func (f *fakeFavoriteRepo) FavoritedAmong(_ context.Context, userID bson.ObjectID, listingIDs []bson.ObjectID) (map[bson.ObjectID]bool, error) {
	out := map[bson.ObjectID]bool{}
	for _, id := range listingIDs {
		if slices.Contains(f.byUser[userID], id) {
			out[id] = true
		}
	}
	return out, nil
}

// fakeSequencer counts per key, like the counters collection.
type fakeSequencer struct{ byKey map[string]int64 }

func (f *fakeSequencer) NextReference(_ context.Context, prefix string) (string, error) {
	if f.byKey == nil {
		f.byKey = map[string]int64{}
	}
	period := prefix + "-2610"
	f.byKey[period]++
	return domain.FormatReference(period, f.byKey[period]), nil
}

// fakeNotificationRepo is the outbox and the inbox at once, like the
// real notifications collection.
type fakeNotificationRepo struct{ all []*domain.Notification }

func (f *fakeNotificationRepo) Enqueue(_ context.Context, n *domain.Notification) error {
	if n.ID.IsZero() {
		n.ID = bson.NewObjectID()
	}
	f.all = append(f.all, n)
	return nil
}

func (f *fakeNotificationRepo) ListPending(context.Context) ([]*domain.Notification, error) {
	return nil, nil
}

func (f *fakeNotificationRepo) MarkSent(context.Context, bson.ObjectID, time.Time) error { return nil }

func (f *fakeNotificationRepo) ListByUser(_ context.Context, userID bson.ObjectID, limit int) ([]*domain.Notification, error) {
	var out []*domain.Notification
	for i := len(f.all) - 1; i >= 0 && len(out) < limit; i-- {
		if f.all[i].UserID == userID {
			out = append(out, f.all[i])
		}
	}
	return out, nil
}

func (f *fakeNotificationRepo) CountUnread(_ context.Context, userID bson.ObjectID) (int, error) {
	n := 0
	for _, x := range f.all {
		if x.UserID == userID && x.ReadAt == nil {
			n++
		}
	}
	return n, nil
}

func (f *fakeNotificationRepo) MarkRead(_ context.Context, userID, id bson.ObjectID, at time.Time) error {
	for _, x := range f.all {
		if x.ID == id && x.UserID == userID {
			x.ReadAt = &at
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeNotificationRepo) MarkAllRead(_ context.Context, userID bson.ObjectID, at time.Time) error {
	for _, x := range f.all {
		if x.UserID == userID && x.ReadAt == nil {
			x.ReadAt = &at
		}
	}
	return nil
}

// ReviewStore side of fakeReviewRepo.

func (f *fakeReviewRepo) ListPage(_ context.Context, farmID bson.ObjectID, filter ports.ReviewFilter, page, pageSize int) ([]*domain.Review, int, error) {
	var all []*domain.Review
	for _, r := range f.byID {
		if r.FarmID != farmID {
			continue
		}
		switch filter {
		case ports.ReviewFilterPhotos:
			if len(r.Photos) == 0 {
				continue
			}
		case ports.ReviewFilterFive:
			if r.Rating != 5 {
				continue
			}
		case ports.ReviewFilterCritical:
			if r.Rating > 3 {
				continue
			}
		}
		all = append(all, r)
	}
	slices.SortFunc(all, func(a, b *domain.Review) int { return strings.Compare(b.ID.Hex(), a.ID.Hex()) })
	start := min((page-1)*pageSize, len(all))
	end := min(start+pageSize, len(all))
	return all[start:end], len(all), nil
}

func (f *fakeReviewRepo) FindByID(_ context.Context, id bson.ObjectID) (*domain.Review, error) {
	r, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (f *fakeReviewRepo) SetReply(_ context.Context, id bson.ObjectID, reply domain.ReviewReply) error {
	r, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	if r.Reply != nil {
		return domain.ErrConflict
	}
	r.Reply = &reply
	return nil
}

func (f *fakeReviewRepo) AddHelpfulVote(_ context.Context, id, userID bson.ObjectID) (bool, error) {
	r, ok := f.byID[id]
	if !ok {
		return false, domain.ErrNotFound
	}
	if slices.Contains(f.votes[id], userID) {
		return false, nil
	}
	f.votes[id] = append(f.votes[id], userID)
	r.HelpfulCount++
	return true, nil
}

func (f *fakeReviewRepo) VotedBy(_ context.Context, userID bson.ObjectID, ids []bson.ObjectID) (map[bson.ObjectID]bool, error) {
	out := map[bson.ObjectID]bool{}
	for _, id := range ids {
		if slices.Contains(f.votes[id], userID) {
			out[id] = true
		}
	}
	return out, nil
}

func (f *fakeListingSearch) PublishedPrices(_ context.Context, species, region string, limit int) ([]float64, error) {
	var out []float64
	for _, l := range f.store.byID {
		if l.Status != domain.ListingPublished || l.Flagged || l.CycleID == nil {
			continue
		}
		c, ok := f.cycles.byID[*l.CycleID]
		if !ok || c.Species != species {
			continue
		}
		if region != "" {
			farm, ok := f.farms.byID[c.FarmID]
			if !ok || !containsFold(farm.Location, region) {
				continue
			}
		}
		out = append(out, l.UnitPrice)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

// CycleEditor side of fakeCycleStore.
func (f *fakeCycleStore) UpdateDetails(_ context.Context, id bson.ObjectID, d ports.CycleDetails, capacityDelta int) error {
	c, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	if c.AvailableQuantity+capacityDelta < 0 {
		return domain.ErrConflict
	}
	updated := *c
	updated.StartsAt, updated.EndsAt = d.StartsAt, d.EndsAt
	updated.MinPerReservation, updated.MaxPerReservation = d.MinPerReservation, d.MaxPerReservation
	updated.Capacity += capacityDelta
	updated.AvailableQuantity += capacityDelta
	f.byID[id] = &updated
	return nil
}

// fakeAuditLog keeps the journal in memory, newest last.
type fakeAuditLog struct {
	actions []*domain.AdminAction
}

func (f *fakeAuditLog) Record(_ context.Context, a *domain.AdminAction) error {
	f.actions = append(f.actions, a)
	return nil
}

func (f *fakeAuditLog) List(_ context.Context, limit int) ([]*domain.AdminAction, error) {
	out := make([]*domain.AdminAction, 0, len(f.actions))
	for i := len(f.actions) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, f.actions[i])
	}
	return out, nil
}

// has reports whether an action with this code was journaled.
func (f *fakeAuditLog) has(action string) bool {
	for _, a := range f.actions {
		if a.Action == action {
			return true
		}
	}
	return false
}
