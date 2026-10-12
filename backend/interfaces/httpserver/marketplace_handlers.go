package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/listing"
	"github.com/pauluno777/betailo/backend/domain"
)

// marketplaceListingResponse is one card's worth of data for the public
// marketplace — a listing joined with exactly the cycle/farm/review
// fields ListingCard needs, computed server-side by
// ports.ListingSearchRepository rather than requiring the frontend to
// stitch together separate calls per card.
type marketplaceListingResponse struct {
	Listing listingResponse `json:"listing"`

	Species     string     `json:"species"`
	CycleType   string     `json:"cycleType"`
	CycleStatus string     `json:"cycleStatus"`
	StartsAt    time.Time  `json:"startsAt"`
	EndsAt      *time.Time `json:"endsAt,omitempty"`
	Capacity    int        `json:"capacity"`
	Available   int        `json:"availableQuantity"`

	FarmID         string  `json:"farmId"`
	FarmName       string  `json:"farmName"`
	FarmLocation   string  `json:"farmLocation"`
	FarmCountry    string  `json:"farmCountry,omitempty"`
	FarmRegion     string  `json:"farmRegion,omitempty"`
	FarmVerified   bool    `json:"farmVerified"`
	FarmTrustScore float64 `json:"farmTrustScore"`

	ReviewAverage float64 `json:"reviewAverage"`
	ReviewCount   int     `json:"reviewCount"`

	Favorited bool `json:"favorited"`
}

type marketplaceSearchResponse struct {
	Items  []marketplaceListingResponse `json:"items"`
	Total  int                          `json:"total"`
	Facets searchFacetsResponse         `json:"facets"`
}

type facetCountResponse struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type regionFacetResponse struct {
	Country string `json:"country"`
	Region  string `json:"region"`
	Count   int    `json:"count"`
}

// searchFacetsResponse is what each filter would lead to: counts per
// species, cycle type, status and region, and the price range.
type searchFacetsResponse struct {
	Species    []facetCountResponse  `json:"species"`
	CycleTypes []facetCountResponse  `json:"cycleTypes"`
	Statuses   []facetCountResponse  `json:"statuses"`
	Regions    []regionFacetResponse `json:"regions"`
	PriceMin   float64               `json:"priceMin"`
	PriceMax   float64               `json:"priceMax"`
}

func toFacetsResponse(f ports.SearchFacets) searchFacetsResponse {
	counts := func(in []ports.FacetCount) []facetCountResponse {
		out := make([]facetCountResponse, 0, len(in))
		for _, c := range in {
			out = append(out, facetCountResponse{Value: c.Value, Count: c.Count})
		}
		return out
	}
	regions := make([]regionFacetResponse, 0, len(f.Regions))
	for _, r := range f.Regions {
		regions = append(regions, regionFacetResponse{Country: r.Country, Region: r.Region, Count: r.Count})
	}
	return searchFacetsResponse{
		Species: counts(f.Species), CycleTypes: counts(f.CycleTypes), Statuses: counts(f.Statuses),
		Regions: regions, PriceMin: f.PriceMin, PriceMax: f.PriceMax,
	}
}

func handleSearchListings(uc *listing.SearchListings, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		criteria := ports.ListingSearchCriteria{
			Species: listParam(q, "species"), CycleTypes: listParam(q, "cycleType"),
			Country: strings.ToUpper(strings.TrimSpace(q.Get("country"))), Region: strings.TrimSpace(q.Get("region")),
			MinPrice: parseFloatOr(q.Get("minPrice"), 0), MaxPrice: parseFloatOr(q.Get("maxPrice"), 0),
			VerifiedOnly: q.Get("verifiedOnly") == "true",
			Query:        q.Get("q"), Sort: q.Get("sort"),
			Page: parseIntOr(q.Get("page"), 1), PageSize: parseIntOr(q.Get("pageSize"), 0),
		}
		for _, st := range listParam(q, "status") {
			criteria.Statuses = append(criteria.Statuses, domain.CycleStatus(st))
		}
		var ok bool
		if criteria.AvailableFrom, criteria.AvailableTo, ok = availabilityWindow(q.Get("available")); !ok {
			writeError(w, domain.Coded("search.available_invalid", fmt.Errorf("httpserver: available must be YYYY-MM or YYYY-MM-DD: %w", domain.ErrInvalid)))
			return
		}

		claims, _ := ClaimsFromContext(r.Context())
		result, err := uc.Execute(r.Context(), criteria, claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, marketplaceSearchResponse{
			Items: toMarketplaceItems(r.Context(), storage, result.Items), Total: result.Total, Facets: toFacetsResponse(result.Facets),
		})
	}
}

// listParam reads a multi-value filter given either repeated
// (?species=a&species=b) or comma-separated (?species=a,b).
func listParam(q url.Values, key string) []string {
	var out []string
	for _, raw := range q[key] {
		for _, v := range strings.Split(raw, ",") {
			if v = strings.TrimSpace(v); v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

// availabilityWindow turns "available" into the window a lot must be
// open in: a month ("2026-10", the whole month) or a day ("2026-10-15").
// Empty means no window; ok is false for anything else.
func availabilityWindow(raw string) (from, to time.Time, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, time.Time{}, true
	}
	if m, err := time.Parse("2006-01", raw); err == nil {
		return m, m.AddDate(0, 1, 0).Add(-time.Nanosecond), true
	}
	if d, err := time.Parse("2006-01-02", raw); err == nil {
		return d, d.Add(24*time.Hour - time.Nanosecond), true
	}
	return time.Time{}, time.Time{}, false
}

func parseFloatOr(s string, def float64) float64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

func parseIntOr(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func toMarketplaceItems(ctx context.Context, storage ports.ObjectStorage, in []ports.ListingSearchItem) []marketplaceListingResponse {
	items := make([]marketplaceListingResponse, 0, len(in))
	for _, it := range in {
		items = append(items, marketplaceListingResponse{
			Listing: toListingResponse(ctx, storage, it.Listing),
			Species: it.Species, CycleType: it.CycleType, CycleStatus: string(it.CycleStatus),
			StartsAt: it.StartsAt, EndsAt: it.EndsAt, Capacity: it.Capacity, Available: it.Available,
			FarmID: it.FarmID.Hex(), FarmName: it.FarmName, FarmLocation: it.FarmLocation,
			FarmCountry: it.FarmCountry, FarmRegion: it.FarmRegion,
			FarmVerified: it.FarmVerified, FarmTrustScore: it.FarmTrustScore,
			ReviewAverage: it.ReviewAverage, ReviewCount: it.ReviewCount,
			Favorited: it.Favorited,
		})
	}
	return items
}
