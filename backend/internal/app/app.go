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
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/admin"
	"github.com/pauluno777/betailo/backend/application/usecase/auth"
	"github.com/pauluno777/betailo/backend/application/usecase/cycle"
	"github.com/pauluno777/betailo/backend/application/usecase/dashboard"
	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/application/usecase/favorite"
	"github.com/pauluno777/betailo/backend/application/usecase/forum"
	"github.com/pauluno777/betailo/backend/application/usecase/listing"
	"github.com/pauluno777/betailo/backend/application/usecase/report"
	"github.com/pauluno777/betailo/backend/application/usecase/reservation"
	"github.com/pauluno777/betailo/backend/application/usecase/review"
	"github.com/pauluno777/betailo/backend/application/usecase/trust"
	"github.com/pauluno777/betailo/backend/infrastructure/clock"
	"github.com/pauluno777/betailo/backend/infrastructure/jwt"
	"github.com/pauluno777/betailo/backend/infrastructure/mongodb"
	"github.com/pauluno777/betailo/backend/infrastructure/objectstorage"
	"github.com/pauluno777/betailo/backend/infrastructure/payment"
	"github.com/pauluno777/betailo/backend/infrastructure/security"
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
	hasher := security.BcryptHasher{}
	systemClock := clock.System{}
	sandboxPayments := payment.Sandbox{}

	register := &auth.Register{Users: users, Hasher: hasher, Clock: systemClock}
	login := &auth.Login{
		Users: users, RefreshTokens: refreshTokens, Hasher: hasher, Signer: signer,
		Clock: systemClock, AccessTTL: accessTokenTTL(), RefreshTTL: refreshTokenTTL(),
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

	reserveCycle := &reservation.ReserveCycle{
		Listings: listings, Cycles: cycles, Reservations: reservations,
		Payments: sandboxPayments, Tx: &mongodb.TxManager{Client: client}, Clock: systemClock,
	}
	payReservation := &reservation.PayReservation{Reservations: reservations, Payments: sandboxPayments, Clock: systemClock}
	markDelivered := &reservation.MarkDelivered{Reservations: reservations, Listings: listings, Cycles: cycles, Farms: farms, Clock: systemClock}
	confirmDelivery := &reservation.ConfirmDelivery{
		Reservations: reservations, Reports: reports, Listings: listings, Cycles: cycles,
		Farms: farms, Trust: trustRecompute, Clock: systemClock,
	}
	openDispute := &reservation.OpenDispute{Reservations: reservations, Reports: reports, Clock: systemClock}
	listMyReservations := &reservation.ListMyReservations{Reservations: reservations}
	listReservationsByFarm := &reservation.ListReservationsByFarm{Farms: farms, Cycles: cycles, Listings: listings, Reservations: reservations}

	createReview := &review.CreateReview{
		Reservations: reservations, Listings: listings, Cycles: cycles, Farms: farms, Reviews: reviews, Trust: trustRecompute,
	}
	listFarmReviews := &review.ListByFarm{Reviews: reviews}

	createReport := &report.CreateReport{Reports: reports, Listings: listings, Clock: systemClock}

	createForumPost := &forum.CreatePost{Forum: forumPosts, Clock: systemClock}
	listForumPosts := &forum.ListPosts{Forum: forumPosts}
	getForumPost := &forum.GetPost{Forum: forumPosts}
	addForumReply := &forum.AddReply{Forum: forumPosts, Clock: systemClock}
	markBestAnswer := &forum.MarkBestAnswer{Forum: forumPosts}

	sellerDashboard := &dashboard.SellerDashboard{
		Farms: farms, Cycles: cycles, Listings: listings, Reservations: reservations, Users: users, Clock: systemClock,
	}

	adminOverview := &admin.Overview{Cycles: cycles, Listings: listings, Reservations: reservations, Reports: reports, Clock: systemClock}
	adminListUsers := &admin.ListUsers{Users: users}
	adminVerifyUser := &admin.VerifyUser{Users: users, Trust: trustRecompute}
	adminSuspendUser := &admin.SuspendUser{Users: users}
	adminListCycles := &admin.ListAllCycles{Cycles: cycles}
	adminListListings := &admin.ListAllListings{Cycles: cycles, Listings: listings}
	adminListEscrow := &admin.ListEscrowTransactions{Cycles: cycles, Listings: listings, Reservations: reservations}
	adminReleaseEscrow := &reservation.AdminReleaseEscrow{
		Reservations: reservations, Listings: listings, Cycles: cycles, Farms: farms, Trust: trustRecompute, Clock: systemClock,
	}
	adminRefundEscrow := &reservation.AdminRefundEscrow{Reservations: reservations, Payments: sandboxPayments, Clock: systemClock}
	adminListReports := &admin.ListReports{Reports: reports}
	adminGetReportDetail := &admin.GetReportDetail{
		Reports: reports, Users: users, Listings: listings, Cycles: cycles, Farms: farms, Search: listings, Clock: systemClock,
	}
	adminResolveReport := &admin.ResolveReport{Reports: reports, Users: users, Trust: trustRecompute}

	bgWorker := &worker.Worker{
		Cycles: cycles,
		ExpireUnpaid: &reservation.ExpireUnpaidReservations{
			Reservations: reservations, Listings: listings, Cycles: cycles, Notifications: notifications, Clock: systemClock,
		},
		AutoRelease: &reservation.AutoReleaseDeliveries{
			Reservations: reservations, Reports: reports, Listings: listings, Cycles: cycles,
			Farms: farms, Trust: trustRecompute, Notifications: notifications, Clock: systemClock,
		},
		Notifications: notifications,
		Clock:         systemClock,
		Logger:        logger,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr: ":" + port,
		Handler: httpserver.NewRouter(logger, httpserver.Dependencies{
			Register:     register,
			Login:        login,
			RefreshToken: refreshToken,
			Logout:       logout,
			Signer:       signer,
			Cookies:      cookieConfig(),
			RateLimits:   httpserver.DefaultRateLimits(os.Getenv("TRUST_PROXY_HEADERS") == "true"),

			CreateFarm: &farm.CreateFarm{Farms: farms, Trust: trustRecompute, Clock: systemClock},
			GetFarmProfile: &farm.GetProfile{
				Farms: farms, Users: users, Reviews: reviews, Cycles: cycles, Listings: listings, Reservations: reservations,
			},
			ListMyFarms: &farm.ListMyFarms{Farms: farms},
			UpdateFarm:  &farm.UpdateFarm{Farms: farms},
			DeleteFarm:  &farm.DeleteFarm{Farms: farms, Cycles: cycles},

			CreateCycle:       &cycle.CreateCycle{Farms: farms, Cycles: cycles},
			GetCycle:          &cycle.GetCycle{Cycles: cycles, Clock: systemClock},
			AddHealthEvent:    &cycle.AddHealthEvent{Farms: farms, Cycles: cycles, Clock: systemClock},
			ListCyclesByFarm:  &cycle.ListCyclesByFarm{Cycles: cycles},
			UpdateCycleStatus: &cycle.UpdateCycleStatus{Farms: farms, Cycles: cycles},
			DeleteCycle:       &cycle.DeleteCycle{Farms: farms, Cycles: cycles, Listings: listings},

			CreateListing: &listing.CreateListing{Farms: farms, Cycles: cycles, Listings: listings},
			GetListing:    &listing.GetListing{Listings: listings, Cycles: cycles, Farms: farms},
			GetListingDetail: &listing.GetDetail{
				Listings: listings, Cycles: cycles, Farms: farms, Users: users, Reviews: reviews,
				Reservations: reservations, Favorites: favorites, Clock: systemClock,
			},
			UpdateListing:       &listing.UpdateListing{Farms: farms, Cycles: cycles, Listings: listings},
			UnpublishListing:    &listing.UnpublishListing{Farms: farms, Cycles: cycles, Listings: listings},
			PresignListingPhoto: &listing.PresignListingPhoto{Farms: farms, Cycles: cycles, Listings: listings, Storage: storageProvider},
			SetListingPhotos:    &listing.SetListingPhotos{Farms: farms, Cycles: cycles, Listings: listings},
			SearchListings:      &listing.SearchListings{Search: listings, Favorites: favorites},
			AddFavorite:         &favorite.Add{Listings: listings, Favorites: favorites, Clock: systemClock},
			RemoveFavorite:      &favorite.Remove{Favorites: favorites},
			ListMyFavorites:     &favorite.ListMine{Favorites: favorites, Search: listings},
			Storage:             storageProvider,

			ReserveCycle:           reserveCycle,
			PayReservation:         payReservation,
			MarkDelivered:          markDelivered,
			ConfirmDelivery:        confirmDelivery,
			OpenDispute:            openDispute,
			ListMyReservations:     listMyReservations,
			ListReservationsByFarm: listReservationsByFarm,

			CreateReview:    createReview,
			ListFarmReviews: listFarmReviews,

			CreateReport: createReport,

			CreateForumPost: createForumPost,
			ListForumPosts:  listForumPosts,
			GetForumPost:    getForumPost,
			AddForumReply:   addForumReply,
			MarkBestAnswer:  markBestAnswer,

			SellerDashboard: sellerDashboard,

			AdminOverview:        adminOverview,
			AdminListUsers:       adminListUsers,
			AdminVerifyUser:      adminVerifyUser,
			AdminSuspendUser:     adminSuspendUser,
			AdminListCycles:      adminListCycles,
			AdminListListings:    adminListListings,
			AdminListEscrow:      adminListEscrow,
			AdminReleaseEscrow:   adminReleaseEscrow,
			AdminRefundEscrow:    adminRefundEscrow,
			AdminListReports:     adminListReports,
			AdminGetReportDetail: adminGetReportDetail,
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
	a.Logger.Info("connected to MongoDB and bootstrapped collections")

	if err := a.storageProvider.EnsureBucket(ctx); err != nil {
		return err
	}
	a.Logger.Info("object storage provider ready")

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
