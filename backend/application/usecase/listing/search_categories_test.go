package listing

import (
	"context"
	"slices"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

func TestExpandCategories(t *testing.T) {
	cat := domain.SeedCatalog()
	cases := []struct {
		name                string
		categories, species []string
		want                []string
	}{
		{"a whole category", []string{"caprine"}, nil, []string{"goats", "kids"}},
		{"narrowed by one of its sub-categories", []string{"poultry"}, []string{"laying-hens"}, []string{"laying-hens"}},
		{"two categories, one narrowed", []string{"poultry", "ovine"}, []string{"turkeys"}, []string{"turkeys", "sheep", "lambs"}},
		{"an unknown category finds nothing", []string{"dragons"}, nil, []string{"-"}},
	}
	for _, c := range cases {
		got := expandCategories(cat, c.categories, c.species)
		slices.Sort(got)
		want := slices.Clone(c.want)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s: got %v, want %v", c.name, got, want)
		}
	}
}

// captureSearch records the criteria the repository receives.
type captureSearch struct {
	ports.ListingSearchRepository
	got ports.ListingSearchCriteria
}

func (c *captureSearch) Search(_ context.Context, criteria ports.ListingSearchCriteria) (ports.ListingSearchResult, error) {
	c.got = criteria
	return ports.ListingSearchResult{}, nil
}

func TestSearchListings_FindsPlacesNamedInEitherLanguage(t *testing.T) {
	repo := &captureSearch{}
	uc := &SearchListings{Search: repo}
	if _, err := uc.Execute(context.Background(), ports.ListingSearchCriteria{Query: "poussins Yaoundé"}, bson.ObjectID{}); err != nil {
		t.Fatal(err)
	}
	if len(repo.got.QueryPlaces) != 2 || len(repo.got.QueryPlaces[0]) != 0 || !slices.Equal(repo.got.QueryPlaces[1], []string{"yaounde"}) {
		t.Errorf("QueryPlaces = %v, want no place for the first word and yaounde for the second", repo.got.QueryPlaces)
	}
}
