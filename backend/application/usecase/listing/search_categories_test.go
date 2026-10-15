package listing

import (
	"slices"
	"testing"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestExpandCategories(t *testing.T) {
	cat := domain.SeedCatalog()
	cases := []struct {
		name                string
		categories, species []string
		want                []string
	}{
		{"a whole category", []string{"caprins"}, nil, []string{"chevres", "chevreaux"}},
		{"narrowed by one of its sub-categories", []string{"volaille"}, []string{"pondeuses"}, []string{"pondeuses"}},
		{"two categories, one narrowed", []string{"volaille", "ovins"}, []string{"dindes"}, []string{"dindes", "moutons", "agneaux"}},
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
