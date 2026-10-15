package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
)

// CatalogEntity is the kind of entry a catalog operation acts on.
type CatalogEntity string

const (
	EntityCategory    CatalogEntity = "category"
	EntitySubcategory CatalogEntity = "subcategory"
	EntityCountry     CatalogEntity = "country"
	EntityRegion      CatalogEntity = "region"
	EntityCity        CatalogEntity = "city"
	EntityTerm        CatalogEntity = "term"
)

// CatalogOpKind is what an operation does.
type CatalogOpKind string

const (
	// OpAdd adds a new entry under its parent, at the end of its list.
	OpAdd CatalogOpKind = "add"
	// OpUpdate changes some fields of an entry (a partial edit). Its key
	// never changes, and its children are edited as their own entries.
	OpUpdate CatalogOpKind = "update"
	// OpMove puts an entry at Position in its list, under Ref.Parent
	// when that changes (a kind to another category, a town to another
	// region).
	OpMove CatalogOpKind = "move"
)

// TermLists names the plain vocabularies an EntityTerm lives in.
var TermLists = []string{"forumCategories", "reviewTags", "reportReasons", "farmReviewChecks"}

// CatalogRef names one entry: its kind, its key (a country's code), and
// where it lives — the country of a region or town, the vocabulary of a
// term, and for an add or a move the parent (the category of a kind,
// the region of a town).
type CatalogRef struct {
	Entity  CatalogEntity `json:"entity"`
	Key     string        `json:"key"`
	Country string        `json:"country,omitempty"`
	List    string        `json:"list,omitempty"`
	Parent  string        `json:"parent,omitempty"`
}

// CatalogOp is one change to the reference data. Changes accumulate in
// a changeset and are applied, in order, to the version in force when
// it is published.
type CatalogOp struct {
	Kind CatalogOpKind `json:"kind"`
	Ref  CatalogRef    `json:"ref"`
	// Value is, for an add, the whole entry; for an update, only the
	// fields that change ({"label": {...}, "active": false}).
	Value json.RawMessage `json:"value,omitempty"`
	// Position is a move's place in its list, from 0.
	Position *int `json:"position,omitempty"`
}

func opError(code string, format string, args ...any) error {
	return Coded(code, fmt.Errorf("catalog: "+format+": %w", append(args, ErrInvalid)...))
}

// children are the fields an update can't touch: they are entries of
// their own.
var children = map[CatalogEntity]string{EntityCategory: "subcategories", EntityCountry: "regions", EntityRegion: "cities"}

// Apply performs op on c. It checks that the target exists (or, for an
// add, that its key is free in its scope) and that an update keeps the
// key; Validate checks the result as a whole.
func (c *Catalog) Apply(op CatalogOp) error {
	r := op.Ref
	switch op.Kind {
	case OpAdd:
		return c.add(r, op.Value)
	case OpUpdate:
		return c.update(r, op.Value)
	case OpMove:
		if op.Position == nil {
			return opError("catalog.op_invalid", "a move needs a position")
		}
		return c.move(r, *op.Position)
	default:
		return opError("catalog.op_invalid", "unknown operation %q", op.Kind)
	}
}

// --- finding entries ----------------------------------------------------

func (c *Catalog) categoryIndex(key string) int {
	return slices.IndexFunc(c.Categories, func(x CatalogCategory) bool { return x.Key == key })
}

func (c *Catalog) subcategoryIndex(key string) (int, int) {
	for i, cat := range c.Categories {
		if j := slices.IndexFunc(cat.Subcategories, func(s CatalogSubcategory) bool { return s.Key == key }); j >= 0 {
			return i, j
		}
	}
	return -1, -1
}

func (c *Catalog) countryIndex(code string) int {
	return slices.IndexFunc(c.Countries, func(x CatalogCountry) bool { return x.Code == code })
}

func regionIndex(k *CatalogCountry, key string) int {
	return slices.IndexFunc(k.Regions, func(x CatalogRegion) bool { return x.Key == key })
}

