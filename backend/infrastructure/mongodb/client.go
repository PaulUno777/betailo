// Package mongodb is the only layer in this codebase that imports
// mongo-driver. It implements the application/ports interfaces
// (ListingRepository, CycleRepository, ReservationRepository, TxManager)
// against MongoDB, using the official driver (mongo-driver/v2) with
// explicit queries (Find, UpdateOne, Aggregate) rather than an ODM such
// as mgm — see the "Concepts & Architecture" document, section "Vue
// d'ensemble de l'architecture", for the reasoning.
package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Config holds what's needed to open a MongoDB client. In production
// these come from environment variables (see docker-compose.yml):
// MONGODB_URI points at a local `mongo` container in dev and at an
// Atlas SRV connection string in production — same code either way.
type Config struct {
	URI            string
	Database       string
	ConnectTimeout time.Duration
}

// Connect validates the config, connects a *mongo.Client and pings the
// primary. The ping is deliberate: the driver's client is lazy by
// design, so without it a bad URI would only surface on the first real
// query instead of at startup (kept separate from Ping below so callers
// that already have a client don't pay for two round trips).
func Connect(ctx context.Context, cfg Config) (*mongo.Client, error) {
	if cfg.URI == "" {
		return nil, fmt.Errorf("mongodb: MONGODB_URI must not be empty")
	}

	connectCtx := ctx
	if cfg.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, cfg.ConnectTimeout)
		defer cancel()
	}

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, fmt.Errorf("mongodb: connect: %w", err)
	}

	if err := client.Ping(connectCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongodb: ping: %w", err)
	}

	return client, nil
}

// Ping is a thin wrapper kept here so callers (e.g. a future /health
// check) don't need to know the driver's read-preference API.
func Ping(ctx context.Context, client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return client.Ping(ctx, readpref.Primary())
}
