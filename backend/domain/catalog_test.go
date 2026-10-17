package domain

import (
	"encoding/json"
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
	for _, name := range CatalogParts {
		if c.Versions[name] != 1 {
			t.Errorf("seed %s version = %d, want 1", name, c.Versions[name])
		}
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
	if s, err := c.SpeciesByKey("piglets"); err != nil || s.Unit.Key != "piglet" {
		t.Errorf("SpeciesByKey(porcs) = %+v, %v", s, err)
	}
	if _, err := c.SpeciesByKey("calves"); !errors.Is(err, ErrInvalid) {
		t.Errorf("a sub-category of an inactive category is offered: %v", err)
	}
	c.Categories[0].Subcategories[0].Active = false
	if _, err := c.SpeciesByKey(c.Categories[0].Subcategories[0].Key); !errors.Is(err, ErrInvalid) {
		t.Errorf("an inactive sub-category is still offered: %v", err)
	}
	if got := c.SpeciesOf("caprine"); !slices.Equal(got, []string{"goats", "kids"}) {
		t.Errorf("SpeciesOf(caprins) = %v", got)
	}

	if k, err := c.Country(""); err != nil || k.Code != DefaultCountry {
		t.Errorf("Country(\"\") = %+v, %v", k, err)
	}
	if _, err := c.Country("XX"); !errors.Is(err, ErrInvalid) {
		t.Errorf("Country(XX): err = %v", err)
	}

	cm, _ := c.Country("cm")
	for in, want := range map[string]string{"Extrême-Nord": "far-north", "far north": "far-north", "  OUEST ": "west", "North West": "north-west", "north-west": "north-west", "": ""} {
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
	want := []string{"subcategory chicks", "category swine", "subcategory piglets", "subcategory finishing-pigs", "subcategory breeding-pigs", "country CM region south-west", "country " + prev.Countries[1].Code}
	// The region's towns go with it.
	for _, city := range prev.Countries[0].Regions[len(prev.Countries[0].Regions)-1].Cities {
		want = append(want, "country CM city "+city.Key)
	}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("removals %v miss %q", got, w)
		}
	}
	if len(got) != len(want) {
		t.Errorf("removals = %v, want %v", got, want)
	}
}

func TestCatalogCountry_CanonicalCity(t *testing.T) {
	cm, _ := SeedCatalog().Country("CM")
	for _, tc := range []struct{ region, town, city, wantRegion string }{
		{"west", "Bafoussam", "bafoussam", "west"},
		{"", "ngaoundere", "ngaoundere", "adamawa"},        // the town settles the region
		{"littoral", "  EDEA ", "edea", "littoral"},        // accents and case don't matter
		{"littoral", "Bafoussam", "Bafoussam", "littoral"}, // listed elsewhere: kept as typed
		{"west", "Bamendjou", "Bamendjou", "west"},         // not listed: kept as typed
	} {
		city, region, err := cm.CanonicalCity(tc.region, tc.town)
		if err != nil || city != tc.city || region != tc.wantRegion {
			t.Errorf("CanonicalCity(%q, %q) = %q, %q, %v; want %q, %q", tc.region, tc.town, city, region, err, tc.city, tc.wantRegion)
		}
	}
	if _, _, err := cm.CanonicalCity("west", strings.Repeat("x", MaxTownLen+1)); !errors.Is(err, ErrInvalid) {
		t.Errorf("long town: err = %v", err)
	}
}

func TestCatalog_PlaceKeys_FindsKeysFromEitherLanguage(t *testing.T) {
	c := SeedCatalog()
	if got := c.PlaceKeys("far north"); !slices.Equal(got, []string{"far-north"}) {
		t.Errorf("PlaceKeys(far north) = %v", got)
	}
	if got := c.PlaceKeys("Yaoundé"); !slices.Equal(got, []string{"yaounde"}) {
		t.Errorf("PlaceKeys(Yaoundé) = %v", got)
	}
}

func TestLabel_ReadsAPlainStringAsEveryLanguage(t *testing.T) {
	var r CatalogRegion
	if err := json.Unmarshal([]byte(`{"key":"west","label":"Ouest","active":true}`), &r); err != nil {
		t.Fatal(err)
	}
	if r.Label != (Label{FR: "Ouest", EN: "Ouest"}) {
		t.Errorf("Label = %+v", r.Label)
	}
}

func TestCatalog_Validate_RequiresKnownIcons(t *testing.T) {
	c := SeedCatalog()
	c.Categories[0].Icon = "dragon"
	c.Categories[1].Subcategories[0].Icon = ""
	err := c.Validate()
	for _, want := range []string{`category poultry: icon "dragon"`, `subcategory piglets: icon ""`} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Validate() = %v, want it to mention %s", err, want)
		}
	}
}

func TestCatalog_SpeciesKeys_MatchesNamesInBothLanguages(t *testing.T) {
	c := SeedCatalog()
	if got := c.SpeciesKeys("poussins"); !slices.Contains(got, "chicks") {
		t.Errorf(`SpeciesKeys("poussins") = %v, want chicks`, got)
	}
	if got := c.SpeciesKeys("oeufs"); !slices.Contains(got, "table-eggs") || !slices.Contains(got, "fertile-eggs") {
		t.Errorf(`SpeciesKeys("oeufs") = %v, want both kinds of eggs`, got)
	}
	if got := c.SpeciesKeys("volaille"); len(got) != len(c.Categories[0].Subcategories) {
		t.Errorf(`SpeciesKeys("volaille") = %v, want every poultry species`, got)
	}
	if got := c.SpeciesKeys("licorne"); got != nil {
		t.Errorf(`SpeciesKeys("licorne") = %v, want none`, got)
	}
}
