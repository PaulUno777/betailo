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
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/auth"
	"github.com/pauluno777/betailo/backend/application/usecase/reservation"
	"github.com/pauluno777/betailo/backend/infrastructure/clock"
	"github.com/pauluno777/betailo/backend/infrastructure/jwt"
	"github.com/pauluno777/betailo/backend/infrastructure/mongodb"
	"github.com/pauluno777/betailo/backend/infrastructure/objectstorage"
	"github.com/pauluno777/betailo/backend/infrastructure/security"
	"github.com/pauluno777/betailo/backend/interfaces/httpserver"
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

	reserveCycle := &reservation.ReserveCycle{
		Listings:     &mongodb.ListingRepository{Coll: database.Collection("listings")},
		Cycles:       &mongodb.CycleRepository{Coll: database.Collection("cycles")},
		Reservations: &mongodb.ReservationRepository{Coll: database.Collection("reservations")},
		Tx:           &mongodb.TxManager{Client: client},
	}

	users := &mongodb.UserRepository{Coll: database.Collection("users")}
	hasher := security.BcryptHasher{}
	systemClock := clock.System{}

	register := &auth.Register{Users: users, Hasher: hasher, Clock: systemClock}
	login := &auth.Login{Users: users, Hasher: hasher, Signer: signer, Clock: systemClock, TTL: accessTokenTTL()}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr: ":" + port,
		Handler: httpserver.NewRouter(logger, httpserver.Dependencies{
			Register: register,
			Login:    login,
			Signer:   signer,
		}),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
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
		Endpoint:  os.Getenv("STORAGE_ENDPOINT"),
		Bucket:    os.Getenv("STORAGE_BUCKET"),
		AccessKey: os.Getenv("STORAGE_ACCESS_KEY"),
		SecretKey: os.Getenv("STORAGE_SECRET_KEY"),
		Region:    os.Getenv("STORAGE_REGION"),
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
