package ports

import (
	"context"

	"github.com/pauluno777/betailo/backend/domain"
)

// CatalogChangesetRepository holds the one shared changeset.
type CatalogChangesetRepository interface {
	// Get returns the changeset, empty (Rev 0) when there is none.
	Get(ctx context.Context) (*domain.CatalogChangeset, error)
	// Save writes cs as revision rev+1 if the stored one is still rev;
	// otherwise domain.ErrConflict.
	Save(ctx context.Context, cs *domain.CatalogChangeset, rev int) error
}

// CatalogUsage counts what uses each reference entry, keyed like
// "species:chicks", "region:CM:west", "city:CM:bafoussam",
// "country:CM", "forumCategories:sante", "reviewTags:on_time",
// "reportReasons:other", "farmReviewChecks:name_matches",
// "unit:chick".
type CatalogUsage interface {
	Usage(ctx context.Context) (map[string]int, error)
}
