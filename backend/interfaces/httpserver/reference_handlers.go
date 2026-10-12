package httpserver

import (
	"net/http"
	"sort"

	"github.com/pauluno777/betailo/backend/domain"
)

type geoCountryResponse struct {
	Code    string   `json:"code"`
	Regions []string `json:"regions"`
}

type geoResponse struct {
	DefaultCountry string `json:"defaultCountry"`
	// Countries lists those with a fixed region list; a farm in any
	// other country types its region freely.
	Countries []geoCountryResponse `json:"countries"`
}

// handleGeoReference serves the region lists farms are validated
// against, so clients offer exactly the choices the API accepts.
func handleGeoReference(w http.ResponseWriter, _ *http.Request) {
	out := geoResponse{DefaultCountry: domain.DefaultCountry, Countries: []geoCountryResponse{}}
	for code, regions := range domain.KnownRegions {
		out.Countries = append(out.Countries, geoCountryResponse{Code: code, Regions: regions})
	}
	sort.Slice(out.Countries, func(i, j int) bool { return out.Countries[i].Code < out.Countries[j].Code })
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, http.StatusOK, out)
}
