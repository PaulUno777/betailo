package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestReference_ServesTheCatalogWithAContentETag(t *testing.T) {
	h := handleReference(nil) // nil source: the seed, every reference at version 1
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/reference", nil))
	out := decodeJSON[struct {
		Versions       map[string]int `json:"versions"`
		DefaultCountry string
		Categories     []struct {
			Key           string
			Subcategories []struct{ Key string }
		}
		ForumCategories []struct{ Key string }
	}](t, rec)
	if rec.Code != http.StatusOK || out.Versions["animals"] != 1 || out.Versions["places"] != 1 || out.DefaultCountry != "CM" || len(out.Categories) != 7 || len(out.Categories[0].Subcategories) != 12 || len(out.ForumCategories) == 0 {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
	}

	etag := rec.Header().Get("ETag")
	req := httptest.NewRequest(http.MethodGet, "/reference", nil)
	req.Header.Set("If-None-Match", etag)
	again := httptest.NewRecorder()
	h.ServeHTTP(again, req)
	if etag == "" || again.Code != http.StatusNotModified {
		t.Errorf("ETag %q, second status %d; want 304", etag, again.Code)
	}
}

// staticCatalog serves one catalog as the one in force.
type staticCatalog struct{ c *domain.Catalog }

func (s staticCatalog) Current(context.Context) (*domain.Catalog, error) { return s.c, nil }

// A catalog edited in place keeps its version numbers; its ETag must
// still change, or browsers keep their cached copy.
func TestReference_ETagFollowsTheContentNotTheVersion(t *testing.T) {
	edited := domain.SeedCatalog()
	edited.Categories[0].Label.FR = "Volailles"

	for _, path := range []string{"/reference", "/reference/animals"} {
		etag := func(c *domain.Catalog) string {
			h := handleReference(staticCatalog{c})
			if path != "/reference" {
				h = handleReferencePart(staticCatalog{c})
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.SetPathValue("name", "animals")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			return rec.Header().Get("ETag")
		}
		seed, changed := etag(domain.SeedCatalog()), etag(edited)
		if seed == "" || seed == changed {
			t.Errorf("%s: ETag %q before and %q after an edit at the same version; want them different", path, seed, changed)
		}
		if again := etag(domain.SeedCatalog()); again != seed {
			t.Errorf("%s: ETag %q then %q for the same content; want it stable", path, seed, again)
		}
	}
}
