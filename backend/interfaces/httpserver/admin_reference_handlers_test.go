package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	catalogusecase "github.com/pauluno777/betailo/backend/application/usecase/catalog"
	"github.com/pauluno777/betailo/backend/domain"
)

// fakeCatalogRepo keeps each reference's versions in memory, oldest first.
type fakeCatalogRepo struct {
	ports.CatalogRepository
	parts map[string][]*domain.CatalogPart
}

func (f *fakeCatalogRepo) Latest(_ context.Context, name string) (*domain.CatalogPart, error) {
	if list := f.parts[name]; len(list) > 0 {
		return list[len(list)-1], nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeCatalogRepo) Get(_ context.Context, name string, version int) (*domain.CatalogPart, error) {
	for _, p := range f.parts[name] {
		if p.Version == version {
			return p, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeCatalogRepo) List(_ context.Context, name string) ([]*domain.CatalogPart, error) {
	var out []*domain.CatalogPart
	for _, n := range domain.CatalogParts {
		if name == "" || n == name {
			for i := len(f.parts[n]) - 1; i >= 0; i-- {
				out = append(out, f.parts[n][i])
			}
		}
	}
	return out, nil
}

func (f *fakeCatalogRepo) Insert(_ context.Context, p *domain.CatalogPart) error {
	if f.parts == nil {
		f.parts = map[string][]*domain.CatalogPart{}
	}
	f.parts[p.Name] = append(f.parts[p.Name], p)
	return nil
}

type fakeChangesetRepo struct{ cs domain.CatalogChangeset }

func (f *fakeChangesetRepo) Get(context.Context) (*domain.CatalogChangeset, error) {
	cs := f.cs
	cs.Changes = append([]domain.CatalogChange(nil), f.cs.Changes...)
	return &cs, nil
}

func (f *fakeChangesetRepo) Save(_ context.Context, cs *domain.CatalogChangeset, rev int) error {
	if f.cs.Rev != rev {
		return domain.ErrConflict
	}
	f.cs = *cs
	f.cs.Rev = rev + 1
	return nil
}

type fakeUsage map[string]int

func (f fakeUsage) Usage(context.Context) (map[string]int, error) { return f, nil }

// Super admins queue changes together, then publish them as one
// version; nobody else reaches any of it.
func TestAdminReference_HTTP_ChangesetFlow(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	if err := catalogusecase.EnsureSeed(context.Background(), tf.catalogVersions, nil, time.Now()); err != nil {
		t.Fatal(err)
	}

	moderator, _ := staffToken(t, router, tf, domain.StaffModerator)
	for _, path := range []string{"/admin/reference/changeset", "/admin/reference/schema/animals", "/admin/reference/usage"} {
		if rec := doJSON(t, router, http.MethodGet, path, nil, moderator); rec.Code != http.StatusForbidden {
			t.Errorf("moderator GET %s: status = %d, want 403", path, rec.Code)
		}
	}

	admin, _ := staffToken(t, router, tf, domain.StaffSuperAdmin)
	view := decodeJSON[changesetResponse](t, doJSON(t, router, http.MethodGet, "/admin/reference/changeset", nil, admin))
	if view.Rev != 0 || len(view.Changes) != 0 || view.Ready {
		t.Fatalf("empty changeset = %+v", view)
	}

	queue := map[string]any{"rev": 0, "ops": []map[string]any{
		{"kind": "update", "ref": map[string]any{"entity": "category", "key": "volaille"}, "value": map[string]any{"label": map[string]any{"en": "Fowl"}}},
		{"kind": "add", "ref": map[string]any{"entity": "term", "list": "reviewTags"}, "value": map[string]any{"key": "calm", "label": map[string]any{"fr": "Calme", "en": "Calm"}, "active": true}},
	}}
	rec := doJSON(t, router, http.MethodPost, "/admin/reference/changeset/ops", queue, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("queue: %d %s", rec.Code, rec.Body.String())
	}
	view = decodeJSON[changesetResponse](t, rec)
	if view.Rev != 1 || len(view.Changes) != 2 || !view.Ready || view.Changes[1].Op.Ref.Key != "calm" || !strings.Contains(string(view.Changes[0].Before), "Poultry") {
		t.Fatalf("after queueing = %+v", view)
	}

	// A refused operation names its place in the batch.
	bad := map[string]any{"rev": 1, "ops": []map[string]any{{"kind": "update", "ref": map[string]any{"entity": "category", "key": "ovins"}, "value": map[string]any{"key": "other"}}}}
	rec = doJSON(t, router, http.MethodPost, "/admin/reference/changeset/ops", bad, admin)
	if body := decodeJSON[errorResponse](t, rec); body.Code != "catalog.op_key_immutable" || body.Params["index"] != float64(0) {
		t.Errorf("renaming a key: %d %+v", rec.Code, body)
	}

	usage := decodeJSON[map[string]int](t, doJSON(t, router, http.MethodGet, "/admin/reference/usage", nil, admin))
	if usage["species:poussins"] != 4 {
		t.Errorf("usage = %v", usage)
	}

	rec = doJSON(t, router, http.MethodPost, "/admin/reference/changeset/publish", map[string]any{"rev": 1, "note": "Fowl and calm"}, admin)
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish: %d %s", rec.Code, rec.Body.String())
	}
	var published struct {
		Reference referenceResponse        `json:"reference"`
		Published []catalogVersionResponse `json:"published"`
	}
	raw, _ := json.Marshal(decodeJSON[map[string]any](t, rec))
	_ = json.Unmarshal(raw, &published)
	if len(published.Published) != 2 || published.Reference.Versions["animals"] != 2 || published.Reference.Versions["places"] != 1 || published.Reference.Categories[0].Label.EN != "Fowl" {
		t.Errorf("published = %+v", published.Published)
	}
	if last := tf.audit.actions[len(tf.audit.actions)-1]; last.Action != domain.ActionCatalogPublish || last.TargetLabel != "animals v2, reviewTags v2" || !strings.HasPrefix(last.Detail, "2 · ") {
		t.Errorf("journal = %+v", last)
	}

	// Restoring the review highlights' version 1 becomes changes to review.
	rec = doJSON(t, router, http.MethodPost, "/admin/reference/changeset/restore", map[string]any{"rev": 2, "name": "reviewTags", "version": 1}, admin)
	view = decodeJSON[changesetResponse](t, rec)
	var draft struct {
		ReviewTags []domain.CatalogTerm `json:"reviewTags"`
	}
	raw, _ = json.Marshal(view.Draft)
	_ = json.Unmarshal(raw, &draft)
	if len(view.Changes) != 1 || draft.ReviewTags[len(draft.ReviewTags)-1].Active {
		t.Errorf("restore: %d changes, the added tag still active; want only the review highlights restored", len(view.Changes))
	}
}

// Each reference is served on its own, with its own version as ETag.
func TestReference_HTTP_OneReferenceAtATime(t *testing.T) {
	router := newTestRouter()
	rec := doJSON(t, router, http.MethodGet, "/reference/places", nil, "")
	body := decodeJSON[map[string]any](t, rec)
	if rec.Code != http.StatusOK || body["name"] != "places" || body["defaultCountry"] != "CM" || body["countries"] == nil || body["categories"] != nil {
		t.Fatalf("places = %d %v", rec.Code, body)
	}
	req, _ := http.NewRequest(http.MethodGet, "/reference/places", nil)
	req.Header.Set("If-None-Match", rec.Header().Get("ETag"))
	again := httptest.NewRecorder()
	router.ServeHTTP(again, req)
	if again.Code != http.StatusNotModified {
		t.Errorf("same version again: %d, want 304", again.Code)
	}
	if rec := doJSON(t, router, http.MethodGet, "/reference/dragons", nil, ""); rec.Code != http.StatusNotFound {
		t.Errorf("unknown reference: %d, want 404", rec.Code)
	}
}
