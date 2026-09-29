package httpserver

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/listing"
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
	FarmVerified   bool    `json:"farmVerified"`
	FarmTrustScore float64 `json:"farmTrustScore"`

	ReviewAverage float64 `json:"reviewAverage"`
	ReviewCount   int     `json:"reviewCount"`
}

type marketplaceSearchResponse struct {
	Items []marketplaceListingResponse `json:"items"`
	Total int                          `json:"total"`
}

func handleSearchListings(uc *listing.SearchListings, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		criteria := ports.ListingSearchCriteria{
			Species: q.Get("species"), CycleType: q.Get("cycleType"), Region: q.Get("region"),
			MinPrice: parseFloatOr(q.Get("minPrice"), 0), MaxPrice: parseFloatOr(q.Get("maxPrice"), 0),
			VerifiedOnly: q.Get("verifiedOnly") == "true",
			Query:        q.Get("q"), Sort: q.Get("sort"),
			Page: parseIntOr(q.Get("page"), 1), PageSize: parseIntOr(q.Get("pageSize"), 0),
		}

		result, err := uc.Execute(r.Context(), criteria)
		if err != nil {
			writeError(w, err)
			return
		}

		items := make([]marketplaceListingResponse, 0, len(result.Items))
		for _, it := range result.Items {
			items = append(items, marketplaceListingResponse{
				Listing: toListingResponse(r.Context(), storage, it.Listing),
				Species: it.Species, CycleType: it.CycleType, CycleStatus: string(it.CycleStatus),
				StartsAt: it.StartsAt, EndsAt: it.EndsAt, Capacity: it.Capacity, Available: it.Available,
				FarmID: it.FarmID.Hex(), FarmName: it.FarmName, FarmLocation: it.FarmLocation,
				FarmVerified: it.FarmVerified, FarmTrustScore: it.FarmTrustScore,
				ReviewAverage: it.ReviewAverage, ReviewCount: it.ReviewCount,
			})
		}
		writeJSON(w, http.StatusOK, marketplaceSearchResponse{Items: items, Total: result.Total})
	}
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
