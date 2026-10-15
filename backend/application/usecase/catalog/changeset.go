package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Changeset is how super admins change the reference data: operations
// (add an entry, edit some of its fields, move it) accumulate in one
// shared changeset, each checked against the cumulative result when it
// is queued, then are published together as the next version, replayed
// onto whatever version is in force by then.
type Changeset struct {
	Versions ports.CatalogRepository
	Pending  ports.CatalogChangesetRepository
	Publish  *Publish
	Clock    ports.Clock
}

// ChangeView is a queued change as the review shows it.
type ChangeView struct {
	domain.CatalogChange
	// Before is, for an update, the values it replaces.
	Before json.RawMessage
	// Err is set when the change no longer applies to the version in
	// force (it was published differently meanwhile).
	Err error
}

// ChangesetView is the changeset replayed onto the version in force.
type ChangesetView struct {
	Rev     int
	Base    *domain.Catalog
	Draft   *domain.Catalog
	Changes []ChangeView
	// Problems are what Validate finds in the draft (normally none: each
	// change was checked when queued).
	Problems []string
}

// Touched is the references the changes act on, in CatalogParts order.
func (v *ChangesetView) Touched() []string {
	var out []string
	for _, name := range domain.CatalogParts {
		if slices.ContainsFunc(v.Changes, func(c ChangeView) bool { return domain.PartOf(c.Op.Ref) == name }) {
			out = append(out, name)
		}
	}
	return out
}

// Ready is whether the changeset can be published as it is.
func (v *ChangesetView) Ready() bool {
	if len(v.Changes) == 0 || len(v.Problems) > 0 {
		return false
	}
	return !slices.ContainsFunc(v.Changes, func(c ChangeView) bool { return c.Err != nil })
}

func (uc *Changeset) latest(ctx context.Context) (*domain.Catalog, error) {
	c, _, err := Current(ctx, uc.Versions)
	return c, err
}

func replay(base *domain.Catalog, cs *domain.CatalogChangeset) *ChangesetView {
	draft := base.Clone()
	view := &ChangesetView{Rev: cs.Rev, Base: base, Draft: draft}
	for _, ch := range cs.Changes {
		cv := ChangeView{CatalogChange: ch, Before: draft.Before(ch.Op)}
		cv.Err = draft.Apply(ch.Op)
		view.Changes = append(view.Changes, cv)
	}
	if err := draft.Validate(); err != nil {
		view.Problems = problemsOf(err)
	}
	return view
}

func problemsOf(err error) []string {
	if _, params := domain.CodeOf(err); params != nil {
		if ps, ok := params["problems"].([]string); ok {
			return ps
		}
	}
	return []string{err.Error()}
}

// View is the changeset as it stands, replayed onto the version in force.
func (uc *Changeset) View(ctx context.Context) (*ChangesetView, error) {
	base, err := uc.latest(ctx)
	if err != nil {
		return nil, err
	}
	cs, err := uc.Pending.Get(ctx)
	if err != nil {
		return nil, err
	}
	return replay(base, cs), nil
}

func (uc *Changeset) current(ctx context.Context, rev int) (*domain.CatalogChangeset, *ChangesetView, error) {
	base, err := uc.latest(ctx)
	if err != nil {
		return nil, nil, err
	}
	cs, err := uc.Pending.Get(ctx)
	if err != nil {
		return nil, nil, err
	}
	if cs.Rev != rev {
		return nil, nil, domain.CodedWith("catalog.changeset_stale", map[string]any{"rev": cs.Rev},
			fmt.Errorf("catalog: changeset is at revision %d, not %d: %w", cs.Rev, rev, domain.ErrConflict))
	}
	return cs, replay(base, cs), nil
}

// Queue adds ops to the changeset, all or none: each must apply to the
// draft as it stands, and the result must validate. A refusal names
// the operation at fault ("index").
func (uc *Changeset) Queue(ctx context.Context, rev int, ops []domain.CatalogOp, by bson.ObjectID) (*ChangesetView, error) {
	if len(ops) == 0 {
		return nil, domain.Coded("catalog.no_changes", fmt.Errorf("catalog: nothing to queue: %w", domain.ErrInvalid))
	}
	cs, view, err := uc.current(ctx, rev)
	if err != nil {
		return nil, err
	}
	draft := view.Draft.Clone()
	now := uc.Clock.Now()
	added := make([]domain.CatalogChange, 0, len(ops))
	for i, op := range ops {
		op = op.Normalize()
		if err := draft.Apply(op); err != nil {
			return nil, withIndex(err, i)
		}
		added = append(added, domain.CatalogChange{ID: bson.NewObjectID(), Op: op, By: by, At: now})
	}
	if err := draft.Validate(); err != nil {
		return nil, err
	}
	cs.Changes = append(cs.Changes, added...)
	cs.UpdatedAt = now
	if err := uc.Pending.Save(ctx, cs, rev); err != nil {
		return nil, err
	}
	return uc.View(ctx)
}