func cityIndex(k *CatalogCountry, key string) (int, int) {
	for i, r := range k.Regions {
		if j := slices.IndexFunc(r.Cities, func(x CatalogCity) bool { return x.Key == key }); j >= 0 {
			return i, j
		}
	}
	return -1, -1
}

func (c *Catalog) termList(name string) *[]CatalogTerm {
	switch name {
	case "forumCategories":
		return &c.ForumCategories
	case "reviewTags":
		return &c.ReviewTags
	case "reportReasons":
		return &c.ReportReasons
	case "farmReviewChecks":
		return &c.FarmReviewChecks
	}
	return nil
}

func missing(r CatalogRef) error {
	return opError("catalog.op_target_missing", "%s %q not found", r.Entity, r.Key)
}

func taken(r CatalogRef) error {
	return CodedWith("catalog.op_key_taken", map[string]any{"key": r.Key}, fmt.Errorf("catalog: %s key %q is already used: %w", r.Entity, r.Key, ErrConflict))
}

func (c *Catalog) country(r CatalogRef) (*CatalogCountry, error) {
	i := c.countryIndex(r.Country)
	if i < 0 {
		return nil, opError("catalog.op_target_missing", "country %q not found", r.Country)
	}
	return &c.Countries[i], nil
}

// --- add ------------------------------------------------------------------

func decode[T any](raw json.RawMessage, into *T) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return opError("catalog.op_invalid", "value: %v", err)
	}
	return nil
}

func (c *Catalog) add(r CatalogRef, raw json.RawMessage) error {
	switch r.Entity {
	case EntityCategory:
		var v CatalogCategory
		if err := decode(raw, &v); err != nil {
			return err
		}
		if c.categoryIndex(v.Key) >= 0 || slices.Contains(c.speciesKeys(), v.Key) {
			return taken(CatalogRef{Entity: r.Entity, Key: v.Key})
		}
		for _, s := range v.Subcategories {
			if i, _ := c.subcategoryIndex(s.Key); i >= 0 {
				return taken(CatalogRef{Entity: EntitySubcategory, Key: s.Key})
			}
		}
		c.Categories = append(c.Categories, v)
	case EntitySubcategory:
		var v CatalogSubcategory
		if err := decode(raw, &v); err != nil {
			return err
		}
		i := c.categoryIndex(r.Parent)
		if i < 0 {
			return opError("catalog.op_target_missing", "category %q not found", r.Parent)
		}
		if j, _ := c.subcategoryIndex(v.Key); j >= 0 || c.categoryIndex(v.Key) >= 0 {
			return taken(CatalogRef{Entity: r.Entity, Key: v.Key})
		}
		c.Categories[i].Subcategories = append(c.Categories[i].Subcategories, v)
	case EntityCountry:
		var v CatalogCountry
		if err := decode(raw, &v); err != nil {
			return err
		}
		if c.countryIndex(v.Code) >= 0 {
			return taken(CatalogRef{Entity: r.Entity, Key: v.Code})
		}
		c.Countries = append(c.Countries, v)
	case EntityRegion:
		var v CatalogRegion
		if err := decode(raw, &v); err != nil {
			return err
		}
		k, err := c.country(r)
		if err != nil {
			return err
		}
		if regionIndex(k, v.Key) >= 0 {
			return taken(CatalogRef{Entity: r.Entity, Key: v.Key})
		}
		for _, city := range v.Cities {
			if i, _ := cityIndex(k, city.Key); i >= 0 {
				return taken(CatalogRef{Entity: EntityCity, Key: city.Key})
			}
		}
		k.Regions = append(k.Regions, v)
	case EntityCity:
		var v CatalogCity
		if err := decode(raw, &v); err != nil {
			return err
		}
		k, err := c.country(r)
		if err != nil {
			return err
		}
		ri := regionIndex(k, r.Parent)
		if ri < 0 {
			return opError("catalog.op_target_missing", "region %q not found", r.Parent)
		}
		if i, _ := cityIndex(k, v.Key); i >= 0 {
			return taken(CatalogRef{Entity: r.Entity, Key: v.Key})
		}
		k.Regions[ri].Cities = append(k.Regions[ri].Cities, v)
	case EntityTerm:
		var v CatalogTerm
		if err := decode(raw, &v); err != nil {
			return err
		}
		list := c.termList(r.List)
		if list == nil {
			return opError("catalog.op_invalid", "unknown list %q", r.List)
		}
		if slices.ContainsFunc(*list, func(t CatalogTerm) bool { return t.Key == v.Key }) {
			return taken(CatalogRef{Entity: r.Entity, Key: v.Key})
		}
		*list = append(*list, v)
	default:
		return opError("catalog.op_invalid", "unknown entity %q", r.Entity)
	}
	return nil
}

