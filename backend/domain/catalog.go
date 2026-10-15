package domain

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Catalog is the platform's reference data: the vocabularies the super
// admin maintains (animal categories and their sub-categories, countries
// and their regions, forum categories, review tags, report reasons, the
// farm review checklist).
// It is one JSON document, published as numbered, immutable versions;
// the latest one is what the API validates against and serves on
// GET /reference.
//
// Behaviour stays in code: a sub-category picks one of CycleTypes, it
// doesn't define one.
type Catalog struct {
	Version     int           `json:"version"`
	Note        string        `json:"note,omitempty"`
	PublishedBy bson.ObjectID `json:"-"`
	PublishedAt time.Time     `json:"publishedAt"`

	Categories       []CatalogCategory `json:"categories"`
	Countries        []CatalogCountry  `json:"countries"`
	ForumCategories  []CatalogTerm     `json:"forumCategories"`
	ReviewTags       []CatalogTerm     `json:"reviewTags"`
	ReportReasons    []CatalogTerm     `json:"reportReasons"`
	FarmReviewChecks []CatalogTerm     `json:"farmReviewChecks"`
}

// Label is a text in each of the platform's languages.
type Label struct {
	FR string `json:"fr"`
	EN string `json:"en"`
}

// CatalogTerm is one entry of a plain vocabulary. Key is what gets
// stored and never changes once used; Active false hides it from new
// choices while what already uses it stays valid.
type CatalogTerm struct {
	Key    string `json:"key"`
	Label  Label  `json:"label"`
	Active bool   `json:"active"`
}

// CatalogCategory groups animals the way buyers browse (Volaille,
// Porcins, Caprins…), in display order. Its sub-categories are what is
// sold. Deactivating a category hides every one of them.
type CatalogCategory struct {
	Key           string               `json:"key"`
	Label         Label                `json:"label"`
	Color         string               `json:"color"`
	Active        bool                 `json:"active"`
	Subcategories []CatalogSubcategory `json:"subcategories"`
}

// CatalogSubcategory is a kind of animal sold (Poules pondeuses,
// Dindes, Agneaux…). Its Key is what cycles and listings store as their
// species, so it is unique across the whole catalog; it can move to
// another category, its listings follow. Finer detail (a breed) is the
// listing's own Breed.
type CatalogSubcategory struct {
	Key       string `json:"key"`
	Label     Label  `json:"label"`
	CycleType string `json:"cycleType"`
	// Unit is what one animal is called on a listing. Its Key is stored
	// on listings ("sujet"); Label is how it reads.
	Unit   CatalogUnit `json:"unit"`
	Active bool        `json:"active"`
}

// CatalogUnit names one animal of a species.
type CatalogUnit struct {
	Key   string `json:"key"`
	Label Label  `json:"label"`
}

// CatalogCountry is a country farms can be in. A country with Regions
// makes farms pick one of them; without, the region is typed freely.
type CatalogCountry struct {
	Code        string          `json:"code"`
	CallingCode string          `json:"callingCode"`
	Active      bool            `json:"active"`
	Regions     []CatalogRegion `json:"regions,omitempty"`
}

// CatalogRegion is a first-level region. Key is stable; Label is its
// usual written form, the same in every language.
type CatalogRegion struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Active bool   `json:"active"`
}

// CycleTypes are the kinds of cycle the code knows how to run. A species
// is raised in one of them.
var CycleTypes = []string{"couvee", "engraissement", "ponte", "reproduction", "elevage"}

// CategoryColors are the design system's tones a category can wear.
var CategoryColors = []string{"clay", "forest", "amber", "sky", "alert", "plum", "stone"}

//go:embed catalog_seed.json
var catalogSeed []byte

// SeedCatalog is version 1: the lists the platform shipped with, used
// when no version has been published yet.
func SeedCatalog() *Catalog {
	var c Catalog
	if err := json.Unmarshal(catalogSeed, &c); err != nil {
		panic(fmt.Sprintf("domain: catalog_seed.json: %v", err))
	}
	c.Version = 1
	return &c
}

// --- lookups the use cases validate against ---

// SpeciesByKey returns an active sub-category of an active category, or
// ErrInvalid: what can be sold now.
func (c *Catalog) SpeciesByKey(key string) (CatalogSubcategory, error) {
	for _, cat := range c.Categories {
		for _, s := range cat.Subcategories {
			if s.Key == key && s.Active && cat.Active {
				return s, nil
			}
		}
	}
	return CatalogSubcategory{}, Coded("catalog.species_unknown", fmt.Errorf("unknown species %q: %w", key, ErrInvalid))
}

