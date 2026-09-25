// Package app is the composition root: the one place in this codebase
// that knows about both the application layer (usecases, ports) and the
// concrete infrastructure (mongodb, objectstorage) at once, and wires
// them together. Everything else — domain, application, infrastructure,
// interfaces — depends only inward; app is where the dependency graph
// actually gets built (see cmd/api/main.go, which just calls
// Initialize/Bootstrap/Run).
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

	"github.com/pauluno777/betailo/backend/application/usecase/reservation"
	"github.com/pauluno777/betailo/backend/infrastructure/mongodb"
	"github.com/pauluno777/betailo/backend/infrastructure/objectstorage"
	"github.com/pauluno777/betailo/backend/interfaces/httpserver"
)

// Application holds what Run needs. Its usecases (ReserveCycle, and
// whatever follows it) are exported so a future httpserver handler set
// can take this struct and wire routes to them without app needing to
// know about routing.
type Application struct {
	Logger       *slog.Logger
	ReserveCycle *reservation.ReserveCycle

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

	reserveCycle := &reservation.ReserveCycle{
		Listings:     &mongodb.ListingRepository{Coll: database.Collection("listings")},
		Cycles:       &mongodb.CycleRepository{Coll: database.Collection("cycles")},
		Reservations: &mongodb.ReservationRepository{Coll: database.Collection("reservations")},
		Tx:           &mongodb.TxManager{Client: client},
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      httpserver.NewRouter(logger),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	application := &Application{
		Logger:          logger,
		ReserveCycle:    reserveCycle,
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
// "Architecture de déploiement" section of the architecture document).
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
