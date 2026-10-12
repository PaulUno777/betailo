package httpserver

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/account"
	"github.com/pauluno777/betailo/backend/application/usecase/admin"
	"github.com/pauluno777/betailo/backend/application/usecase/auth"
	"github.com/pauluno777/betailo/backend/application/usecase/cycle"
	"github.com/pauluno777/betailo/backend/application/usecase/dashboard"
	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/application/usecase/favorite"
	"github.com/pauluno777/betailo/backend/application/usecase/forum"
	"github.com/pauluno777/betailo/backend/application/usecase/inbox"
	"github.com/pauluno777/betailo/backend/application/usecase/listing"
	"github.com/pauluno777/betailo/backend/application/usecase/report"
	"github.com/pauluno777/betailo/backend/application/usecase/reservation"
	"github.com/pauluno777/betailo/backend/application/usecase/review"
	"github.com/pauluno777/betailo/backend/application/usecase/trust"
	"github.com/pauluno777/betailo/backend/domain"
	"github.com/pauluno777/betailo/backend/infrastructure/clock"
	"github.com/pauluno777/betailo/backend/infrastructure/jwt"
	"github.com/pauluno777/betailo/backend/infrastructure/security"
)

// These HTTP tests exercise the real request pipeline — JSON decoding,
// the real usecases, real bcrypt hashing and real JWT signing/parsing —
// with only persistence faked (an in-memory map instead of MongoDB).
// That's deliberate: it's the wiring in server.go, the middleware, and
// the handlers' status-code/JSON-shape behavior that's under test here,
// on top of infrastructure that's already covered on its own
// (infrastructure/security, infrastructure/jwt) and usecases already
// covered with fakes (application/usecase/auth) — this is the layer
// none of those touch.

type fakeUserRepo struct {
	byEmail map[string]*domain.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byEmail: map[string]*domain.User{}}
}

func (f *fakeUserRepo) Create(_ context.Context, u *domain.User) error {
	if _, exists := f.byEmail[u.Email]; exists {
		return domain.ErrEmailTaken
	}
	if u.ID.IsZero() {
		u.ID = bson.NewObjectID()
	}
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) FindByID(_ context.Context, id bson.ObjectID) (*domain.User, error) {
	for _, u := range f.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUserRepo) Update(_ context.Context, u *domain.User) error {
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserRepo) List(_ context.Context) ([]*domain.User, error) {
	out := make([]*domain.User, 0, len(f.byEmail))
	for _, u := range f.byEmail {
		out = append(out, u)
	}
	return out, nil
}

// fakeRefreshTokenRepo is an in-memory ports.RefreshTokenRepository,
// keyed by hash like the real Mongo repository's unique index — used
// here (rather than the application/usecase/auth package's own fake)
// because this layer's tests go through the real HTTP handlers, not
// the usecase directly.
type fakeRefreshTokenRepo struct {
	byHash map[string]*ports.RefreshToken
}

func newFakeRefreshTokenRepo() *fakeRefreshTokenRepo {
	return &fakeRefreshTokenRepo{byHash: map[string]*ports.RefreshToken{}}
}

func (f *fakeRefreshTokenRepo) Create(_ context.Context, t *ports.RefreshToken) error {
	f.byHash[t.TokenHash] = t
	return nil
}

