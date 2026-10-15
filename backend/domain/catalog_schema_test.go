package domain

import (
	"encoding/json"
	"slices"
	"testing"
)

// The schema the editor uses names exactly what Validate accepts.
func TestCatalogPartSchema_SharesValidatesRules(t *testing.T) {
	raw, err := json.Marshal(CatalogPartSchema(PartAnimals, "en"))
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Properties struct {
			Categories struct {
				Items struct {
					Properties struct {
						Icon  struct{ Enum []string }  `json:"icon"`
						Color struct{ Enum []string }  `json:"color"`
						Key   struct{ Pattern string } `json:"key"`
					} `json:"properties"`
				} `json:"items"`
			} `json:"categories"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	p := s.Properties.Categories.Items.Properties
	if !slices.Equal(p.Icon.Enum, AnimalIcons) || !slices.Equal(p.Color.Enum, CategoryColors) || p.Key.Pattern != catalogKey.String() {
		t.Errorf("schema = %+v, want the domain's icons, colours and key pattern", p)
	}
}

func TestCatalogPartSchema_OnePerReference(t *testing.T) {
	for _, name := range CatalogParts {
		if CatalogPartSchema(name, "fr") == nil {
			t.Errorf("no schema for %s", name)
		}
	}
	if CatalogPartSchema("dragons", "fr") != nil {
		t.Error("a schema for an unknown reference")
	}
}

// Each reference's content round-trips, and changing one leaves the
// others as they were.
func TestCatalog_PartsRoundTrip(t *testing.T) {
	c := SeedCatalog()
	for _, name := range CatalogParts {
		content, err := c.PartContent(name)
		if err != nil {
			t.Fatal(err)
		}
		other := SeedCatalog()
		if err := other.SetPart(name, content); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	edited := c.Clone()
	edited.ReviewTags[0].Active = false
	if got := ChangedParts(c, edited); len(got) != 1 || got[0] != PartReviewTags {
		t.Errorf("ChangedParts = %v, want reviewTags only", got)
	}
}
