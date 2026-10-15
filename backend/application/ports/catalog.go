package ports

import (
	"context"

	"github.com/pauluno777/betailo/backend/domain"
)

// CatalogRepository stores the published versions of the platform's
// reference data. Versions are never changed once written.
type CatalogRepository interface {
	// Latest returns the highest version, or domain.ErrNotFound when
	// none has been published yet.
	Latest(ctx context.Context) (*domain.Catalog, error)
	// Get returns one version, or domain.ErrNotFound.
	Get(ctx context.Context, version int) (*domain.Catalog, error)
	// List returns every version, newest first.
	List(ctx context.Context) ([]*domain.Catalog, error)
	// Insert writes c as version c.Version; domain.ErrConflict when that
	// number is taken (someone published at the same time).
	Insert(ctx context.Context, c *domain.Catalog) error
}

// CatalogSource is the reference data in force, for the use cases that
// validate against it.
type CatalogSource interface {
	Current(ctx context.Context) (*domain.Catalog, error)
}
