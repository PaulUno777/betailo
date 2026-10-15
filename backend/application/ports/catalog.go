package ports

import (
	"context"

	"github.com/pauluno777/betailo/backend/domain"
)

// CatalogRepository stores the published versions of each reference
// (domain.CatalogParts). Versions are never changed once written.
type CatalogRepository interface {
	// Latest returns name's highest version, or domain.ErrNotFound when
	// none has been published yet.
	Latest(ctx context.Context, name string) (*domain.CatalogPart, error)
	// Get returns one version of name, or domain.ErrNotFound.
	Get(ctx context.Context, name string, version int) (*domain.CatalogPart, error)
	// List returns name's versions (every reference's when name is ""),
	// newest first.
	List(ctx context.Context, name string) ([]*domain.CatalogPart, error)
	// Insert writes p as version p.Version of p.Name; domain.ErrConflict
	// when that number is taken (someone published at the same time).
	Insert(ctx context.Context, p *domain.CatalogPart) error
}

// LegacyCatalogSource reads the catalog as it was stored before the
// references were split (one document per version), for the one-time
// move to CatalogRepository. domain.ErrNotFound when there is none.
type LegacyCatalogSource interface {
	Latest(ctx context.Context) (*domain.Catalog, int, error)
}

// CatalogSource is the reference data in force, for the use cases that
// validate against it.
type CatalogSource interface {
	Current(ctx context.Context) (*domain.Catalog, error)
}
