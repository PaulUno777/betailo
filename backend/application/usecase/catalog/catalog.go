// Package catalog serves and publishes the platform's reference data
// (see domain.Catalog).
package catalog

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Cache is the catalog in force, read from the repository at most once
// per TTL: every listing, farm and review checks against it, and a
// publish on another instance reaches this one within TTL.
type Cache struct {
	Repo ports.CatalogRepository
	TTL  time.Duration
	Now  func() time.Time

	mu      sync.Mutex
	current *domain.Catalog
	loaded  time.Time
}

var _ ports.CatalogSource = (*Cache)(nil)

// Current implements ports.CatalogSource. Before anything is published
// it is the seed, version 1.
func (c *Cache) Current(ctx context.Context) (*domain.Catalog, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if c.current != nil && now.Sub(c.loaded) < c.TTL {
		return c.current, nil
	}
	latest, err := c.Repo.Latest(ctx)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		latest = domain.SeedCatalog()
	case err != nil:
		if c.current != nil {
			return c.current, nil // keep serving the last good one
		}
		return nil, err
	}
	c.current, c.loaded = latest, now
	return latest, nil
}

// Forget drops the cached copy, so the next read sees a version this
// instance just published.
func (c *Cache) Forget() {
	c.mu.Lock()
	c.current = nil
	c.mu.Unlock()
}

func (c *Cache) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// EnsureSeed publishes version 1 when the database has no catalog yet,
// so the lists the platform shipped with become an ordinary version.
func EnsureSeed(ctx context.Context, repo ports.CatalogRepository, now time.Time) error {
	if _, err := repo.Latest(ctx); !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	seed := domain.SeedCatalog()
	seed.PublishedAt = now
	if err := repo.Insert(ctx, seed); err != nil && !errors.Is(err, domain.ErrConflict) {
		return err
	}
	return nil
}

// Of reads the catalog from src, or the seed when src is nil (a use case
// built without one, in tests).
func Of(ctx context.Context, src ports.CatalogSource) (*domain.Catalog, error) {
	if src == nil {
		return domain.SeedCatalog(), nil
	}
	return src.Current(ctx)
}