// SpeciesOf is every sub-category key of a category, active or not, so
// a filter on it still finds older listings; nil for an unknown one.
func (c *Catalog) SpeciesOf(categoryKey string) []string {
	for _, cat := range c.Categories {
		if cat.Key == categoryKey {
			keys := make([]string, 0, len(cat.Subcategories))
			for _, s := range cat.Subcategories {
				keys = append(keys, s.Key)
			}
			return keys
		}
	}
	return nil
}

// speciesKeys is every sub-category key, across categories.
func (c *Catalog) speciesKeys() []string {
	var out []string
	for _, cat := range c.Categories {
		out = append(out, c.SpeciesOf(cat.Key)...)
	}
	return out
}

// Country returns an active country, or ErrInvalid. An empty code is
// DefaultCountry.
func (c *Catalog) Country(code string) (CatalogCountry, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		code = DefaultCountry
	}
	for _, k := range c.Countries {
		if k.Code == code && k.Active {
			return k, nil
		}
	}
	return CatalogCountry{}, Coded("farm.country_invalid", fmt.Errorf("farm: %q is not a country farms can be in: %w", code, ErrInvalid))
}

// CanonicalRegion returns region in its canonical form for the country:
// the label of one of its active regions (matched on key or label,
// ignoring case and accents) when it has a list, else the trimmed text.
// Empty stays empty (a farm may not say yet).
func (k CatalogCountry) CanonicalRegion(region string) (string, error) {
	region = strings.Join(strings.Fields(region), " ")
	if region == "" {
		return "", nil
	}
	if len(k.Regions) == 0 {
		if utf8.RuneCountInString(region) > MaxRegionLen {
			return "", Coded("farm.region_invalid", fmt.Errorf("farm: region must be at most %d characters: %w", MaxRegionLen, ErrInvalid))
		}
		return region, nil
	}
	folded := foldKey(region)
	for _, r := range k.Regions {
		if r.Active && (foldKey(r.Label) == folded || r.Key == folded) {
			return r.Label, nil
		}
	}
	return "", Coded("farm.region_invalid", fmt.Errorf("farm: %q is not a region of %s: %w", region, k.Code, ErrInvalid))
}

// ActiveKeys is the keys of the active terms, in order.
func ActiveKeys(terms []CatalogTerm) []string {
	var out []string
	for _, t := range terms {
		if t.Active {
			out = append(out, t.Key)
		}
	}
	return out
}

// --- publishing ---

var (
	catalogKey  = regexp.MustCompile(`^[a-z0-9]+([_-][a-z0-9]+)*$`)
	callingCode = regexp.MustCompile(`^[1-9][0-9]{0,3}$`)
)

// maxCatalogLabel bounds every label.
const maxCatalogLabel = 80