// --- update ---------------------------------------------------------------

// patch merges the fields of raw into *entry: an object's fields are
// merged one level down ({"label": {"en": "Fowl"}} keeps the French),
// the key and the children are refused.
func patch[T any](entity CatalogEntity, entry *T, raw json.RawMessage) error {
	var changes map[string]json.RawMessage
	if err := json.Unmarshal(raw, &changes); err != nil || changes == nil {
		return opError("catalog.op_invalid", "an update's value must be an object of the fields that change")
	}
	current, _ := json.Marshal(entry)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(current, &fields)
	for name, value := range changes {
		switch {
		case name == "key" || name == "code":
			if !bytes.Equal(bytes.TrimSpace(value), bytes.TrimSpace(fields[name])) {
				return opError("catalog.op_key_immutable", "a key never changes once published")
			}
		case name == children[entity]:
			return opError("catalog.op_invalid", "%s are edited as their own entries", name)
		default:
			var inner map[string]json.RawMessage
			var before map[string]json.RawMessage
			if json.Unmarshal(value, &inner) == nil && inner != nil && json.Unmarshal(fields[name], &before) == nil && before != nil {
				for k, v := range inner {
					before[k] = v
				}
				value, _ = json.Marshal(before)
			}
			fields[name] = value
		}
	}
	merged, _ := json.Marshal(fields)
	var next T
	if err := decode(merged, &next); err != nil {
		return err
	}
	*entry = next
	return nil
}

func (c *Catalog) update(r CatalogRef, raw json.RawMessage) error {
	switch r.Entity {
	case EntityCategory:
		if i := c.categoryIndex(r.Key); i >= 0 {
			return patch(r.Entity, &c.Categories[i], raw)
		}
	case EntitySubcategory:
		if i, j := c.subcategoryIndex(r.Key); i >= 0 {
			return patch(r.Entity, &c.Categories[i].Subcategories[j], raw)
		}
	case EntityCountry:
		if i := c.countryIndex(r.Key); i >= 0 {
			return patch(r.Entity, &c.Countries[i], raw)
		}
	case EntityRegion:
		k, err := c.country(r)
		if err != nil {
			return err
		}
		if i := regionIndex(k, r.Key); i >= 0 {
			return patch(r.Entity, &k.Regions[i], raw)
		}
	case EntityCity:
		k, err := c.country(r)
		if err != nil {
			return err
		}
		if i, j := cityIndex(k, r.Key); i >= 0 {
			return patch(r.Entity, &k.Regions[i].Cities[j], raw)
		}
	case EntityTerm:
		if list := c.termList(r.List); list != nil {
			if i := slices.IndexFunc(*list, func(t CatalogTerm) bool { return t.Key == r.Key }); i >= 0 {
				return patch(r.Entity, &(*list)[i], raw)
			}
		}
	default:
		return opError("catalog.op_invalid", "unknown entity %q", r.Entity)
	}
	return missing(r)
}

// --- move -----------------------------------------------------------------

