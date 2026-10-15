package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReference_ServesTheCatalogWithItsVersionsAsETag(t *testing.T) {
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
	if rec.Code != http.StatusOK || out.Versions["animals"] != 1 || out.Versions["places"] != 1 || out.DefaultCountry != "CM" || len(out.Categories) != 7 || len(out.Categories[0].Subcategories) != 9 || len(out.ForumCategories) == 0 {
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
