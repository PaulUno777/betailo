package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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

// cacheable answers 304 when the client already holds etag.
func cacheable(w http.ResponseWriter, r *http.Request, etag string) bool {
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=60")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}

// handleReference serves every reference together (GET /reference):
// what clients loaded before the references were split. The versions
// in force make the ETag.
func handleReference(src ports.CatalogSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cat, err := catalog.Of(r.Context(), src)
		if err != nil {
			writeError(w, err)
			return
		}
		tags := make([]string, 0, len(domain.CatalogParts))
		for _, name := range domain.CatalogParts {
			tags = append(tags, name+strconv.Itoa(cat.Versions[name]))
		}
		if cacheable(w, r, `"catalog-`+strings.Join(tags, "-")+`"`) {
			return
		}
		writeJSON(w, http.StatusOK, toReferenceResponse(cat))
	}
}

// handleReferencePart serves one reference (GET /reference/{name}):
// its content, its name and version, plus what it needs read alongside
// (the cycle types for the animals, the default country for places),
// so a client loads, and re-downloads after a publish, only what it
// uses. The name and version make the ETag.
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
		if cacheable(w, r, `"`+name+`-`+strconv.Itoa(version)+`"`) {
			return
		}
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
		writeJSON(w, http.StatusOK, body)
	}
}