// moveIn takes entry i out of from and puts it at pos in to (clamped).
func moveIn[T any](from *[]T, i int, to *[]T, pos int) {
	item := (*from)[i]
	*from = slices.Delete(*from, i, i+1)
	pos = max(0, min(pos, len(*to)))
	*to = slices.Insert(*to, pos, item)
}

func (c *Catalog) move(r CatalogRef, pos int) error {
	switch r.Entity {
	case EntityCategory:
		if i := c.categoryIndex(r.Key); i >= 0 {
			moveIn(&c.Categories, i, &c.Categories, pos)
			return nil
		}
	case EntitySubcategory:
		if i, j := c.subcategoryIndex(r.Key); i >= 0 {
			to := i
			if r.Parent != "" {
				if to = c.categoryIndex(r.Parent); to < 0 {
					return opError("catalog.op_target_missing", "category %q not found", r.Parent)
				}
			}
			moveIn(&c.Categories[i].Subcategories, j, &c.Categories[to].Subcategories, pos)
			return nil
		}
	case EntityCountry:
		if i := c.countryIndex(r.Key); i >= 0 {
			moveIn(&c.Countries, i, &c.Countries, pos)
			return nil
		}
	case EntityRegion:
		k, err := c.country(r)
		if err != nil {
			return err
		}
		if i := regionIndex(k, r.Key); i >= 0 {
			moveIn(&k.Regions, i, &k.Regions, pos)
			return nil
		}
	case EntityCity:
		k, err := c.country(r)
		if err != nil {
			return err
		}
		if i, j := cityIndex(k, r.Key); i >= 0 {
			to := i
			if r.Parent != "" {
				if to = regionIndex(k, r.Parent); to < 0 {
					return opError("catalog.op_target_missing", "region %q not found", r.Parent)
				}
			}
			moveIn(&k.Regions[i].Cities, j, &k.Regions[to].Cities, pos)
			return nil
		}
	case EntityTerm:
		if list := c.termList(r.List); list != nil {
			if i := slices.IndexFunc(*list, func(t CatalogTerm) bool { return t.Key == r.Key }); i >= 0 {
				moveIn(list, i, list, pos)
				return nil
			}
		}
	default:
		return opError("catalog.op_invalid", "unknown entity %q", r.Entity)
	}
	return missing(r)
}

// --- diff -------------------------------------------------------------------

// Clone is a deep copy of c's lists.
func (c *Catalog) Clone() *Catalog {
	raw, _ := json.Marshal(c)
	var out Catalog
	_ = json.Unmarshal(raw, &out)
	return &out
}

// own is an entry's fields without its key and children, as JSON
// values, for comparing and patching.
func own(entity CatalogEntity, entry any) map[string]json.RawMessage {
	raw, _ := json.Marshal(entry)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	delete(fields, "key")
	delete(fields, "code")
	delete(fields, children[entity])
	return fields
}

