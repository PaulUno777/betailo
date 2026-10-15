package catalog

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

type fakePending struct{ cs domain.CatalogChangeset }

func (f *fakePending) Get(context.Context) (*domain.CatalogChangeset, error) {
	cs := f.cs
	cs.Changes = append([]domain.CatalogChange(nil), f.cs.Changes...)
	return &cs, nil
}

func (f *fakePending) Save(_ context.Context, cs *domain.CatalogChangeset, rev int) error {
	if f.cs.Rev != rev {
		return domain.ErrConflict
	}
	f.cs = *cs
	f.cs.Rev = rev + 1
	return nil
}

func newChangeset() (*Changeset, *fakeRepo, *fakePending) {
	repo := seeded()
	pending := &fakePending{}
	clock := fixedClock{time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)}
	return &Changeset{Versions: repo, Pending: pending, Publish: &Publish{Repo: repo, Clock: clock}, Clock: clock}, repo, pending
}

func op(kind domain.CatalogOpKind, ref domain.CatalogRef, value string) domain.CatalogOp {
	return domain.CatalogOp{Kind: kind, Ref: ref, Value: json.RawMessage(value)}
}

var (
	addRegion = op(domain.OpAdd, domain.CatalogRef{Entity: domain.EntityRegion, Country: "SN"}, `{"key":"dakar","label":{"fr":"Dakar","en":"Dakar"},"active":true}`)
	addTown   = op(domain.OpAdd, domain.CatalogRef{Entity: domain.EntityCity, Country: "SN", Parent: "dakar"}, `{"key":"rufisque","label":{"fr":"Rufisque","en":"Rufisque"},"active":true}`)
	renameEN  = op(domain.OpUpdate, domain.CatalogRef{Entity: domain.EntityCategory, Key: "volaille"}, `{"label":{"en":"Fowl"}}`)
)

func TestChangeset_ChangesAccumulateAndPublishTogether(t *testing.T) {
	uc, repo, pending := newChangeset()
	ctx := context.Background()
	alice, bob := bson.NewObjectID(), bson.NewObjectID()

	v, err := uc.Queue(ctx, 0, []domain.CatalogOp{addRegion, addTown}, alice)
	if err != nil {
		t.Fatal(err)
	}
	if v, err = uc.Queue(ctx, v.Rev, []domain.CatalogOp{renameEN}, bob); err != nil {
		t.Fatal(err)
	}
	if len(v.Changes) != 3 || v.Changes[0].Op.Ref.Key != "dakar" || v.Changes[2].By != bob {
		t.Fatalf("changes = %+v", v.Changes)
	}
	if got := string(v.Changes[2].Before); got != `{"label":{"fr":"Volaille","en":"Poultry"}}` {
		t.Errorf("before = %s, want the label it replaces", got)
	}
	if !v.Ready() {
		t.Fatalf("not ready: %+v", v.Problems)
	}

	// Someone else's stale write is refused.
	if _, err := uc.Queue(ctx, 0, []domain.CatalogOp{renameEN}, bob); codeOf(err) != "catalog.changeset_stale" {
		t.Errorf("stale queue: %v", err)
	}

	if touched := v.Touched(); len(touched) != 2 || touched[0] != domain.PartAnimals || touched[1] != domain.PartPlaces {
		t.Errorf("touched = %v, want animals and places", touched)
	}
	published, n, err := uc.PublishAll(ctx, v.Rev, "Senegal and a rename", alice)
	if err != nil || n != 3 || len(published.Parts) != 2 || len(repo.parts[domain.PartAnimals]) != 2 || len(repo.parts[domain.PartReviewTags]) != 1 {
		t.Fatalf("PublishAll = %+v (%d changes), %v", published, n, err)
	}
	if len(pending.cs.Changes) != 0 {
		t.Errorf("changeset not emptied after publishing")
	}
}