// Validate checks a catalog about to be published: well-formed and
// unique keys, both languages filled in, known cycle types and colours,
// ISO country codes with their calling code, and the default country
// present and active. All problems are reported at once, joined.
func (c *Catalog) Validate() error {
	var errs []error
	bad := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }
	label := func(where string, l Label) {
		for lang, text := range map[string]string{"fr": l.FR, "en": l.EN} {
			if t := strings.TrimSpace(text); t == "" || utf8.RuneCountInString(t) > maxCatalogLabel {
				bad("%s: the %s label must be 1 to %d characters", where, lang, maxCatalogLabel)
			}
		}
	}
	keys := func(list string, n int, key func(int) string) {
		seen := map[string]bool{}
		for i := 0; i < n; i++ {
			k := key(i)
			switch {
			case !catalogKey.MatchString(k):
				bad("%s[%d]: key %q must be lowercase letters and digits, joined by - or _", list, i, k)
			case seen[k]:
				bad("%s: key %q appears twice", list, k)
			}
			seen[k] = true
		}
	}

	if len(c.Categories) == 0 {
		bad("categories: at least one is needed")
	}
	keys("categories", len(c.Categories), func(i int) string { return c.Categories[i].Key })
	allSpecies := c.speciesKeys()
	keys("subcategories", len(allSpecies), func(i int) string { return allSpecies[i] })
	for _, cat := range c.Categories {
		where := "category " + cat.Key
		label(where, cat.Label)
		if !slices.Contains(CategoryColors, cat.Color) {
			bad("%s: color %q must be one of %s", where, cat.Color, strings.Join(CategoryColors, ", "))
		}
		if slices.Contains(allSpecies, cat.Key) {
			bad("%s: a sub-category already has this key", where)
		}
		if len(cat.Subcategories) == 0 {
			bad("%s: at least one sub-category is needed", where)
		}
		for _, s := range cat.Subcategories {
			where := "subcategory " + s.Key
			label(where, s.Label)
			if !slices.Contains(CycleTypes, s.CycleType) {
				bad("%s: cycleType %q must be one of %s", where, s.CycleType, strings.Join(CycleTypes, ", "))
			}
			if k := strings.TrimSpace(s.Unit.Key); k == "" || utf8.RuneCountInString(k) > 30 {
				bad("%s: unit key must be 1 to 30 characters", where)
			}
			label(where+" unit", s.Unit.Label)
		}
	}

	seenCountry := map[string]bool{}
	for _, k := range c.Countries {
		where := "country " + k.Code
		if len(k.Code) != 2 || !isASCIIUpper(k.Code[0]) || !isASCIIUpper(k.Code[1]) {
			bad("%s: code must be 2 capital letters (ISO 3166-1)", where)
		}
		if seenCountry[k.Code] {
			bad("countries: %q appears twice", k.Code)
		}
		seenCountry[k.Code] = true
		if !callingCode.MatchString(k.CallingCode) {
			bad("%s: callingCode %q must be 1 to 4 digits, without +", where, k.CallingCode)
		}
		keys(where+" regions", len(k.Regions), func(i int) string { return k.Regions[i].Key })
		seenLabel := map[string]bool{}
		for _, r := range k.Regions {
			if t := strings.TrimSpace(r.Label); t == "" || utf8.RuneCountInString(t) > MaxRegionLen {
				bad("%s region %s: label must be 1 to %d characters", where, r.Key, MaxRegionLen)
			}
			if seenLabel[foldKey(r.Label)] {
				bad("%s: region label %q appears twice", where, r.Label)
			}
			seenLabel[foldKey(r.Label)] = true
		}
	}
	if def, err := c.Country(DefaultCountry); err != nil || def.Code != DefaultCountry {
		bad("countries: %s, the default country, must be listed and active", DefaultCountry)
	}

	for _, list := range []struct {
		name  string
		terms []CatalogTerm
	}{{"forumCategories", c.ForumCategories}, {"reviewTags", c.ReviewTags}, {"reportReasons", c.ReportReasons}, {"farmReviewChecks", c.FarmReviewChecks}} {
		keys(list.name, len(list.terms), func(i int) string { return list.terms[i].Key })
		for _, t := range list.terms {
			label(list.name+" "+t.Key, t.Label)
		}
	}

	if len(errs) > 0 {
		return Coded("catalog.invalid", fmt.Errorf("catalog: %w: %w", ErrInvalid, errors.Join(errs...)))
	}
	return nil
}

// CatalogRemovals lists the keys next drops compared with prev. Keys
// are stored on farms, listings, posts and reviews, so an entry is never
// removed: it is deactivated instead. Publishing refuses a catalog that
// removes one.
func CatalogRemovals(prev, next *Catalog) []string {
	var out []string
	missing := func(list string, before, after []string) {
		for _, k := range before {
			if !slices.Contains(after, k) {
				out = append(out, list+" "+k)
			}
		}
	}
	categoryKeys := func(c *Catalog) (ks []string) {
		for _, cat := range c.Categories {
			ks = append(ks, cat.Key)
		}
		return
	}
	termKeys := func(ts []CatalogTerm) (ks []string) {
		for _, t := range ts {
			ks = append(ks, t.Key)
		}
		return
	}
	missing("category", categoryKeys(prev), categoryKeys(next))
	missing("subcategory", prev.speciesKeys(), next.speciesKeys())
	missing("forumCategories", termKeys(prev.ForumCategories), termKeys(next.ForumCategories))
	missing("reviewTags", termKeys(prev.ReviewTags), termKeys(next.ReviewTags))
	missing("reportReasons", termKeys(prev.ReportReasons), termKeys(next.ReportReasons))
	missing("farmReviewChecks", termKeys(prev.FarmReviewChecks), termKeys(next.FarmReviewChecks))
	nextCountries := map[string]CatalogCountry{}
	for _, k := range next.Countries {
		nextCountries[k.Code] = k
	}
	for _, k := range prev.Countries {
		after, ok := nextCountries[k.Code]
		if !ok {
			out = append(out, "country "+k.Code)
			continue
		}
		var before, kept []string
		for _, r := range k.Regions {
			before = append(before, r.Key)
		}
		for _, r := range after.Regions {
			kept = append(kept, r.Key)
		}
		missing("country "+k.Code+" region", before, kept)
	}
	return out
}