// changed is the fields of to that differ from from: an update's value.
func changed(entity CatalogEntity, from, to any) json.RawMessage {
	a, b := own(entity, from), own(entity, to)
	out := map[string]json.RawMessage{}
	for name, v := range b {
		var x, y any
		_ = json.Unmarshal(a[name], &x)
		_ = json.Unmarshal(v, &y)
		if !reflect.DeepEqual(x, y) {
			out[name] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	raw, _ := json.Marshal(out)
	return raw
}

// Diff is the operations that turn from into to: additions, partial
// updates, moves and — since a published key is never removed —
// deactivation of whatever to leaves out. It backs importing a whole
// catalog and restoring an old version. Each operation is applied to a
// working copy as it is planned, so the list replays exactly.
func Diff(from, to *Catalog) ([]CatalogOp, error) {
	work := from.Clone()
	var ops []CatalogOp
	do := func(op CatalogOp) error {
		if err := work.Apply(op); err != nil {
			return err
		}
		ops = append(ops, op)
		return nil
	}
	val := func(v any) json.RawMessage { raw, _ := json.Marshal(v); return raw }
	// after keeps the target's order without touching entries it leaves
	// out (they stay where they are, deactivated): an entry moves only
	// when it sits before the one the target puts just ahead of it, or
	// when it changes parent. at is the entry's index, prev its
	// predecessor's (-1 for the first), in the same list.
	after := func(at, prev int, moved bool) *int {
		switch {
		case moved && prev < 0:
			return new(0)
		case moved:
			return new(prev + 1)
		case prev >= 0 && at < prev:
			return new(prev) // at is taken out first: prev shifts down one
		}
		return nil
	}
	deactivate := json.RawMessage(`{"active":false}`)

	// Categories and their kinds.
	for ci, cat := range to.Categories {
		if work.categoryIndex(cat.Key) < 0 {
			// New: added with its new kinds; kinds that exist elsewhere move in below.
			fresh := cat
			fresh.Subcategories = nil
			for _, s := range cat.Subcategories {
				if i, _ := work.subcategoryIndex(s.Key); i < 0 {
					fresh.Subcategories = append(fresh.Subcategories, s)
				}
			}
			if err := do(CatalogOp{Kind: OpAdd, Ref: CatalogRef{Entity: EntityCategory, Key: cat.Key}, Value: val(fresh)}); err != nil {
				return nil, err
			}
		} else if v := changed(EntityCategory, work.Categories[work.categoryIndex(cat.Key)], cat); v != nil {
			if err := do(CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityCategory, Key: cat.Key}, Value: v}); err != nil {
				return nil, err
			}
		}
		if p := after(work.categoryIndex(cat.Key), prevIndex(ci, to.Categories, work.categoryIndex), false); p != nil {
			if err := do(CatalogOp{Kind: OpMove, Ref: CatalogRef{Entity: EntityCategory, Key: cat.Key}, Position: p}); err != nil {
				return nil, err
			}
		}
		for si, s := range cat.Subcategories {
			i, j := work.subcategoryIndex(s.Key)
			if i < 0 {
				if err := do(CatalogOp{Kind: OpAdd, Ref: CatalogRef{Entity: EntitySubcategory, Key: s.Key, Parent: cat.Key}, Value: val(s)}); err != nil {
					return nil, err
				}
			} else if v := changed(EntitySubcategory, work.Categories[i].Subcategories[j], s); v != nil {
				if err := do(CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntitySubcategory, Key: s.Key}, Value: v}); err != nil {
					return nil, err
				}
			}
			i, j = work.subcategoryIndex(s.Key)
			prev := -1
			if si > 0 {
				_, prev = work.subcategoryIndex(cat.Subcategories[si-1].Key)
			}
			if p := after(j, prev, work.Categories[i].Key != cat.Key); p != nil {
				if err := do(CatalogOp{Kind: OpMove, Ref: CatalogRef{Entity: EntitySubcategory, Key: s.Key, Parent: cat.Key}, Position: p}); err != nil {
					return nil, err
				}
			}
		}
	}
	for _, cat := range work.Categories {
		if to.categoryIndex(cat.Key) < 0 && cat.Active {
			if err := do(CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityCategory, Key: cat.Key}, Value: deactivate}); err != nil {
				return nil, err
			}
		}
		for _, s := range cat.Subcategories {
			if i, _ := to.subcategoryIndex(s.Key); i < 0 && s.Active {
				if err := do(CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntitySubcategory, Key: s.Key}, Value: deactivate}); err != nil {
					return nil, err
				}
			}
		}
	}

	// Countries, their regions and towns.
	for ki, k := range to.Countries {
		if work.countryIndex(k.Code) < 0 {
			if err := do(CatalogOp{Kind: OpAdd, Ref: CatalogRef{Entity: EntityCountry, Key: k.Code}, Value: val(k)}); err != nil {
				return nil, err
			}
		} else if v := changed(EntityCountry, work.Countries[work.countryIndex(k.Code)], k); v != nil {
			if err := do(CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityCountry, Key: k.Code}, Value: v}); err != nil {
				return nil, err
			}
		}
		if p := after(work.countryIndex(k.Code), prevIndex(ki, to.Countries, work.countryIndex), false); p != nil {
			if err := do(CatalogOp{Kind: OpMove, Ref: CatalogRef{Entity: EntityCountry, Key: k.Code}, Position: p}); err != nil {
				return nil, err
			}
		}
		wk := &work.Countries[work.countryIndex(k.Code)]
		for ri, r := range k.Regions {
			ref := CatalogRef{Entity: EntityRegion, Key: r.Key, Country: k.Code}
			if regionIndex(wk, r.Key) < 0 {
				fresh := r
				fresh.Cities = nil
				for _, city := range r.Cities {
					if i, _ := cityIndex(wk, city.Key); i < 0 {
						fresh.Cities = append(fresh.Cities, city)
					}
				}
				if err := do(CatalogOp{Kind: OpAdd, Ref: ref, Value: val(fresh)}); err != nil {
					return nil, err
				}
				wk = &work.Countries[work.countryIndex(k.Code)]
			} else if v := changed(EntityRegion, wk.Regions[regionIndex(wk, r.Key)], r); v != nil {
				if err := do(CatalogOp{Kind: OpUpdate, Ref: ref, Value: v}); err != nil {
					return nil, err
				}
			}
			prevRegion := -1
			if ri > 0 {
				prevRegion = regionIndex(wk, k.Regions[ri-1].Key)
			}
			if p := after(regionIndex(wk, r.Key), prevRegion, false); p != nil {
				if err := do(CatalogOp{Kind: OpMove, Ref: ref, Position: p}); err != nil {
					return nil, err
				}
			}
			for ti, city := range r.Cities {
				cref := CatalogRef{Entity: EntityCity, Key: city.Key, Country: k.Code, Parent: r.Key}
				i, j := cityIndex(wk, city.Key)
				if i < 0 {
					if err := do(CatalogOp{Kind: OpAdd, Ref: cref, Value: val(city)}); err != nil {
						return nil, err
					}
				} else if v := changed(EntityCity, wk.Regions[i].Cities[j], city); v != nil {
					if err := do(CatalogOp{Kind: OpUpdate, Ref: cref, Value: v}); err != nil {
						return nil, err
					}
				}
				i, j = cityIndex(wk, city.Key)
				prev := -1
				if ti > 0 {
					_, prev = cityIndex(wk, r.Cities[ti-1].Key)
				}
				if p := after(j, prev, wk.Regions[i].Key != r.Key); p != nil {
					if err := do(CatalogOp{Kind: OpMove, Ref: cref, Position: p}); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	for _, k := range work.Countries {
		tk := to.countryIndex(k.Code)
		if tk < 0 {
			if k.Active {
				if err := do(CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityCountry, Key: k.Code}, Value: deactivate}); err != nil {
					return nil, err
				}
			}
			continue
		}
		target := &to.Countries[tk]
		for _, r := range k.Regions {
			if regionIndex(target, r.Key) < 0 && r.Active {
				if err := do(CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityRegion, Key: r.Key, Country: k.Code}, Value: deactivate}); err != nil {
					return nil, err
				}
			}
			for _, city := range r.Cities {
				if i, _ := cityIndex(target, city.Key); i < 0 && city.Active {
					if err := do(CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityCity, Key: city.Key, Country: k.Code}, Value: deactivate}); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	// The vocabularies.
	for _, name := range TermLists {
		for ti, t := range *to.termList(name) {
			ref := CatalogRef{Entity: EntityTerm, Key: t.Key, List: name}
			list := work.termList(name)
			i := slices.IndexFunc(*list, func(x CatalogTerm) bool { return x.Key == t.Key })
			if i < 0 {
				if err := do(CatalogOp{Kind: OpAdd, Ref: ref, Value: val(t)}); err != nil {
					return nil, err
				}
			} else if v := changed(EntityTerm, (*list)[i], t); v != nil {
				if err := do(CatalogOp{Kind: OpUpdate, Ref: ref, Value: v}); err != nil {
					return nil, err
				}
			}
			index := func(key string) int {
				return slices.IndexFunc(*work.termList(name), func(x CatalogTerm) bool { return x.Key == key })
			}
			prev := -1
			if ti > 0 {
				prev = index((*to.termList(name))[ti-1].Key)
			}
			if p := after(index(t.Key), prev, false); p != nil {
				if err := do(CatalogOp{Kind: OpMove, Ref: ref, Position: p}); err != nil {
					return nil, err
				}
			}
		}
		for _, t := range *work.termList(name) {
			if !slices.ContainsFunc(*to.termList(name), func(x CatalogTerm) bool { return x.Key == t.Key }) && t.Active {
				if err := do(CatalogOp{Kind: OpUpdate, Ref: CatalogRef{Entity: EntityTerm, Key: t.Key, List: name}, Value: deactivate}); err != nil {
					return nil, err
				}
			}
		}
	}
	return ops, nil
}

