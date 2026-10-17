package domain

import (
	"encoding/json"
	"errors"
	"testing"
)

func raw(s string) json.RawMessage { return json.RawMessage(s) }

func TestCatalog_Apply_AddsUpdatesAndMoves(t *testing.T) {
	c := SeedCatalog()
	steps := []CatalogOp{
		{Kind: OpAdd, Ref: CatalogRef{Entity: EntitySubcategory, Parent: "poultry"}, Value: raw(`{"key":"oies","label":{"fr":"Oies","en":"Geese"},"cycleType":"rearing","icon":"duck","unit":{"key":"oie","label":{"fr":"oie","en":"goose"}},"active":true}`)},
		{Kind: OpAdd, Ref: CatalogRef{Entity: EntityCity, Country: "CM", Parent: "west"}, Value: raw(`{"key":"bamendjou","label":{"fr":"Bamendjou","en":"Bamendjou"},"active":true}`)},
		{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityCategory, Key: "poultry"}, Value: raw(`{"label":{"en":"Fowl"}}`)},
		{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityTerm, List: "reviewTags", Key: "good_value"}, Value: raw(`{"active":false}`)},
		{Kind: OpMove, Ref: CatalogRef{Entity: EntityCategory, Key: "rabbits"}, Position: new(0)},
		{Kind: OpMove, Ref: CatalogRef{Entity: EntityCity, Country: "CM", Key: "bamendjou", Parent: "littoral"}, Position: new(0)},
	}
	for i, op := range steps {
		if err := c.Apply(op); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("result invalid: %v", err)
	}
	poultry := c.Categories[c.categoryIndex("poultry")]
	if poultry.Label != (Label{FR: "Volaille", EN: "Fowl"}) {
		t.Errorf("partial label edit = %+v, want the French kept", poultry.Label)
	}
	if c.Categories[0].Key != "rabbits" {
		t.Errorf("first category = %s, want lapins", c.Categories[0].Key)
	}
	cm := c.Countries[c.countryIndex("CM")]
	if r := cm.Regions[regionIndex(&cm, "littoral")]; r.Cities[0].Key != "bamendjou" {
		t.Errorf("littoral starts with %s, want the town moved in", r.Cities[0].Key)
	}
}

func TestCatalog_Apply_Refusals(t *testing.T) {
	for _, tc := range []struct {
		name string
		op   CatalogOp
		code string
	}{
		{"a key already used", CatalogOp{Kind: OpAdd, Ref: CatalogRef{Entity: EntityTerm, List: "reviewTags"}, Value: raw(`{"key":"good_value","label":{"fr":"x","en":"x"},"active":true}`)}, "catalog.op_key_taken"},
		{"a kind named like a category", CatalogOp{Kind: OpAdd, Ref: CatalogRef{Entity: EntitySubcategory, Parent: "ovine"}, Value: raw(`{"key":"poultry","label":{"fr":"x","en":"x"},"cycleType":"rearing","icon":"hen","unit":{"key":"x","label":{"fr":"x","en":"x"}},"active":true}`)}, "catalog.op_key_taken"},
		{"a renamed key", CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityCategory, Key: "ovine"}, Value: raw(`{"key":"moutons2"}`)}, "catalog.op_key_immutable"},
		{"children in an update", CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityCategory, Key: "ovine"}, Value: raw(`{"subcategories":[]}`)}, "catalog.op_invalid"},
		{"an unknown field", CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityCategory, Key: "ovine"}, Value: raw(`{"colour":"sky"}`)}, "catalog.op_invalid"},
		{"a missing entry", CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityCity, Country: "CM", Key: "atlantis"}, Value: raw(`{"active":false}`)}, "catalog.op_target_missing"},
		{"a missing parent", CatalogOp{Kind: OpAdd, Ref: CatalogRef{Entity: EntityCity, Country: "CM", Parent: "atlantique"}, Value: raw(`{"key":"x","label":{"fr":"x","en":"x"},"active":true}`)}, "catalog.op_target_missing"},
	} {
		err := SeedCatalog().Apply(tc.op)
		if code, _ := CodeOf(err); code != tc.code {
			t.Errorf("%s: code = %q (%v), want %s", tc.name, code, err, tc.code)
		}
	}
}

func jsonOf(c *Catalog) string {
	b, _ := json.Marshal(catalogListsOf(c))
	return string(b)
}

func catalogListsOf(c *Catalog) any {
	return []any{c.Categories, c.Countries, c.ForumCategories, c.ReviewTags, c.ReportReasons, c.FarmReviewChecks}
}

// Replaying the diff turns the old lists into the new ones exactly.
func TestDiff_ReplaysToTheTarget(t *testing.T) {
	from := SeedCatalog()
	to := SeedCatalog()
	to.Categories[0].Label.EN = "Fowl"
	to.Categories[0], to.Categories[1] = to.Categories[1], to.Categories[0]
	pig := to.Categories[0].Subcategories[0]
	to.Categories[0].Subcategories = to.Categories[0].Subcategories[1:]
	to.Categories[1].Subcategories = append(to.Categories[1].Subcategories, pig) // a kind moves category
	to.Categories = append(to.Categories, CatalogCategory{Key: "abeilles", Label: Label{FR: "Abeilles", EN: "Bees"}, Icon: "hen", Color: "amber", Active: true,
		Subcategories: []CatalogSubcategory{{Key: "essaims", Label: Label{FR: "Essaims", EN: "Swarms"}, CycleType: "rearing", Icon: "hen", Unit: CatalogUnit{Key: "essaim", Label: Label{FR: "essaim", EN: "swarm"}}, Active: true}}})
	cm := &to.Countries[0]
	cm.Regions[0].Cities = append(cm.Regions[0].Cities, CatalogCity{Key: "mbe", Label: Label{FR: "Mbé", EN: "Mbe"}, Active: true})
	cm.Regions[1].Label.EN = "Center"
	to.ReviewTags = append(to.ReviewTags[1:], to.ReviewTags[0])

	ops, err := Diff(from, to)
	if err != nil {
		t.Fatal(err)
	}
	replayed := from.Clone()
	for _, op := range ops {
		if err := replayed.Apply(op); err != nil {
			t.Fatalf("replay %+v: %v", op, err)
		}
	}
	if jsonOf(replayed) != jsonOf(to) {
		t.Errorf("replayed diff differs from the target\nops: %d", len(ops))
	}
	if same, _ := Diff(to, to); len(same) != 0 {
		t.Errorf("Diff of identical lists = %d ops, want none", len(same))
	}
}

// What the target leaves out is deactivated, never removed.
func TestDiff_DeactivatesWhatIsLeftOut(t *testing.T) {
	from := SeedCatalog()
	to := SeedCatalog()
	to.ForumCategories = to.ForumCategories[1:]
	to.Countries[0].Regions[0].Cities = to.Countries[0].Regions[0].Cities[1:]
	ops, err := Diff(from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 2 {
		t.Fatalf("ops = %+v, want two deactivations", ops)
	}
	for _, op := range ops {
		if op.Kind != OpUpdate || string(op.Value) != `{"active":false}` {
			t.Errorf("op = %+v, want a deactivation", op)
		}
	}
	replayed := from.Clone()
	for _, op := range ops {
		_ = replayed.Apply(op)
	}
	if len(CatalogRemovals(from, replayed)) != 0 || errors.Is(replayed.Validate(), ErrInvalid) {
		t.Errorf("deactivating removed something or broke the lists")
	}
}