func (f *fakeRefreshTokenRepo) FindByHash(_ context.Context, tokenHash string) (*ports.RefreshToken, error) {
	t, ok := f.byHash[tokenHash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

func (f *fakeRefreshTokenRepo) Rotate(_ context.Context, oldID bson.ObjectID, next *ports.RefreshToken) error {
	var old *ports.RefreshToken
	for _, t := range f.byHash {
		if t.ID == oldID {
			old = t
			break
		}
	}
	if old == nil || old.RevokedAt != nil {
		return domain.ErrConflict
	}
	now := next.IssuedAt
	old.RevokedAt = &now
	old.RevokedReason = "rotated"
	old.ReplacedBy = &next.ID
	f.byHash[next.TokenHash] = next
	return nil
}

func (f *fakeRefreshTokenRepo) RevokeFamily(_ context.Context, familyID bson.ObjectID, reason string, revokedAt time.Time) error {
	for _, t := range f.byHash {
		if t.FamilyID == familyID && t.RevokedAt == nil {
			t.RevokedAt = &revokedAt
			t.RevokedReason = reason
		}
	}
	return nil
}

func (f *fakeRefreshTokenRepo) RevokeAllForUser(_ context.Context, userID bson.ObjectID, reason string, revokedAt time.Time) error {
	for _, t := range f.byHash {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &revokedAt
			t.RevokedReason = reason
		}
	}
	return nil
}

// testFakes is every in-memory store newTestRouterAndFakes wires up —
// exposed so a test can reach in directly when the HTTP surface alone
// can't set up a scenario (promoting a user to admin: registration
// itself refuses the admin role, on purpose — see
// TestRegister_HTTP_RejectsAdminRole).
type testFakes struct {
	users         *fakeUserRepo
	refreshTokens *fakeRefreshTokenRepo
	farms         *fakeFarmRepo
	cycles        *fakeCycleStore
	listings      *fakeListingStore
	reservations  *fakeReservationRepo
	reviews       *fakeReviewRepo
	reports       *fakeReportRepo
	forumPosts    *fakeForumRepo
	storage       fakeObjectStorage
	favorites     *fakeFavoriteRepo
	sequencer     *fakeSequencer
	notifications *fakeNotificationRepo
	signer        *jwt.Signer
	clock         clock.System
}

// promoteToAdmin flips a registered user's role directly in the fake
// store — the only way to get an admin account in these tests, since
// POST /auth/register rejects the admin role by design. The caller
// must log in again afterward: a token already issued still carries the
// old role, since Login signs the role at issuance time, not read live
// from the store on every request.
func (tf *testFakes) setDisplayName(userID, name string) {
	id, _ := bson.ObjectIDFromHex(userID)
	for _, u := range tf.users.byEmail {
		if u.ID == id {
			u.DisplayName = name
		}
	}
}

func (tf *testFakes) promoteToAdmin(userID string) {
	id, _ := bson.ObjectIDFromHex(userID)
	for _, u := range tf.users.byEmail {
		if u.ID == id {
			u.Role = domain.RoleAdmin
		}
	}
}

// newTestRouter builds a full router with real bcrypt (at the fast
// MinCost, so tests stay quick), a real JWT signer, and the real system
// clock — the signer's exp claim is validated against actual wall-clock
// time by the JWT library regardless of what Clock a usecase used to
// compute it, so these HTTP tests can't use a fixed/fake clock the way
// application/usecase/auth's own tests do. Persistence is the only
// thing faked, via in-memory stores.
func newTestRouter() http.Handler {
	router, _ := newTestRouterAndFakes()
	return router
}

func newTestRouterAndFakes() (http.Handler, *testFakes) {
	tf := &testFakes{
		users: newFakeUserRepo(), refreshTokens: newFakeRefreshTokenRepo(), farms: newFakeFarmRepo(), cycles: newFakeCycleStore(),
		listings: newFakeListingStore(), reservations: newFakeReservationRepo(),
		reviews: newFakeReviewRepo(), reports: newFakeReportRepo(), forumPosts: newFakeForumRepo(), favorites: newFakeFavoriteRepo(), sequencer: &fakeSequencer{}, notifications: &fakeNotificationRepo{},
		signer: &jwt.Signer{Secret: []byte("test-secret-do-not-use-in-production")},
		clock:  clock.System{},
	}
	hasher := security.BcryptHasher{Cost: bcrypt.MinCost}
	payments := fakePaymentProvider{}

	search := &fakeListingSearch{store: tf.listings, cycles: tf.cycles, farms: tf.farms, users: tf.users, reviews: tf.reviews}

	trustRecompute := &trust.Recompute{
		Users: tf.users, Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings,
		Reservations: tf.reservations, Reviews: tf.reviews, Reports: tf.reports, Clock: tf.clock,
	}

	parties := reservation.Parties{Listings: tf.listings, Cycles: tf.cycles, Farms: tf.farms}

	deps := Dependencies{
		Register: &auth.Register{Users: tf.users, Hasher: hasher, Clock: tf.clock},
		Login: &auth.Login{
			Users: tf.users, RefreshTokens: tf.refreshTokens, Hasher: hasher, Signer: tf.signer,
			Clock: tf.clock, AccessTTL: time.Hour, RefreshTTL: 48 * time.Hour,
		},
		RefreshToken: &auth.RefreshToken{
			Users: tf.users, RefreshTokens: tf.refreshTokens, Signer: tf.signer,
			Clock: tf.clock, AccessTTL: time.Hour, RefreshTTL: 48 * time.Hour,
		},
		Logout:         &auth.Logout{RefreshTokens: tf.refreshTokens, Clock: tf.clock},
		Signer:         tf.signer,
		Cookies:        CookieConfig{Secure: true, SameSite: http.SameSiteNoneMode},
		GetProfile:     &account.GetProfile{Users: tf.users},
		UpdateProfile:  &account.UpdateProfile{Users: tf.users},
		ChangePassword: &auth.ChangePassword{Users: tf.users, Hasher: hasher, RefreshTokens: tf.refreshTokens, Clock: tf.clock},
		BecomeSeller:   &farm.BecomeSeller{Users: tf.users, Farms: tf.farms, Trust: trustRecompute, Tx: fakeTxManager{}, Clock: tf.clock},

		CreateFarm: &farm.CreateFarm{Farms: tf.farms, Trust: trustRecompute, Clock: tf.clock},
		GetFarmProfile: &farm.GetProfile{
			Farms: tf.farms, Users: tf.users, Reviews: tf.reviews, Cycles: tf.cycles, Listings: tf.listings, Reservations: tf.reservations,
		},
		ListMyFarms: &farm.ListMyFarms{Farms: tf.farms},
		UpdateFarm:  &farm.UpdateFarm{Farms: tf.farms},
		DeleteFarm:  &farm.DeleteFarm{Farms: tf.farms, Cycles: tf.cycles},

		CreateCycle: &cycle.CreateCycle{Farms: tf.farms, Cycles: tf.cycles, Sequencer: tf.sequencer},
		GetCycle:    &cycle.GetCycle{Cycles: tf.cycles, Clock: tf.clock},
		AddHealthEvent: &cycle.AddHealthEvent{
			Farms: tf.farms, Cycles: tf.cycles, Clock: tf.clock,
			Listings: tf.listings, Reservations: tf.reservations, Notifications: tf.notifications,
		},
		ListCyclesByFarm:  &cycle.ListCyclesByFarm{Cycles: tf.cycles},
		UpdateCycleStatus: &cycle.UpdateCycleStatus{Farms: tf.farms, Cycles: tf.cycles},
		DeleteCycle:       &cycle.DeleteCycle{Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings},

		CreateListing:      &listing.CreateListing{Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings},
		GetListing:         &listing.GetListing{Listings: tf.listings, Cycles: tf.cycles, Farms: tf.farms},
		QuickCreateListing: &listing.QuickCreate{Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings, Sequencer: tf.sequencer, Tx: fakeTxManager{}, Clock: tf.clock},
		PublishListing:     &listing.Publish{Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings},
		ListFarmListings:   &listing.ListFarmListings{Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings, Reservations: tf.reservations},
		PriceHint:          &listing.PriceHint{Search: search},
		UpdateCycle:        &cycle.UpdateCycle{Farms: tf.farms, Cycles: tf.cycles, Editor: tf.cycles},
		GetListingDetail: &listing.GetDetail{
			Listings: tf.listings, Cycles: tf.cycles, Farms: tf.farms, Users: tf.users, Reviews: tf.reviews,
			Reservations: tf.reservations, Favorites: tf.favorites, ReviewStore: tf.reviews, Clock: tf.clock,
		},
		UpdateListing:       &listing.UpdateListing{Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings},
		UnpublishListing:    &listing.UnpublishListing{Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings},
		PresignListingPhoto: &listing.PresignListingPhoto{Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings, Storage: tf.storage},
		SetListingPhotos:    &listing.SetListingPhotos{Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings},
		SearchListings:      &listing.SearchListings{Search: search, Favorites: tf.favorites},
		AddFavorite:         &favorite.Add{Listings: tf.listings, Favorites: tf.favorites, Clock: tf.clock},
		RemoveFavorite:      &favorite.Remove{Favorites: tf.favorites},
		ListMyFavorites:     &favorite.ListMine{Favorites: tf.favorites, Search: search},
		Storage:             tf.storage,

		ReserveCycle: &reservation.ReserveCycle{
			Listings: tf.listings, Cycles: tf.cycles, Farms: tf.farms, Sequencer: tf.sequencer, Reservations: tf.reservations,
			Payments: payments, Tx: fakeTxManager{}, Clock: tf.clock, Notifications: tf.notifications,
		},
		CancelReservation: &reservation.CancelReservation{
			Reservations: tf.reservations, Listings: tf.listings, Cycles: tf.cycles, Tx: fakeTxManager{}, Clock: tf.clock,
			Parties: parties, Notifications: tf.notifications,
		},
		QuoteReservation:         &reservation.Quote{Listings: tf.listings, Cycles: tf.cycles, Farms: tf.farms},
		ListNotifications:        &inbox.List{Inbox: tf.notifications},
		UnreadNotifications:      &inbox.UnreadCount{Inbox: tf.notifications},
		MarkNotificationRead:     &inbox.MarkRead{Inbox: tf.notifications, Clock: tf.clock},
		MarkAllNotificationsRead: &inbox.MarkAllRead{Inbox: tf.notifications, Clock: tf.clock},
		PayReservation: &reservation.PayReservation{
			Reservations: tf.reservations, Payments: payments, Clock: tf.clock, Parties: parties, Notifications: tf.notifications,
		},
		MarkDelivered: &reservation.MarkDelivered{
			Reservations: tf.reservations, Listings: tf.listings, Cycles: tf.cycles, Farms: tf.farms, Clock: tf.clock, Notifications: tf.notifications,
		},
		ConfirmDelivery: &reservation.ConfirmDelivery{
			Reservations: tf.reservations, Reports: tf.reports, Listings: tf.listings, Cycles: tf.cycles,
			Farms: tf.farms, Trust: trustRecompute, Clock: tf.clock, Notifications: tf.notifications,
		},
		OpenDispute: &reservation.OpenDispute{
			Reservations: tf.reservations, Reports: tf.reports, Clock: tf.clock, Parties: parties, Notifications: tf.notifications,
		},
		ListMyReservations: &reservation.ListMyReservations{
			Reservations: tf.reservations, Listings: tf.listings, Cycles: tf.cycles, Farms: tf.farms, Reports: tf.reports, Reviews: tf.reviews,
		},
		ListReservationsByFarm: &reservation.ListReservationsByFarm{
			Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings, Reservations: tf.reservations, Reports: tf.reports, Reviews: tf.reviews,
		},
		NavCounts: &reservation.NavCounts{
			Farms: tf.farms,
			Buyer: &reservation.ListMyReservations{
				Reservations: tf.reservations, Listings: tf.listings, Cycles: tf.cycles, Farms: tf.farms, Reports: tf.reports, Reviews: tf.reviews,
			},
			ForFarm: &reservation.ListReservationsByFarm{
				Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings, Reservations: tf.reservations, Reports: tf.reports, Reviews: tf.reviews,
			},
		},

		CreateReview: &review.CreateReview{
			Reservations: tf.reservations, Listings: tf.listings, Cycles: tf.cycles, Farms: tf.farms, Reviews: tf.reviews, Trust: trustRecompute,
			Clock: tf.clock, Notifications: tf.notifications,
		},
		ListFarmReviews:    &review.ListByFarm{Reviews: tf.reviews, Store: tf.reviews, Users: tf.users, Reservations: tf.reservations},
		PresignReviewPhoto: &review.PresignReviewPhoto{Reservations: tf.reservations, Storage: tf.storage},
		MarkReviewHelpful:  &review.MarkHelpful{Store: tf.reviews},
		ReplyToReview:      &review.Reply{Store: tf.reviews, Farms: tf.farms, Clock: tf.clock, Notifications: tf.notifications},

		CreateReport: &report.CreateReport{Reports: tf.reports, Listings: tf.listings, Clock: tf.clock},

		CreateForumPost: &forum.CreatePost{Forum: tf.forumPosts, Clock: tf.clock},
		ListForumPosts:  &forum.ListPosts{Forum: tf.forumPosts},
		GetForumPost:    &forum.GetPost{Forum: tf.forumPosts},
		AddForumReply:   &forum.AddReply{Forum: tf.forumPosts, Clock: tf.clock, Notifications: tf.notifications},
		MarkBestAnswer:  &forum.MarkBestAnswer{Forum: tf.forumPosts, Clock: tf.clock, Notifications: tf.notifications},

		SellerDashboard: &dashboard.SellerDashboard{
			Farms: tf.farms, Cycles: tf.cycles, Listings: tf.listings, Reservations: tf.reservations, Users: tf.users, Clock: tf.clock,
		},

		AdminOverview:     &admin.Overview{Cycles: tf.cycles, Listings: tf.listings, Reservations: tf.reservations, Reports: tf.reports, Clock: tf.clock},
		AdminListUsers:    &admin.ListUsers{Users: tf.users},
		AdminVerifyUser:   &admin.VerifyUser{Users: tf.users, Trust: trustRecompute},
		AdminSuspendUser:  &admin.SuspendUser{Users: tf.users},
		AdminListCycles:   &admin.ListAllCycles{Cycles: tf.cycles},
		AdminListListings: &admin.ListAllListings{Cycles: tf.cycles, Listings: tf.listings},
		AdminListEscrow:   &admin.ListEscrowTransactions{Cycles: tf.cycles, Listings: tf.listings, Reservations: tf.reservations},
		AdminReleaseEscrow: &reservation.AdminReleaseEscrow{
			Reservations: tf.reservations, Listings: tf.listings, Cycles: tf.cycles, Farms: tf.farms, Trust: trustRecompute, Clock: tf.clock,
		},
		AdminRefundEscrow: &reservation.AdminRefundEscrow{
			Reservations: tf.reservations, Payments: payments, Clock: tf.clock, Parties: parties, Notifications: tf.notifications,
		},
		AdminListReports: &admin.ListReports{Reports: tf.reports},
		AdminGetReportDetail: &admin.GetReportDetail{
			Reports: tf.reports, Users: tf.users, Listings: tf.listings, Cycles: tf.cycles, Farms: tf.farms,
			Search: search, Clock: tf.clock,
		},
		AdminResolveReport: &admin.ResolveReport{Reports: tf.reports, Users: tf.users, Trust: trustRecompute},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(logger, deps), tf
}

// fakeTxManager just runs fn inline — no transaction semantics needed
// to exercise the router/handler layer itself.
type fakeTxManager struct{}

func (fakeTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
