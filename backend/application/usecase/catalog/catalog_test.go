package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// fakeRepo keeps each reference's versions in memory, oldest first.
type fakeRepo struct {
	parts map[string][]*domain.CatalogPart
	reads int
	fail  error
}

func (f *fakeRepo) Latest(_ context.Context, name string) (*domain.CatalogPart, error) {
	f.reads++
	if f.fail != nil {
		return nil, f.fail
	}
	list := f.parts[name]
	if len(list) == 0 {
		return nil, domain.ErrNotFound
	}
	return list[len(list)-1], nil
}

func (f *fakeRepo) Get(_ context.Context, name string, version int) (*domain.CatalogPart, error) {
	for _, p := range f.parts[name] {
		if p.Version == version {
			return p, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeRepo) List(_ context.Context, name string) ([]*domain.CatalogPart, error) {
	var out []*domain.CatalogPart
	for _, n := range domain.CatalogParts {
		if name == "" || n == name {
			for i := len(f.parts[n]) - 1; i >= 0; i-- {
				out = append(out, f.parts[n][i])
			}
		}
	}
	return out, nil
}

func (f *fakeRepo) Insert(_ context.Context, p *domain.CatalogPart) error {
	if f.parts == nil {
		f.parts = map[string][]*domain.CatalogPart{}
	}
	for _, q := range f.parts[p.Name] {
		if q.Version == p.Version {
			return domain.ErrConflict
		}
	}
	f.parts[p.Name] = append(f.parts[p.Name], p)
	return nil
}

// seeded is a repo holding the seed as version 1 of every reference.
func seeded() *fakeRepo {
	repo := &fakeRepo{}
	_ = EnsureSeed(context.Background(), repo, nil, time.Now())
	return repo
}

type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

type fakeLegacy struct {
	c   *domain.Catalog
	rev int
}

func (f fakeLegacy) Latest(context.Context) (*domain.Catalog, int, error) {
	if f.c == nil {
		return nil, 0, domain.ErrNotFound
	}
	return f.c, f.rev, nil
}

func TestCache_ReadsOncePerTTLAndKeepsTheLastGoodOne(t *testing.T) {
	now := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	repo := seeded()
	repo.reads = 0
	cache := &Cache{Repo: repo, TTL: 30 * time.Second, Now: func() time.Time { return now }}
	for i := 0; i < 3; i++ {
		if _, err := cache.Current(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if repo.reads != len(domain.CatalogParts) {
		t.Errorf("reads = %d within the TTL, want one per reference", repo.reads)
	}
	repo.fail = errors.New("mongo down")
	now = now.Add(time.Minute)
	if c, err := cache.Current(context.Background()); err != nil || c.Versions[domain.PartAnimals] != 1 {
		t.Errorf("on failure: %v, %v; want the last good catalog", c, err)
	}
}

func TestEnsureSeed_GivesEveryReferenceVersionOneOnce(t *testing.T) {
	repo := &fakeRepo{}
	for i := 0; i < 2; i++ {
		if err := EnsureSeed(context.Background(), repo, nil, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range domain.CatalogParts {
		if got := repo.parts[name]; len(got) != 1 || got[0].Version != 1 || got[0].SeedRevision != domain.SeedRevision() {
			t.Errorf("%s = %+v", name, got)
		}
	}
}

// A database from before the split keeps its lists: they become each
// reference's version 1.
func TestEnsureSeed_MovesTheLegacyCatalog(t *testing.T) {
	old := domain.SeedCatalog()
	old.ReviewTags[0].Label.EN = "Healthy herd"
	repo := &fakeRepo{}
	if err := EnsureSeed(context.Background(), repo, fakeLegacy{c: old, rev: domain.SeedRevision()}, time.Now()); err != nil {
		t.Fatal(err)
	}
	c, _, _ := Current(context.Background(), repo)
	if c.ReviewTags[0].Label.EN != "Healthy herd" || repo.parts[domain.PartReviewTags][0].Note == "" {
		t.Errorf("legacy lists not carried over: %+v", repo.parts[domain.PartReviewTags][0])
	}
}

// A newer seed revision reaches only the references still on an older
// seed whose content it changes; staff's versions are left alone.
func TestEnsureSeed_UpgradesOlderSeedReferencesOnly(t *testing.T) {
	repo := seeded()
	for _, name := range domain.CatalogParts {
		repo.parts[name][0].SeedRevision = 0
	}
	repo.parts[domain.PartPlaces][0].Content = mustPart(t, withoutTowns(), domain.PartPlaces)
	repo.parts[domain.PartAnimals][0].PublishedBy = bson.NewObjectID()
	if err := EnsureSeed(context.Background(), repo, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	if n := len(repo.parts[domain.PartPlaces]); n != 2 {
		t.Errorf("places versions = %d, want the seed's towns as v2", n)
	}
	if n := len(repo.parts[domain.PartReviewTags]); n != 1 {
		t.Errorf("reviewTags versions = %d, want none added (same content)", n)
	}
	if n := len(repo.parts[domain.PartAnimals]); n != 1 {
		t.Errorf("animals versions = %d, want staff's version left alone", n)
	}
}

func withoutTowns() *domain.Catalog {
	c := domain.SeedCatalog()
	for i := range c.Countries {
		for j := range c.Countries[i].Regions {
			c.Countries[i].Regions[j].Cities = nil
		}
	}
	return c
}

func mustPart(t *testing.T, c *domain.Catalog, name string) []byte {
	t.Helper()
	content, err := c.PartContent(name)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func TestPublish_MakesANewVersionOfTheTouchedReferencesOnly(t *testing.T) {
	repo := seeded()
	cache := &Cache{Repo: repo, TTL: time.Hour}
	_, _ = cache.Current(context.Background())
	edited := domain.SeedCatalog()
	edited.Categories[0].Label.EN = "Fowl"
	edited.ReviewTags[0].Active = false
	admin := bson.NewObjectID()

	got, err := (&Publish{Repo: repo, Clock: fixedClock{time.Now()}, Cache: cache}).Execute(context.Background(), PublishInput{
		Base: edited.Versions, Note: "  Fowl  ", Lists: edited, By: admin,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Parts) != 2 || got.Catalog.Versions[domain.PartAnimals] != 2 || got.Catalog.Versions[domain.PartReviewTags] != 2 || got.Catalog.Versions[domain.PartPlaces] != 1 {
		t.Errorf("published %d parts, versions %v", len(got.Parts), got.Catalog.Versions)
	}
	if p := got.Parts[0]; p.Note != "Fowl" || p.PublishedBy != admin {
		t.Errorf("part = %+v", p)
	}
	if c, _ := cache.Current(context.Background()); c.Categories[0].Label.EN != "Fowl" {
		t.Errorf("cache still serves the old animals")
	}
}

func TestPublish_Refusals(t *testing.T) {
	removed := domain.SeedCatalog()
	removed.Categories = removed.Categories[1:]
	invalid := domain.SeedCatalog()
	invalid.Categories[0].Icon = "dragon"
	changed := domain.SeedCatalog()
	changed.Categories[0].Label.EN = "Fowl"
	for _, tc := range []struct {
		name string
		in   PublishInput
		code string
	}{
		{"nothing changed", PublishInput{Base: map[string]int{}, Note: "x", Lists: domain.SeedCatalog()}, "catalog.no_changes"},
		{"animals published since", PublishInput{Base: map[string]int{domain.PartAnimals: 0}, Note: "x", Lists: changed}, "catalog.stale"},
		{"no note", PublishInput{Base: changed.Versions, Note: " ", Lists: changed}, "catalog.note_invalid"},
		{"invalid lists", PublishInput{Base: invalid.Versions, Note: "x", Lists: invalid}, "catalog.invalid"},
		{"a key in use removed", PublishInput{Base: removed.Versions, Note: "x", Lists: removed}, "catalog.key_removed"},
	} {
		repo := seeded()
		_, err := (&Publish{Repo: repo, Clock: fixedClock{time.Now()}}).Execute(context.Background(), tc.in)
		if code, _ := domain.CodeOf(err); code != tc.code {
			t.Errorf("%s: code = %q (%v), want %s", tc.name, code, err, tc.code)
		}
		if len(repo.parts[domain.PartAnimals]) != 1 {
			t.Errorf("%s: published anyway", tc.name)
		}
	}
}
