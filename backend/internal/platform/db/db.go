// Package db owns the PostgreSQL connection used by every repository in
// the application (one pool, shared, injected into each module's
// repository — see internal/user, internal/cycle, etc.).
//
// The MVP reads SQL directly (via pgx or database/sql + a generator like
// sqlc) rather than through a heavy ORM — see the proposal document,
// section "Architecture technique et stack Go", for the reasoning.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Config holds what's needed to open a connection pool. In production
// these come from environment variables (see docker-compose.yml).
type Config struct {
	DSN             string // e.g. "postgres://user:pass@host:5432/betailo?sslmode=disable"
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

// Open validates the config and opens a *sql.DB. It does not ping the
// database — callers should call PingContext once the pool is created
// (kept separate so tests can construct a Config without a live DB).
func Open(cfg Config) (*sql.DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("db: DSN must not be empty")
	}

	sqlDB, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	return sqlDB, nil
}

// Ping is a thin wrapper kept here so callers don't need to know which
// driver/timeout convention the project uses.
func Ping(ctx context.Context, sqlDB *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return sqlDB.PingContext(ctx)
}
