// Package app is the composition root: the one place in this codebase
// that knows about both the application layer (usecases, ports) and the
// concrete infrastructure (mongodb, objectstorage, jwt, security) at
// once, and wires them together. Everything else — domain, application,
// infrastructure, interfaces — depends only inward; app is where the
// dependency graph actually gets built (see cmd/api/main.go, which just
// calls Initialize/Bootstrap/Run).
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/account"
	"github.com/pauluno777/betailo/backend/application/usecase/admin"
	"github.com/pauluno777/betailo/backend/application/usecase/auth"
	"github.com/pauluno777/betailo/backend/application/usecase/catalog"
	"github.com/pauluno777/betailo/backend/application/usecase/cycle"
	"github.com/pauluno777/betailo/backend/application/usecase/dashboard"
	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/application/usecase/favorite"
	"github.com/pauluno777/betailo/backend/application/usecase/forum"
	"github.com/pauluno777/betailo/backend/application/usecase/inbox"
	"github.com/pauluno777/betailo/backend/application/usecase/listing"
	"github.com/pauluno777/betailo/backend/application/usecase/people"
	"github.com/pauluno777/betailo/backend/application/usecase/report"
	"github.com/pauluno777/betailo/backend/application/usecase/reservation"
	"github.com/pauluno777/betailo/backend/application/usecase/review"
	"github.com/pauluno777/betailo/backend/application/usecase/trust"
	"github.com/pauluno777/betailo/backend/domain"
	"github.com/pauluno777/betailo/backend/infrastructure/clock"
	"github.com/pauluno777/betailo/backend/infrastructure/jwt"
	"github.com/pauluno777/betailo/backend/infrastructure/mongodb"
	"github.com/pauluno777/betailo/backend/infrastructure/objectstorage"
	"github.com/pauluno777/betailo/backend/infrastructure/passkey"
	"github.com/pauluno777/betailo/backend/infrastructure/payment"
	"github.com/pauluno777/betailo/backend/infrastructure/security"
	"github.com/pauluno777/betailo/backend/infrastructure/smtp"
	"github.com/pauluno777/betailo/backend/infrastructure/totp"
	"github.com/pauluno777/betailo/backend/interfaces/httpserver"
	"github.com/pauluno777/betailo/backend/internal/worker"
)

// Application holds what Run needs. Its usecases (ReserveCycle,
// Register, Login, and whatever follows them) are exported so a future
// caller can take this struct and wire more routes to them without app
// needing to know about routing.
type Application struct {
	Logger       *slog.Logger
	ReserveCycle *reservation.ReserveCycle
	Register     *auth.Register
	Login        *auth.Login

	mongoClient     *mongo.Client
	mongoDatabase   *mongo.Database
	storageProvider *objectstorage.S3Provider
	server          *http.Server
	worker          *worker.Worker
	ensureRootAdmin *auth.EnsureRootAdmin
}