// prevIndex is where, in the working list, the target puts the entry
// just ahead of entry i of target (-1 for the first).
func prevIndex[T interface {
	CatalogCategory | CatalogCountry
}](i int, target []T, index func(string) int) int {
	if i == 0 {
		return -1
	}
	switch prev := any(target[i-1]).(type) {
	case CatalogCategory:
		return index(prev.Key)
	case CatalogCountry:
		return index(prev.Code)
	}
	return -1
}

// Entry returns the entry ref names, as it is now.
func (c *Catalog) Entry(r CatalogRef) (any, bool) {
	switch r.Entity {
	case EntityCategory:
		if i := c.categoryIndex(r.Key); i >= 0 {
			return c.Categories[i], true
		}
	case EntitySubcategory:
		if i, j := c.subcategoryIndex(r.Key); i >= 0 {
			return c.Categories[i].Subcategories[j], true
		}
	case EntityCountry:
		if i := c.countryIndex(r.Key); i >= 0 {
			return c.Countries[i], true
		}
	case EntityRegion, EntityCity:
		i := c.countryIndex(r.Country)
		if i < 0 {
			return nil, false
		}
		k := &c.Countries[i]
		if r.Entity == EntityRegion {
			if j := regionIndex(k, r.Key); j >= 0 {
				return k.Regions[j], true
			}
		} else if a, b := cityIndex(k, r.Key); a >= 0 {
			return k.Regions[a].Cities[b], true
		}
	case EntityTerm:
		if list := c.termList(r.List); list != nil {
			if i := slices.IndexFunc(*list, func(t CatalogTerm) bool { return t.Key == r.Key }); i >= 0 {
				return (*list)[i], true
			}
		}
	}
	return nil, false
}

// Before is, for an update, the current values of the fields it
// changes (what the review shows as "was"); nil otherwise.
func (c *Catalog) Before(op CatalogOp) json.RawMessage {
	if op.Kind != OpUpdate {
		return nil
	}
	entry, ok := c.Entry(op.Ref)
	if !ok {
		return nil
	}
	var changes map[string]json.RawMessage
	if json.Unmarshal(op.Value, &changes) != nil {
		return nil
	}
	fields := own(op.Ref.Entity, entry)
	out := map[string]json.RawMessage{}
	for name := range changes {
		if v, ok := fields[name]; ok {
			out[name] = v
		}
	}
	raw, _ := json.Marshal(out)
	return raw
}

// Normalize fills an add's key from its value ("key", or a country's
// "code"), so every operation names its entry.
func (op CatalogOp) Normalize() CatalogOp {
	if op.Kind == OpAdd && op.Ref.Key == "" {
		var v struct {
			Key  string `json:"key"`
			Code string `json:"code"`
		}
		_ = json.Unmarshal(op.Value, &v)
		op.Ref.Key = v.Key
		if op.Ref.Entity == EntityCountry {
			op.Ref.Key = v.Code
		}
	}
	return op
}
