package httpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/catalog"
	"github.com/pauluno777/betailo/backend/domain"
)

// referenceResponse is the whole reference data, every reference put
// together, with the version in force of each.
type referenceResponse struct {
	*domain.Catalog
	DefaultCountry string   `json:"defaultCountry"`
	CycleTypes     []string `json:"cycleTypes"`
}

func toReferenceResponse(c *domain.Catalog) referenceResponse {
	return referenceResponse{Catalog: c, DefaultCountry: domain.DefaultCountry, CycleTypes: domain.CycleTypes}
}

// writeCached writes body with an ETag made from its own bytes, and
// answers 304 when the client already holds them. A fingerprint of the
// content, not the version number: a reference changed without a new
// version (the shipped seed, edited while nothing is in production)
// still reaches every browser instead of their cached copy.
func writeCached(w http.ResponseWriter, r *http.Request, body any) {
	raw, err := json.Marshal(body)
	if err != nil {
		writeError(w, err)
		return
	}
	sum := sha256.Sum256(raw)
	etag := `"` + hex.EncodeToString(sum[:8]) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=60")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

// handleReference serves every reference together (GET /reference):
// what clients loaded before the references were split. The versions
// served make the ETag.
func handleReference(src ports.CatalogSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cat, err := catalog.Of(r.Context(), src)
		if err != nil {
			writeError(w, err)
			return
		}
		writeCached(w, r, toReferenceResponse(cat))
	}
}

// handleReferencePart serves one reference (GET /reference/{name}):
// its content, its name and version, plus what it needs read alongside
// (the cycle types for the animals, the default country for places),
// so a client loads, and re-downloads after a publish, only what it
// uses. Its content makes the ETag.
func handleReferencePart(src ports.CatalogSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !domain.IsPart(name) {
			writeError(w, domain.Coded("catalog.part_unknown", domain.ErrNotFound))
			return
		}
		cat, err := catalog.Of(r.Context(), src)
		if err != nil {
			writeError(w, err)
			return
		}
		version := cat.Versions[name]
		content, err := cat.PartContent(name)
		if err != nil {
			writeError(w, err)
			return
		}
		var body map[string]any
		_ = json.Unmarshal(content, &body)
		body["name"], body["version"] = name, version
		switch name {
		case domain.PartAnimals:
			body["cycleTypes"] = domain.CycleTypes
		case domain.PartPlaces:
			body["defaultCountry"] = domain.DefaultCountry
		}
		writeCached(w, r, body)
	}
}