func withIndex(err error, i int) error {
	code, params := domain.CodeOf(err)
	if code == "" {
		return err
	}
	out := map[string]any{"index": i}
	for k, v := range params {
		out[k] = v
	}
	return domain.CodedWith(code, out, err)
}

// Undo takes one change back out. Changes that rely on it (a town added
// to a region it added) would no longer apply: the undo is refused and
// names them ("ids").
func (uc *Changeset) Undo(ctx context.Context, rev int, id bson.ObjectID) (*ChangesetView, error) {
	cs, view, err := uc.current(ctx, rev)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(cs.Changes, func(c domain.CatalogChange) bool { return c.ID == id })
	if i < 0 {
		return nil, domain.Coded("catalog.change_missing", fmt.Errorf("catalog: no change %s: %w", id.Hex(), domain.ErrNotFound))
	}
	rest := &domain.CatalogChangeset{Rev: cs.Rev, Changes: slices.Delete(slices.Clone(cs.Changes), i, i+1), UpdatedAt: uc.Clock.Now()}
	after := replay(view.Base, rest)
	var needed []string
	for _, c := range after.Changes {
		if c.Err != nil {
			needed = append(needed, c.ID.Hex())
		}
	}
	if len(needed) > 0 || len(after.Problems) > 0 {
		return nil, domain.CodedWith("catalog.change_needed", map[string]any{"ids": strings.Join(needed, ","), "problems": after.Problems},
			fmt.Errorf("catalog: other changes rely on %s: %w", id.Hex(), domain.ErrConflict))
	}
	if err := uc.Pending.Save(ctx, rest, rev); err != nil {
		return nil, err
	}
	return uc.View(ctx)
}

// Clear drops every pending change.
func (uc *Changeset) Clear(ctx context.Context, rev int) (*ChangesetView, error) {
	if _, _, err := uc.current(ctx, rev); err != nil {
		return nil, err
	}
	if err := uc.Pending.Save(ctx, &domain.CatalogChangeset{UpdatedAt: uc.Clock.Now()}, rev); err != nil {
		return nil, err
	}
	return uc.View(ctx)
}

// Import queues what turns reference name, in the draft, into content:
// one reference edited as JSON becomes ordinary changes (left-out
// entries are deactivated).
func (uc *Changeset) Import(ctx context.Context, rev int, name string, content json.RawMessage, by bson.ObjectID) (*ChangesetView, error) {
	if !domain.IsPart(name) {
		return nil, domain.Coded("catalog.part_unknown", fmt.Errorf("catalog: unknown reference %q: %w", name, domain.ErrNotFound))
	}
	_, view, err := uc.current(ctx, rev)
	if err != nil {
		return nil, err
	}
	target := view.Draft.Clone()
	if err := target.SetPart(name, content); err != nil {
		return nil, err
	}
	ops, err := domain.Diff(view.Draft, target)
	if err != nil {
		return nil, err
	}
	if len(ops) == 0 {
		return view, nil
	}
	return uc.Queue(ctx, rev, ops, by)
}

// Restore queues what brings reference name, in the draft, back to an
// older version: its entries return as they were, what came later is
// deactivated.
func (uc *Changeset) Restore(ctx context.Context, rev int, name string, version int, by bson.ObjectID) (*ChangesetView, error) {
	old, err := uc.Versions.Get(ctx, name, version)
	if err != nil {
		return nil, err
	}
	return uc.Import(ctx, rev, name, old.Content, by)
}

// PublishAll publishes the changeset — a new version of each reference
// it touches — with a note, and empties it.
func (uc *Changeset) PublishAll(ctx context.Context, rev int, note string, by bson.ObjectID) (*Published, int, error) {
	cs, view, err := uc.current(ctx, rev)
	if err != nil {
		return nil, 0, err
	}
	if len(view.Changes) == 0 {
		return nil, 0, domain.Coded("catalog.no_changes", fmt.Errorf("catalog: nothing to publish: %w", domain.ErrInvalid))
	}
	if !view.Ready() {
		return nil, 0, domain.CodedWith("catalog.changeset_conflict", map[string]any{"problems": view.Problems},
			fmt.Errorf("catalog: the changeset doesn't apply to the versions in force: %w", domain.ErrConflict))
	}
	published, err := uc.Publish.Execute(ctx, PublishInput{Base: view.Base.Versions, Note: note, Lists: view.Draft, By: by})
	if err != nil {
		return nil, 0, err
	}
	if err := uc.Pending.Save(ctx, &domain.CatalogChangeset{UpdatedAt: uc.Clock.Now()}, rev); err != nil {
		return nil, 0, err
	}
	return published, len(cs.Changes), nil
}
