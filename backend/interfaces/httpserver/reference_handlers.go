package httpserver

import (
	"net/http"
	"strconv"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/catalog"
	"github.com/pauluno777/betailo/backend/domain"
)

// handleReference serves the reference data in force (GET /reference):
// species, countries and regions, forum categories, review tags, report
// reasons and the farm review checklist, with their labels, so clients
// offer exactly the choices the API accepts. The version number is the
// ETag: a client that has it gets 304.
func handleReference(src ports.CatalogSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cat, err := catalog.Of(r.Context(), src)
		if err != nil {
			writeError(w, err)
			return
		}
		etag := `"catalog-` + strconv.Itoa(cat.Version) + `"`
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "public, max-age=60")
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		writeJSON(w, http.StatusOK, referenceResponse{Catalog: cat, DefaultCountry: domain.DefaultCountry, CycleTypes: domain.CycleTypes})
	}
}

type referenceResponse struct {
	*domain.Catalog
	DefaultCountry string   `json:"defaultCountry"`
	CycleTypes     []string `json:"cycleTypes"`
}

type geoCountryResponse struct {
	Code    string   `json:"code"`
	Regions []string `json:"regions"`
}

type geoResponse struct {
	DefaultCountry string               `json:"defaultCountry"`
	Countries      []geoCountryResponse `json:"countries"`
	CallingCodes   map[string]string    `json:"callingCodes"`
}

// handleGeoReference is the former shape of the country and region
// lists (GET /reference/geo), kept for clients deployed before
// GET /reference. It reads the same catalog.
func handleGeoReference(src ports.CatalogSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cat, err := catalog.Of(r.Context(), src)
		if err != nil {
			writeError(w, err)
			return
		}
		out := geoResponse{DefaultCountry: domain.DefaultCountry, Countries: []geoCountryResponse{}, CallingCodes: map[string]string{}}
		for _, k := range cat.Countries {
			if !k.Active {
				continue
			}
			out.CallingCodes[k.Code] = k.CallingCode
			if len(k.Regions) == 0 {
				continue
			}
			c := geoCountryResponse{Code: k.Code}
			for _, reg := range k.Regions {
				if reg.Active {
					c.Regions = append(c.Regions, reg.Label)
				}
			}
			out.Countries = append(out.Countries, c)
		}
		w.Header().Set("Cache-Control", "public, max-age=60")
		writeJSON(w, http.StatusOK, out)
	}
}
