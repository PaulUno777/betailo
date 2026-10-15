// Package catalog serves and publishes the platform's reference data
// (see domain.Catalog): separate references, each with its own versions.
package catalog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Latest is the version in force of each reference; a reference never
// published is absent.
func Latest(ctx context.Context, repo ports.CatalogRepository) (map[string]*domain.CatalogPart, error) {
	out := map[string]*domain.CatalogPart{}
	for _, name := range domain.CatalogParts {
		p, err := repo.Latest(ctx, name)
		switch {
		case errors.Is(err, domain.ErrNotFound):
			continue
		case err != nil:
			return nil, err
		}
		out[name] = p
	}
	return out, nil
}

// Current is the references in force, put together (the seed for any
// not published yet).
func Current(ctx context.Context, repo ports.CatalogRepository) (*domain.Catalog, map[string]*domain.CatalogPart, error) {
	parts, err := Latest(ctx, repo)
	if err != nil {
		return nil, nil, err
	}
	c, err := domain.AssembleCatalog(parts)
	return c, parts, err
}

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

// Current implements ports.CatalogSource.
func (c *Cache) Current(ctx context.Context) (*domain.Catalog, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if c.current != nil && now.Sub(c.loaded) < c.TTL {
		return c.current, nil
	}
	latest, _, err := Current(ctx, c.Repo)
	if err != nil {
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

// EnsureSeed gives every reference a version, on every startup:
//   - a reference never published gets version 1, from the catalog as it
//     was stored before the references were split (legacy, when there is
//     one) or else from the seed;
//   - a reference whose latest version is an older seed — not one staff
//     published — gets the current seed's content as its next version,
//     when that differs and removes no key, so deployed databases follow.
func EnsureSeed(ctx context.Context, repo ports.CatalogRepository, legacy ports.LegacyCatalogSource, now time.Time) error {
	seed, revision := domain.SeedCatalog(), domain.SeedRevision()
	var old *domain.Catalog
	oldRevision := 0
	if legacy != nil {
		c, rev, err := legacy.Latest(ctx)
		switch {
		case err == nil:
			old, oldRevision = c, rev
		case !errors.Is(err, domain.ErrNotFound):
			return err
		}
	}
	for _, name := range domain.CatalogParts {
		latest, err := repo.Latest(ctx, name)
		switch {
		case errors.Is(err, domain.ErrNotFound):
			from, rev, note := seed, revision, ""
			if old != nil {
				from, rev, note = old, oldRevision, "Reprise du catalogue d’avant la séparation des références"
			}
			content, err := from.PartContent(name)
			if err != nil {
				return err
			}
			if err := insert(ctx, repo, &domain.CatalogPart{Name: name, Version: 1, Note: note, PublishedAt: now, SeedRevision: rev, Content: content}); err != nil {
				return err
			}
			continue
		case err != nil:
			return err
		}
		if !latest.PublishedBy.IsZero() || latest.SeedRevision >= revision {
			continue // staff own it now, or the seed is already in force
		}
		content, _ := seed.PartContent(name)
		if bytes.Equal(content, latest.Content) {
			continue
		}
		current, err := domain.AssembleCatalog(map[string]*domain.CatalogPart{name: latest})
		if err != nil {
			return err
		}
		if removed := domain.CatalogRemovals(current, seed); len(removed) > 0 {
			return fmt.Errorf("catalog: seed revision %d removes %s from %s: %w", revision, strings.Join(removed, ", "), name, domain.ErrConflict)
		}
		next := &domain.CatalogPart{Name: name, Version: latest.Version + 1, PublishedAt: now, SeedRevision: revision, Content: content,
			Note: fmt.Sprintf("Révision %d des listes de la plateforme", revision)}
		if err := insert(ctx, repo, next); err != nil {
			return err
		}
	}
	return nil
}

func insert(ctx context.Context, repo ports.CatalogRepository, p *domain.CatalogPart) error {
	if err := repo.Insert(ctx, p); err != nil && !errors.Is(err, domain.ErrConflict) {
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