func TestChangeset_RefusesWhatDoesNotApplyNamingTheOperation(t *testing.T) {
	uc, _, _ := newChangeset()
	badIcon := op(domain.OpUpdate, domain.CatalogRef{Entity: domain.EntityCategory, Key: "ovins"}, `{"icon":"dragon"}`)
	_, err := uc.Queue(context.Background(), 0, []domain.CatalogOp{renameEN, badIcon}, bson.NewObjectID())
	if codeOf(err) != "catalog.invalid" {
		t.Errorf("invalid result: %v", err)
	}
	_, err = uc.Queue(context.Background(), 0, []domain.CatalogOp{renameEN, addTown}, bson.NewObjectID())
	if code, params := domain.CodeOf(err); code != "catalog.op_target_missing" || params["index"] != 1 {
		t.Errorf("town in a missing region: %s %v", code, params)
	}
}

func TestChangeset_UndoRefusesWhatOtherChangesNeed(t *testing.T) {
	uc, _, _ := newChangeset()
	ctx := context.Background()
	v, _ := uc.Queue(ctx, 0, []domain.CatalogOp{addRegion, addTown}, bson.NewObjectID())
	if _, err := uc.Undo(ctx, v.Rev, v.Changes[0].ID); codeOf(err) != "catalog.change_needed" {
		t.Errorf("undo of a region its town needs: %v", err)
	}
	v, err := uc.Undo(ctx, v.Rev, v.Changes[1].ID)
	if err != nil || len(v.Changes) != 1 {
		t.Fatalf("undo of the town: %d changes, %v", len(v.Changes), err)
	}
}

func TestChangeset_ImportAndRestoreBecomeChanges(t *testing.T) {
	uc, _, _ := newChangeset()
	ctx := context.Background()
	edited := domain.SeedCatalog()
	edited.ReviewTags[0].Label.EN = "Healthy"
	edited.ForumCategories = edited.ForumCategories[1:]
	v, err := uc.Import(ctx, 0, domain.PartReviewTags, mustPart(t, edited, domain.PartReviewTags), bson.NewObjectID())
	if err != nil {
		t.Fatal(err)
	}
	v, err = uc.Import(ctx, v.Rev, domain.PartForumCategories, mustPart(t, edited, domain.PartForumCategories), bson.NewObjectID())
	if err != nil || len(v.Changes) != 2 {
		t.Fatalf("import = %d changes, %v; want an edit and a deactivation", len(v.Changes), err)
	}
	if _, _, err := uc.PublishAll(ctx, v.Rev, "x", bson.NewObjectID()); err != nil {
		t.Fatal(err)
	}

	// Back to version 1: the label returns, the category is active again.
	cur, _ := uc.View(ctx)
	v, err = uc.Restore(ctx, cur.Rev, domain.PartReviewTags, 1, bson.NewObjectID())
	if err != nil {
		t.Fatal(err)
	}
	if v, err = uc.Restore(ctx, v.Rev, domain.PartForumCategories, 1, bson.NewObjectID()); err != nil || len(v.Changes) != 2 {
		t.Fatalf("restore = %v, %v; want 2 changes", v, err)
	}
	if v.Draft.ReviewTags[0].Label.EN != domain.SeedCatalog().ReviewTags[0].Label.EN || !v.Draft.ForumCategories[0].Active {
		t.Errorf("restored draft doesn't match version 1")
	}
}

// Changes queued on version 1 still publish once version 2 is in force.
func TestChangeset_ReplaysOntoANewerVersion(t *testing.T) {
	uc, repo, _ := newChangeset()
	ctx := context.Background()
	v, _ := uc.Queue(ctx, 0, []domain.CatalogOp{renameEN}, bson.NewObjectID())
	newer := domain.SeedCatalog()
	newer.ReviewTags[0].Active = false
	_ = repo.Insert(ctx, &domain.CatalogPart{Name: domain.PartReviewTags, Version: 2, Content: mustPart(t, newer, domain.PartReviewTags)})
	published, _, err := uc.PublishAll(ctx, v.Rev, "rename", bson.NewObjectID())
	if err != nil {
		t.Fatal(err)
	}
	c := published.Catalog
	if c.Versions[domain.PartAnimals] != 2 || c.Versions[domain.PartReviewTags] != 2 || c.ReviewTags[0].Active || c.Categories[0].Label.EN != "Fowl" {
		t.Fatalf("published versions %v; want animals v2 alongside reviewTags v2", c.Versions)
	}
}

func codeOf(err error) string {
	code, _ := domain.CodeOf(err)
	return code
}
