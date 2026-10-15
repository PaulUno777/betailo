package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// The references the catalog is stored as. Each has its own schema
// (CatalogPartSchema), its own versions and its own endpoint, so a
// client loads, and a change republishes, only what it touches.
const (
	PartAnimals          = "animals"
	PartPlaces           = "places"
	PartForumCategories  = "forumCategories"
	PartReviewTags       = "reviewTags"
	PartReportReasons    = "reportReasons"
	PartFarmReviewChecks = "farmReviewChecks"
)

// CatalogParts lists every reference, in the order they are shown.
var CatalogParts = []string{PartAnimals, PartPlaces, PartForumCategories, PartReviewTags, PartReportReasons, PartFarmReviewChecks}

// CatalogPart is one published version of one reference: its content
// (as PartContent writes it) and who published it, when, and why.
type CatalogPart struct {
	Name        string
	Version     int
	Note        string
	PublishedBy bson.ObjectID
	PublishedAt time.Time
	// SeedRevision is the catalog_seed.json revision the version came
	// from (0 for one staff published).
	SeedRevision int
	Content      json.RawMessage
}

// IsPart is whether name is one of CatalogParts.
func IsPart(name string) bool {
	return slices.Contains(CatalogParts, name)
}

// PartOf is the reference an operation's entry belongs to.
func PartOf(r CatalogRef) string {
	switch r.Entity {
	case EntityCategory, EntitySubcategory:
		return PartAnimals
	case EntityCountry, EntityRegion, EntityCity:
		return PartPlaces
	default:
		return r.List
	}
}

type animalsContent struct {
	Categories []CatalogCategory `json:"categories"`
}

type placesContent struct {
	Countries []CatalogCountry `json:"countries"`
}

type termsContent struct {
	Entries []CatalogTerm `json:"entries"`
}

// PartContent is reference name's content in c: {"categories": [...]}
// for the animals, {"countries": [...]} for the places, and
// {"entries": [...]} for a vocabulary.
func (c *Catalog) PartContent(name string) (json.RawMessage, error) {
	var v any
	switch name {
	case PartAnimals:
		v = animalsContent{Categories: nonNil(c.Categories)}
	case PartPlaces:
		v = placesContent{Countries: nonNil(c.Countries)}
	default:
		list := c.termList(name)
		if list == nil {
			return nil, opError("catalog.part_unknown", "unknown reference %q", name)
		}
		v = termsContent{Entries: nonNil(*list)}
	}
	return json.Marshal(v)
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// SetPart replaces reference name in c with content, read strictly (an
// unknown field is refused).
func (c *Catalog) SetPart(name string, content json.RawMessage) error {
	switch name {
	case PartAnimals:
		var v animalsContent
		if err := decode(content, &v); err != nil {
			return err
		}
		c.Categories = v.Categories
	case PartPlaces:
		var v placesContent
		if err := decode(content, &v); err != nil {
			return err
		}
		c.Countries = v.Countries
	default:
		list := c.termList(name)
		if list == nil {
			return opError("catalog.part_unknown", "unknown reference %q", name)
		}
		var v termsContent
		if err := decode(content, &v); err != nil {
			return err
		}
		*list = v.Entries
	}
	return nil
}

// AssembleCatalog puts the latest version of each reference together.
// A reference missing from parts is taken from the seed.
func AssembleCatalog(parts map[string]*CatalogPart) (*Catalog, error) {
	c := SeedCatalog()
	for _, name := range CatalogParts {
		p, ok := parts[name]
		if !ok {
			continue
		}
		if err := c.SetPart(name, p.Content); err != nil {
			return nil, fmt.Errorf("catalog: reference %s v%d: %w", name, p.Version, err)
		}
		c.Versions[name] = p.Version
	}
	return c, nil
}

// ChangedParts is the references whose content differs between a and b.
func ChangedParts(a, b *Catalog) []string {
	var out []string
	for _, name := range CatalogParts {
		x, _ := a.PartContent(name)
		y, _ := b.PartContent(name)
		if !bytes.Equal(x, y) {
			out = append(out, name)
		}
	}
	return out
}