// Initialize connects to MongoDB and object storage and wires every
// usecase to its concrete infrastructure. It does not run the schema
// bootstrap or start serving — see Bootstrap and Run — so a caller can
// fail fast on either step independently, mirroring RunBootstrapSeed in
// larger sibling projects.
func Initialize(ctx context.Context) (*Application, func(), error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	client, database, err := connectMongo(ctx)
	if err != nil {
		return nil, func() {}, err
	}

	storageProvider, err := connectStorage(ctx)
	if err != nil {
		_ = client.Disconnect(context.Background())
		return nil, func() {}, err
	}

	signer, err := newTokenSigner()
	if err != nil {
		_ = client.Disconnect(context.Background())
		return nil, func() {}, err
	}

	users := &mongodb.UserRepository{Coll: database.Collection("users")}
	refreshTokens := &mongodb.RefreshTokenRepository{Coll: database.Collection("refresh_tokens")}
	farms := &mongodb.FarmRepository{Coll: database.Collection("farms")}
	cycles := &mongodb.CycleRepository{Coll: database.Collection("cycles")}
	listings := &mongodb.ListingRepository{Coll: database.Collection("listings")}
	reservations := &mongodb.ReservationRepository{Coll: database.Collection("reservations")}
	reviews := &mongodb.ReviewRepository{Coll: database.Collection("reviews")}
	reports := &mongodb.ReportRepository{Coll: database.Collection("reports")}
	forumPosts := &mongodb.ForumRepository{Coll: database.Collection("forum_posts")}
	notifications := &mongodb.NotificationRepository{Coll: database.Collection("notifications")}
	favorites := &mongodb.FavoriteRepository{Coll: database.Collection("favorites")}
	counters := &mongodb.CounterRepository{Coll: database.Collection("counters")}
	catalogRepo := &mongodb.CatalogRepository{Coll: database.Collection("catalog_versions")}
	// The reference data in force, re-read at most every 30s so a
	// publish on another instance is picked up.
	catalogSource := &catalog.Cache{Repo: catalogRepo, TTL: 30 * time.Second}
	reports.Sequencer = counters
	txManager := &mongodb.TxManager{Client: client}
	hasher := security.BcryptHasher{}
	systemClock := clock.System{}

	maxFarms := maxFarmsPerAccount()
	autoApproveFarms := os.Getenv("FARM_AUTO_APPROVE") == "true"
	if autoApproveFarms {
		logger.Warn("FARM_AUTO_APPROVE is on: farms are approved on submission without staff review. Test use only; turn it off before real sellers arrive.")
	}
	sandboxPayments := payment.Sandbox{}

	authTokens := &mongodb.AuthTokenRepository{Coll: database.Collection("auth_tokens")}
	siteURL := publicURL()
	mailer := newMailer(logger)
	// Passkeys are bound to the site's host name. Where there is none
	// to bind to (the site is reached by IP address), they are simply
	// not offered: the authenticator app still is.
	var passkeys ports.Passkeys
	if svc, err := passkey.New("Betailo", hostOf(siteURL), passkeyOrigins(siteURL)); err != nil {
		logger.Warn("passkeys are unavailable: set APP_PUBLIC_URL to the site's address, with a host name", slog.Any("err", err))
	} else {
		passkeys = svc
	}
	staffMFARequired := os.Getenv("STAFF_MFA_REQUIRED") != "false"
	if !staffMFARequired {
		logger.Warn("STAFF_MFA_REQUIRED is off: staff reach the back office without a second factor. Development and automated tests only.")
	}

	register := &auth.Register{Users: users, Hasher: hasher, Clock: systemClock}
	login := &auth.Login{
		Users: users, RefreshTokens: refreshTokens, Hasher: hasher, Signer: signer,
		Clock: systemClock, AccessTTL: accessTokenTTL(), RefreshTTL: refreshTokenTTL(),
		AuthTokens: authTokens,
	}
	secondFactor := &auth.SecondFactor{
		Users: users, RefreshTokens: refreshTokens, AuthTokens: authTokens, Hasher: hasher, Signer: signer,
		TOTP: &totp.Service{Issuer: "Betailo"}, Passkeys: passkeys, Clock: systemClock,
		AccessTTL: accessTokenTTL(), RefreshTTL: refreshTokenTTL(),
	}
	passwordReset := &auth.PasswordReset{
		Users: users, AuthTokens: authTokens, RefreshTokens: refreshTokens, Hasher: hasher, Mailer: mailer,
		Clock: systemClock, PublicURL: siteURL, Logger: logger,
	}
	refreshToken := &auth.RefreshToken{
		Users: users, RefreshTokens: refreshTokens, Signer: signer,
		Clock: systemClock, AccessTTL: accessTokenTTL(), RefreshTTL: refreshTokenTTL(),
	}
	logout := &auth.Logout{RefreshTokens: refreshTokens, Clock: systemClock}

	trustRecompute := &trust.Recompute{
		Users: users, Farms: farms, Cycles: cycles, Listings: listings,
		Reservations: reservations, Reviews: reviews, Reports: reports, Clock: systemClock,
	}

	parties := reservation.Parties{Listings: listings, Cycles: cycles, Farms: farms}
	reserveCycle := &reservation.ReserveCycle{
		Listings: listings, Cycles: cycles, Farms: farms, Sequencer: counters, Reservations: reservations,
		Payments: sandboxPayments, Tx: txManager, Clock: systemClock, Notifications: notifications,
	}
	cancelReservation := &reservation.CancelReservation{
		Reservations: reservations, Clock: systemClock, Parties: parties, Notifications: notifications,
	}
	payReservation := &reservation.PayReservation{
		Reservations: reservations, Listings: listings, Cycles: cycles, Payments: sandboxPayments, Clock: systemClock,
		Parties: parties, Notifications: notifications,
	}
	closeUnpaid := &reservation.CloseUnpaidForCycle{
		Listings: listings, Reservations: reservations, Notifications: notifications, Clock: systemClock,
	}
	markDelivered := &reservation.MarkDelivered{
		Reservations: reservations, Listings: listings, Cycles: cycles, Farms: farms, Clock: systemClock, Notifications: notifications,
	}
	confirmDelivery := &reservation.ConfirmDelivery{
		Reservations: reservations, Reports: reports, Listings: listings, Cycles: cycles,
		Farms: farms, Trust: trustRecompute, Clock: systemClock, Notifications: notifications,
	}
	openDispute := &reservation.OpenDispute{
		Reservations: reservations, Reports: reports, Clock: systemClock, Parties: parties, Notifications: notifications,
	}
	listMyReservations := &reservation.ListMyReservations{
		Reservations: reservations, Listings: listings, Cycles: cycles, Farms: farms, Reports: reports, Reviews: reviews,
	}
	listReservationsByFarm := &reservation.ListReservationsByFarm{
		Farms: farms, Cycles: cycles, Listings: listings, Reservations: reservations, Reports: reports, Reviews: reviews,
	}
	navCounts := &reservation.NavCounts{Farms: farms, Buyer: listMyReservations, ForFarm: listReservationsByFarm, Cycles: cycles}

	createReview := &review.CreateReview{
		Reservations: reservations, Listings: listings, Cycles: cycles, Farms: farms, Reviews: reviews, Trust: trustRecompute,
		Clock: systemClock, Notifications: notifications,
		Catalog: catalogSource,
	}
	listFarmReviews := &review.ListByFarm{Reviews: reviews, Store: reviews, Users: users, Reservations: reservations}

	createReport := &report.CreateReport{Reports: reports, Listings: listings, Clock: systemClock}

	createForumPost := &forum.CreatePost{Forum: forumPosts, Clock: systemClock, Catalog: catalogSource}
	listForumPosts := &forum.ListPosts{Forum: forumPosts}
	getForumPost := &forum.GetPost{Forum: forumPosts}
	addForumReply := &forum.AddReply{Forum: forumPosts, Clock: systemClock, Notifications: notifications}
	markBestAnswer := &forum.MarkBestAnswer{Forum: forumPosts, Clock: systemClock, Notifications: notifications}

	sellerDashboard := &dashboard.SellerDashboard{
		Farms: farms, Cycles: cycles, Listings: listings, Reservations: reservations, Users: users, Clock: systemClock,
		Reviews: reviews,
	}

	adminOverview := &admin.Overview{
		Cycles: cycles, Listings: listings, Reservations: reservations, Reports: reports, Farms: farms, Clock: systemClock,
		AutoApproveFarms: autoApproveFarms, MaxFarms: maxFarms,
	}
	adminListUsers := &admin.ListUsers{Users: users, Farms: farms}
	accountGate := &auth.AccountGate{Users: users, Clock: systemClock}
	staffInvitations := &auth.StaffInvitations{
		Users: users, Farms: farms, AuthTokens: authTokens, RefreshTokens: refreshTokens, Hasher: hasher, Signer: signer,
		Mailer: mailer, Clock: systemClock, PublicURL: siteURL, AccessTTL: accessTokenTTL(), RefreshTTL: refreshTokenTTL(),
		Forget: accountGate.Forget,
	}
	adminSuspendUser := &admin.SuspendUser{Users: users, Sessions: accountGate}
	adminJournal := &admin.Journal{
		Log: &mongodb.AuditRepository{Coll: database.Collection("admin_actions")}, Users: users, Clock: systemClock, Logger: logger,
	}
	adminListCycles := &admin.ListAllCycles{Cycles: cycles}
	adminListListings := &admin.ListAllListings{Cycles: cycles, Listings: listings}
	adminListEscrow := &admin.ListEscrowTransactions{Cycles: cycles, Listings: listings, Reservations: reservations, Farms: farms, Users: users}
	adminReleaseEscrow := &reservation.AdminReleaseEscrow{
		Reservations: reservations, Listings: listings, Cycles: cycles, Farms: farms, Trust: trustRecompute, Clock: systemClock,
		Notifications: notifications,
	}
	adminRefundEscrow := &reservation.AdminRefundEscrow{
		Reservations: reservations, Payments: sandboxPayments, Clock: systemClock, Parties: parties, Notifications: notifications,
	}
	adminListReports := &admin.ListReports{Reports: reports, Users: users, Listings: listings, Reservations: reservations, Forum: forumPosts}
	adminGetReportDetail := &admin.GetReportDetail{
		Reports: reports, Users: users, Listings: listings, Cycles: cycles, Farms: farms, Search: listings, Clock: systemClock,
		Reservations: reservations, Forum: forumPosts,
	}
	adminResolveReport := &admin.ResolveReport{
		Reports: reports, Users: users, Trust: trustRecompute, Sessions: accountGate, Listings: listings, Cycles: cycles, Farms: farms,
		Notifications: notifications, Clock: systemClock,
	}

	bgWorker := &worker.Worker{
		Cycles:      cycles,
		CloseUnpaid: closeUnpaid,
		AutoRelease: &reservation.AutoReleaseDeliveries{
			Reservations: reservations, Reports: reports, Listings: listings, Cycles: cycles,
			Farms: farms, Trust: trustRecompute, Notifications: notifications, Clock: systemClock,
		},
		Notifications: notifications,
		Clock:         systemClock,
		Logger:        logger,
		Users:         users,
		Mailer:        mailer,
		PublicURL:     siteURL,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr: ":" + port,
		Handler: httpserver.NewRouter(logger, httpserver.Dependencies{
			Register:         register,
			Login:            login,
			RefreshToken:     refreshToken,
			Logout:           logout,
			Signer:           signer,
			Accounts:         accountGate,
			StaffRoles:       accountGate,
			Cookies:          cookieConfig(),
			StaffMFARequired: staffMFARequired,
			SecondFactor:     secondFactor,
			StaffInvitations: staffInvitations,
			PasswordReset:    passwordReset,
			GetProfile:       &account.GetProfile{Users: users},
			UpdateProfile:    &account.UpdateProfile{Users: users},
			PresignAvatar:    &account.PresignAvatar{Storage: storageProvider},
			SetAvatar:        &account.SetAvatar{Users: users},
			People:           &people.Directory{Users: users},
			Catalog:          catalogSource,
			ChangePassword:   &auth.ChangePassword{Users: users, Hasher: hasher, RefreshTokens: refreshTokens, Clock: systemClock},
			SellerAccess:     &farm.SellerAccess{Users: users, Farms: farms, MaxFarms: maxFarms},
			RateLimits:       httpserver.DefaultRateLimits(os.Getenv("TRUST_PROXY_HEADERS") == "true"),

			CreateFarm: &farm.CreateFarm{Users: users, Farms: farms, MaxFarms: maxFarms, Trust: trustRecompute, Clock: systemClock, Sequencer: counters, Catalog: catalogSource},
			GetFarmProfile: &farm.GetProfile{
				Farms: farms, Users: users, Reviews: reviews, Cycles: cycles, Listings: listings, Reservations: reservations,
			},
			ListMyFarms: &farm.ListMyFarms{Farms: farms},
			MyFarmCards: &farm.MyFarmCards{
				Farms: farms, Users: users, Reviews: reviews, Cycles: cycles, Listings: listings, Reservations: reservations,
			},
			UpdateFarm: &farm.UpdateFarm{Farms: farms, Catalog: catalogSource},
			DeleteFarm: &farm.DeleteFarm{Farms: farms, Cycles: cycles},

			PresignFarmProof:       &farm.PresignFarmProof{Farms: farms, Storage: storageProvider},
			WithdrawFarmSubmission: &farm.WithdrawFarmSubmission{Farms: farms},
			SubmitFarmForReview: &farm.SubmitFarmForReview{
				Farms: farms, Trust: trustRecompute, Notifications: notifications, Clock: systemClock, AutoApprove: autoApproveFarms,
			},
			ReviewFarm: &farm.ReviewFarm{Farms: farms, Trust: trustRecompute, Notifications: notifications, Clock: systemClock, Catalog: catalogSource},

			CreateCycle: &cycle.CreateCycle{Farms: farms, Cycles: cycles, Sequencer: counters, Catalog: catalogSource},
			GetCycle:    &cycle.GetCycle{Cycles: cycles, Clock: systemClock},
			AddHealthEvent: &cycle.AddHealthEvent{
				Farms: farms, Cycles: cycles, Clock: systemClock,
				Listings: listings, Reservations: reservations, Notifications: notifications,
			},
			ListCyclesByFarm:  &cycle.ListCyclesByFarm{Cycles: cycles},
			UpdateCycleStatus: &cycle.UpdateCycleStatus{Farms: farms, Cycles: cycles, Unpaid: closeUnpaid},
			DeleteCycle:       &cycle.DeleteCycle{Farms: farms, Cycles: cycles, Listings: listings},

			CreateListing:      &listing.CreateListing{Farms: farms, Cycles: cycles, Listings: listings},
			GetListing:         &listing.GetListing{Listings: listings, Cycles: cycles, Farms: farms},
			QuickCreateListing: &listing.QuickCreate{Farms: farms, Cycles: cycles, Listings: listings, Sequencer: counters, Tx: txManager, Clock: systemClock, Catalog: catalogSource},
			PublishListing:     &listing.Publish{Farms: farms, Cycles: cycles, Listings: listings},
			ListFarmListings:   &listing.ListFarmListings{Farms: farms, Cycles: cycles, Listings: listings, Reservations: reservations},
			PriceHint:          &listing.PriceHint{Search: listings, Catalog: catalogSource},
			UpdateCycle:        &cycle.UpdateCycle{Farms: farms, Cycles: cycles, Editor: cycles},
			GetListingDetail: &listing.GetDetail{
				Listings: listings, Cycles: cycles, Farms: farms, Users: users, Reviews: reviews,
				Reservations: reservations, Favorites: favorites, ReviewStore: reviews, Clock: systemClock,
			},
			UpdateListing:            &listing.UpdateListing{Farms: farms, Cycles: cycles, Listings: listings},
			UnpublishListing:         &listing.UnpublishListing{Farms: farms, Cycles: cycles, Listings: listings},
			PresignListingPhoto:      &listing.PresignListingPhoto{Farms: farms, Cycles: cycles, Listings: listings, Storage: storageProvider},
			SetListingPhotos:         &listing.SetListingPhotos{Farms: farms, Cycles: cycles, Listings: listings},
			SearchListings:           &listing.SearchListings{Search: listings, Favorites: favorites, Catalog: catalogSource},
			ListNotifications:        &inbox.List{Inbox: notifications},
			UnreadNotifications:      &inbox.UnreadCount{Inbox: notifications},
			MarkNotificationRead:     &inbox.MarkRead{Inbox: notifications, Clock: systemClock},
			MarkAllNotificationsRead: &inbox.MarkAllRead{Inbox: notifications, Clock: systemClock},
			AddFavorite:              &favorite.Add{Listings: listings, Favorites: favorites, Clock: systemClock},
			RemoveFavorite:           &favorite.Remove{Favorites: favorites},
			ListMyFavorites:          &favorite.ListMine{Favorites: favorites, Search: listings},
			Storage:                  storageProvider,

			ReserveCycle:           reserveCycle,
			CancelReservation:      cancelReservation,
			QuoteReservation:       &reservation.Quote{Listings: listings, Cycles: cycles, Farms: farms},
			PayReservation:         payReservation,
			MarkDelivered:          markDelivered,
			ConfirmDelivery:        confirmDelivery,
			OpenDispute:            openDispute,
			ListMyReservations:     listMyReservations,
			ListReservationsByFarm: listReservationsByFarm,
			NavCounts:              navCounts,

			CreateReview:       createReview,
			ListFarmReviews:    listFarmReviews,
			PresignReviewPhoto: &review.PresignReviewPhoto{Reservations: reservations, Storage: storageProvider},
			MarkReviewHelpful:  &review.MarkHelpful{Store: reviews},
			ReplyToReview:      &review.Reply{Store: reviews, Farms: farms, Clock: systemClock, Notifications: notifications},

			CreateReport: createReport,

			CreateForumPost: createForumPost,
			ListForumPosts:  listForumPosts,
			GetForumPost:    getForumPost,
			AddForumReply:   addForumReply,
			MarkBestAnswer:  markBestAnswer,

			SellerDashboard: sellerDashboard,

			AdminOverview:        adminOverview,
			AdminListUsers:       adminListUsers,
			AdminNavCounts:       &admin.CountNav{Farms: farms, Reports: reports},
			AdminSetFarmLimit:    &admin.SetFarmLimit{Users: users, Farms: farms, Clock: systemClock},
			AdminListFarmLimits:  &admin.ListFarmLimits{Users: users, Farms: farms, Default: maxFarms},
			AdminListFarms:       &admin.ListFarmsForReview{Farms: farms, Users: users},
			AdminFarmQueueCounts: &admin.CountFarmQueues{Farms: farms},
			AdminGetFarm: &admin.GetFarmForReview{
				Farms: farms, Users: users, Reservations: reservations, Reports: reports, Storage: storageProvider, MaxFarms: maxFarms,
			},
			AdminSuspendUser:     adminSuspendUser,
			AdminReactivateUser:  &admin.ReactivateUser{Users: users, Sessions: accountGate},
			AdminListStaff:       &admin.ListStaff{Users: users},
			AdminSetStaffRole:    &admin.SetStaffRole{Users: users, Farms: farms, Sessions: accountGate},
			AdminListAudit:       &admin.ListAudit{Log: adminJournal.Log},
			AdminJournal:         adminJournal,
			AdminListCycles:      adminListCycles,
			AdminListListings:    adminListListings,
			AdminListEscrow:      adminListEscrow,
			AdminReleaseEscrow:   adminReleaseEscrow,
			AdminRefundEscrow:    adminRefundEscrow,
			AdminListReports:     adminListReports,
			AdminGetReportDetail: adminGetReportDetail,
			AdminSaveReportNote:  &admin.SaveReportNote{Reports: reports},
			AdminResolveReport:   adminResolveReport,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	application := &Application{
		Logger:          logger,
		ReserveCycle:    reserveCycle,
		Register:        register,
		Login:           login,
		mongoClient:     client,
		mongoDatabase:   database,
		storageProvider: storageProvider,
		server:          server,
		worker:          bgWorker,
		ensureRootAdmin: &auth.EnsureRootAdmin{Users: users, Hasher: hasher, Clock: systemClock},
	}

	cleanup := func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Disconnect(disconnectCtx); err != nil {
			logger.Error("mongo disconnect", slog.Any("err", err))
		}
	}

	return application, cleanup, nil
}

// Bootstrap creates MongoDB collections/indexes/validators and the
// object storage bucket if they don't already exist. Idempotent — safe
// to call on every startup (see mongodb.Bootstrap and
// objectstorage.S3Provider.EnsureBucket).
func (a *Application) Bootstrap(ctx context.Context) error {
	if err := mongodb.Bootstrap(ctx, a.mongoDatabase); err != nil {
		return err
	}
	if err := mongodb.BackfillReferences(ctx, a.mongoDatabase); err != nil {
		return err
	}
	// The lists the platform shipped with become version 1 of the
	// reference data on a database that has none yet.
	if err := catalog.EnsureSeed(ctx, &mongodb.CatalogRepository{Coll: a.mongoDatabase.Collection("catalog_versions")}, time.Now()); err != nil {
		return err
	}
	if n, err := mongodb.BackfillFarmPlaces(ctx, a.mongoDatabase); err != nil {
		return err
	} else if n > 0 {
		a.Logger.Info("gave farms a country and region", "count", n)
	}
	if err := a.bootstrapRootAdmin(ctx); err != nil {
		return err
	}
	if n, err := mongodb.MigrateUnpaidToWaitingList(ctx, a.mongoDatabase); err != nil {
		return err
	} else if n > 0 {
		a.Logger.Info("moved unpaid reservations to the waiting list", "count", n)
	}
	a.Logger.Info("connected to MongoDB and bootstrapped collections")

	if err := a.storageProvider.EnsureBucket(ctx); err != nil {
		return err
	}
	a.Logger.Info("object storage provider ready")

	return nil
}

// bootstrapRootAdmin is the platform's day zero: the root account
// comes from SUPER_ADMIN_EMAIL / SUPER_ADMIN_PASSWORD, because nobody
// can register into the staff and only a super admin grants a role. A
// configured account that can't be set up stops the startup: running
// without the way into the back office the operator asked for is worse
// than not running. With nothing configured the API still starts, and
// says so loudly if nobody at all can run the back office.
func (a *Application) bootstrapRootAdmin(ctx context.Context) error {
	email := os.Getenv("SUPER_ADMIN_EMAIL")
	if email == "" {
		ok, err := a.ensureRootAdmin.HasSuperAdmin(ctx)
		if err != nil {
			return err
		}
		if !ok {
			a.Logger.Error("no super admin: nobody can open the back office, approve a farm or settle a payment. " +
				"Set SUPER_ADMIN_EMAIL and SUPER_ADMIN_PASSWORD, then restart")
		}
		return nil
	}
	outcome, err := a.ensureRootAdmin.Execute(ctx, email, os.Getenv("SUPER_ADMIN_PASSWORD"))
	if err != nil {
		return fmt.Errorf("root admin %s: %w", email, err)
	}
	if os.Getenv("SUPER_ADMIN_RESET_MFA") == "true" {
		root, err := a.ensureRootAdmin.Users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
		if err != nil {
			return fmt.Errorf("root admin %s: reset second factors: %w", email, err)
		}
		root.MFA = domain.MFA{}
		if err := a.ensureRootAdmin.Users.Update(ctx, root); err != nil {
			return fmt.Errorf("root admin %s: reset second factors: %w", email, err)
		}
		a.Logger.Error("SUPER_ADMIN_RESET_MFA is on: the root account's second factors were removed at startup. " +
			"Sign in, set one up again, then remove the variable and restart")
	}
	switch outcome {
	case auth.RootAdminCreated:
		a.Logger.Info("created the root admin account", "email", email)
	case auth.RootAdminPromoted:
		a.Logger.Info("made an existing account the root admin; it keeps its own password", "email", email)
	}
	return nil
}

// Run serves HTTP until the process receives SIGINT/SIGTERM, then shuts
// down gracefully.
func (a *Application) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go a.worker.Run(ctx)

	errCh := make(chan error, 1)
	go func() {
		a.Logger.Info("http server listening", slog.String("addr", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		a.Logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	}
}

// connectMongo opens the MongoDB client from
// MONGODB_URI/MONGODB_DATABASE.
func connectMongo(ctx context.Context) (*mongo.Client, *mongo.Database, error) {
	uri := os.Getenv("MONGODB_URI")
	name := os.Getenv("MONGODB_DATABASE")
	if name == "" {
		name = "betailo"
	}

	client, err := mongodb.Connect(ctx, mongodb.Config{URI: uri, Database: name, ConnectTimeout: 10 * time.Second})
	if err != nil {
		return nil, nil, fmt.Errorf("app: connect mongo: %w", err)
	}
	return client, client.Database(name), nil
}

// connectStorage builds the object-storage provider from STORAGE_*
// environment variables — the same code points at SeaweedFS in dev and
// Cloudflare R2 in production, only these values change (see the
// "Concepts & Architecture" document's deployment architecture section).
func connectStorage(ctx context.Context) (*objectstorage.S3Provider, error) {
	provider, err := objectstorage.NewS3Provider(ctx, objectstorage.Config{
		Endpoint:       os.Getenv("STORAGE_ENDPOINT"),
		PublicEndpoint: os.Getenv("STORAGE_PUBLIC_ENDPOINT"),
		Bucket:         os.Getenv("STORAGE_BUCKET"),
		AccessKey:      os.Getenv("STORAGE_ACCESS_KEY"),
		SecretKey:      os.Getenv("STORAGE_SECRET_KEY"),
		Region:         os.Getenv("STORAGE_REGION"),
	})
	if err != nil {
		return nil, fmt.Errorf("app: connect storage: %w", err)
	}
	return provider, nil
}

// newTokenSigner builds the JWT signer from JWT_SECRET. Unlike the
// other environment variables in this file, there is no fallback: an
// empty or default secret would mean anyone can forge a valid token,
// so a missing JWT_SECRET fails startup instead of silently signing
// with an empty key.
func newTokenSigner() (ports.TokenSigner, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("app: JWT_SECRET must be set")
	}
	return &jwt.Signer{Secret: []byte(secret)}, nil
}

// accessTokenTTL reads ACCESS_TOKEN_TTL (a Go duration string, e.g.
// "24h"); auth.Login falls back to its own 24h default when this
// returns zero, so an unset or unparsable value is not a startup
// failure.
// publicURL is where the site's pages are served from: links in e-mails
// point there, and passkeys are bound to its host. APP_PUBLIC_URL when
// set, else the first origin the API accepts.
func publicURL() string {
	if v := strings.TrimRight(strings.TrimSpace(os.Getenv("APP_PUBLIC_URL")), "/"); v != "" {
		return v
	}
	return httpserver.AllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGIN"))[0]
}

func hostOf(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return "localhost"
}

// passkeyOrigins are the origins a browser may present for a passkey.
func passkeyOrigins(siteURL string) []string {
	origins := []string{siteURL}
	for _, o := range httpserver.AllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGIN")) {
		if o != siteURL {
			origins = append(origins, o)
		}
	}
	return origins
}

// newMailer sends through SMTP when SMTP_HOST is set; otherwise every
// message is written to the log, and the startup says so.
func newMailer(logger *slog.Logger) ports.Mailer {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		logger.Warn("SMTP_HOST is not set: e-mails (invitations, password resets, farm decisions) are written to the log instead of being sent")
		return &smtp.LogMailer{Logger: logger}
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	from := os.Getenv("MAIL_FROM")
	if from == "" {
		from = "Betailo <no-reply@" + host + ">"
	}
	return &smtp.Mailer{Config: smtp.Config{Host: host, Port: port, Username: os.Getenv("SMTP_USERNAME"), Password: os.Getenv("SMTP_PASSWORD"), From: from}}
}

func accessTokenTTL() time.Duration {
	raw := os.Getenv("ACCESS_TOKEN_TTL")
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0
	}
	return d
}

