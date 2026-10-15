package domain

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestSeedCatalog_IsValid(t *testing.T) {
	c := SeedCatalog()
	if err := c.Validate(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if c.Version != 1 {
		t.Errorf("seed version = %d, want 1", c.Version)
	}
}

func TestCatalog_Validate_ReportsEveryProblem(t *testing.T) {
	c := SeedCatalog()
	poultry := c.Categories[0].Subcategories
	poultry[0].CycleType = "flying"
	poultry[1].Key = c.Categories[1].Subcategories[0].Key // two categories, one key
	poultry[3].Label.EN = ""
	c.Categories[2].Color = "pink"
	c.Countries[1].CallingCode = "+229"
	c.ReviewTags[0].Key = "Bad Key"
	c.Countries[0].Active = false // the default country

	err := c.Validate()
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	for _, want := range []string{"cycleType", "appears twice", "en label", "callingCode", `"Bad Key"`, "default country", `"pink"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error doesn't mention %q: %v", want, err)
		}
	}
}

func TestCatalog_Lookups(t *testing.T) {
	c := SeedCatalog()
	if s, err := c.SpeciesByKey("porcs"); err != nil || s.Unit.Key != "porcelet" {
		t.Errorf("SpeciesByKey(porcs) = %+v, %v", s, err)
	}
	if _, err := c.SpeciesByKey("veaux"); !errors.Is(err, ErrInvalid) {
		t.Errorf("a sub-category of an inactive category is offered: %v", err)
	}
	c.Categories[0].Subcategories[0].Active = false
	if _, err := c.SpeciesByKey(c.Categories[0].Subcategories[0].Key); !errors.Is(err, ErrInvalid) {
		t.Errorf("an inactive sub-category is still offered: %v", err)
	}
	if got := c.SpeciesOf("caprins"); !slices.Equal(got, []string{"chevres", "chevreaux"}) {
		t.Errorf("SpeciesOf(caprins) = %v", got)
	}

	if k, err := c.Country(""); err != nil || k.Code != DefaultCountry {
		t.Errorf("Country(\"\") = %+v, %v", k, err)
	}
	if _, err := c.Country("XX"); !errors.Is(err, ErrInvalid) {
		t.Errorf("Country(XX): err = %v", err)
	}

	cm, _ := c.Country("cm")
	for in, want := range map[string]string{"extreme-nord": "Extrême-Nord", "  OUEST ": "Ouest", "nord-ouest": "Nord-Ouest", "": ""} {
		if got, err := cm.CanonicalRegion(in); err != nil || got != want {
			t.Errorf("CanonicalRegion(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := cm.CanonicalRegion("Atlantique"); !errors.Is(err, ErrInvalid) {
		t.Errorf("unknown region: err = %v", err)
	}
	sn, _ := c.Country("SN")
	if got, _ := sn.CanonicalRegion("  Dakar   Plateau "); got != "Dakar Plateau" {
		t.Errorf("free-text region = %q", got)
	}
}

func TestCatalogRemovals(t *testing.T) {
	prev := SeedCatalog()
	next := SeedCatalog()
	next.Categories[0].Subcategories = next.Categories[0].Subcategories[1:]
	next.Categories = slices.Delete(next.Categories, 1, 2)
	next.ForumCategories[0].Active = false // deactivating is fine
	cm := &next.Countries[0]
	cm.Regions = cm.Regions[:len(cm.Regions)-1]
	next.Countries = slices.Delete(next.Countries, 1, 2)

	got := CatalogRemovals(prev, next)
	want := []string{"subcategory poussins", "category porcins", "subcategory porcs", "subcategory porcs-charcutiers", "subcategory porcs-reproducteurs", "country CM region sud-ouest", "country " + prev.Countries[1].Code}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("removals %v miss %q", got, w)
		}
	}
	if len(got) != len(want) {
		t.Errorf("removals = %v, want %v", got, want)
	}
}
