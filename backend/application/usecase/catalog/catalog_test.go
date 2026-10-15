package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

type fakeRepo struct {
	ports.CatalogRepository
	versions []*domain.Catalog
	reads    int
	fail     error
}

func (f *fakeRepo) Latest(context.Context) (*domain.Catalog, error) {
	f.reads++
	if f.fail != nil {
		return nil, f.fail
	}
	if len(f.versions) == 0 {
		return nil, domain.ErrNotFound
	}
	return f.versions[len(f.versions)-1], nil
}

func (f *fakeRepo) Insert(_ context.Context, c *domain.Catalog) error {
	f.versions = append(f.versions, c)
	return nil
}

func TestCache_ReadsOncePerTTLAndKeepsTheLastGoodOne(t *testing.T) {
	now := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	repo := &fakeRepo{versions: []*domain.Catalog{{Version: 3}}}
	cache := &Cache{Repo: repo, TTL: 30 * time.Second, Now: func() time.Time { return now }}

	for i := 0; i < 3; i++ {
		if c, err := cache.Current(context.Background()); err != nil || c.Version != 3 {
			t.Fatalf("Current = %v, %v", c, err)
		}
	}
	if repo.reads != 1 {
		t.Errorf("reads = %d within the TTL, want 1", repo.reads)
	}

	repo.versions = append(repo.versions, &domain.Catalog{Version: 4})
	now = now.Add(31 * time.Second)
	if c, _ := cache.Current(context.Background()); c.Version != 4 {
		t.Errorf("after the TTL, version = %d, want 4", c.Version)
	}

	repo.fail = errors.New("mongo down")
	now = now.Add(time.Minute)
	if c, err := cache.Current(context.Background()); err != nil || c.Version != 4 {
		t.Errorf("with the database down: %v, %v; want the last good version", c, err)
	}
}

func TestCache_ServesTheSeedBeforeAnyPublish(t *testing.T) {
	cache := &Cache{Repo: &fakeRepo{}, TTL: time.Minute}
	c, err := cache.Current(context.Background())
	if err != nil || c.Version != 1 || len(c.Categories) == 0 {
		t.Fatalf("Current = %+v, %v; want the seed", c, err)
	}
}

func TestEnsureSeed_PublishesVersionOneOnce(t *testing.T) {
	repo := &fakeRepo{}
	at := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		if err := EnsureSeed(context.Background(), repo, at); err != nil {
			t.Fatal(err)
		}
	}
	if len(repo.versions) != 1 || repo.versions[0].Version != 1 || !repo.versions[0].PublishedAt.Equal(at) {
		t.Errorf("versions = %+v", repo.versions)
	}
}