// refreshTokenTTL reads REFRESH_TOKEN_TTL (a Go duration string, e.g.
// "720h"); auth.Login/auth.RefreshToken fall back to their own 30-day
// default when this returns zero, same non-fatal pattern as
// accessTokenTTL.
func refreshTokenTTL() time.Duration {
	raw := os.Getenv("REFRESH_TOKEN_TTL")
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0
	}
	return d
}

// cookieConfig reads COOKIE_SECURE (default true) and COOKIE_SAMESITE
// (default "none") — see docs/adr/0008 for why "none" is the right
// default given the frontend and backend are on different origins in
// production today, and httpserver.CookieConfig's comment for how to
// flip this once that changes.
func cookieConfig() httpserver.CookieConfig {
	secure := true
	if raw := os.Getenv("COOKIE_SECURE"); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			secure = parsed
		}
	}

	sameSite := http.SameSiteNoneMode
	switch strings.ToLower(os.Getenv("COOKIE_SAMESITE")) {
	case "lax":
		sameSite = http.SameSiteLaxMode
	case "strict":
		sameSite = http.SameSiteStrictMode
	}

	return httpserver.CookieConfig{Secure: secure, SameSite: sameSite}
}

// maxFarmsPerAccount reads MAX_FARMS_PER_ACCOUNT: how many farms an
// account may own unless staff raised it for that account. Unset or
// not a positive number: domain.DefaultMaxFarmsPerAccount.
func maxFarmsPerAccount() int {
	n, err := strconv.Atoi(os.Getenv("MAX_FARMS_PER_ACCOUNT"))
	if err != nil || n < 1 {
		return domain.DefaultMaxFarmsPerAccount
	}
	return n
}
