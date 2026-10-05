package httpserver

import (
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/cycle"
	"github.com/pauluno777/betailo/backend/application/usecase/listing"
)

type quickListingRequest struct {
	Species           string     `json:"species"`
	Breed             string     `json:"breed,omitempty"`
	AgeWeeks          int        `json:"ageWeeks,omitempty"`
	Quantity          int        `json:"quantity"`
	UnitPrice         float64    `json:"unitPrice"`
	Title             string     `json:"title"`
	Description       string     `json:"description,omitempty"`
	AvailableFrom     *time.Time `json:"availableFrom,omitempty"`
	DeliveryAvailable bool       `json:"deliveryAvailable"`
}

type quickListingResponse struct {
	Cycle   cycleResponse   `json:"cycle"`
	Listing listingResponse `json:"listing"`
}

func handleQuickCreateListing(uc *listing.QuickCreate, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		farmID, ok := pathObjectID(w, r, "farmId")
		if !ok {
			return
		}
		var req quickListingRequest
		if !readJSON(w, r, &req) {
			return
		}
		out, err := uc.Execute(r.Context(), listing.QuickCreateInput{
			FarmID: farmID, OwnerID: claims.UserID, Species: req.Species, Breed: req.Breed, AgeWeeks: req.AgeWeeks,
			Quantity: req.Quantity, UnitPrice: req.UnitPrice, Title: req.Title, Description: req.Description,
			AvailableFrom: req.AvailableFrom, DeliveryAvailable: req.DeliveryAvailable,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, quickListingResponse{Cycle: toCycleResponse(out.Cycle), Listing: toListingResponse(r.Context(), storage, out.Listing)})
	}
}

func handlePublishListing(uc *listing.Publish, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		l, err := uc.Execute(r.Context(), id, claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toListingResponse(r.Context(), storage, l))
	}
}

type farmListingResponse struct {
	Listing            listingResponse `json:"listing"`
	Cycle              cycleResponse   `json:"cycle"`
	ActiveReservations int             `json:"activeReservations"`
}

func handleListFarmListings(uc *listing.ListFarmListings, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		farmID, ok := pathObjectID(w, r, "farmId")
		if !ok {
			return
		}
		rows, err := uc.Execute(r.Context(), farmID, claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]farmListingResponse, 0, len(rows))
		for _, row := range rows {
			out = append(out, farmListingResponse{
				Listing: toListingResponse(r.Context(), storage, row.Listing), Cycle: toCycleResponse(row.Cycle),
				ActiveReservations: row.ActiveReservations,
			})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

type priceHintResponse struct {
	Count    int     `json:"count"`
	Low      float64 `json:"low"`
	Median   float64 `json:"median"`
	High     float64 `json:"high"`
	Regional bool    `json:"regional"`
}

// handlePriceHint answers 204 when there aren't enough comparable
// listings for a meaningful range.
func handlePriceHint(uc *listing.PriceHint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		h, err := uc.Execute(r.Context(), q.Get("species"), q.Get("region"))
		if err != nil {
			writeError(w, err)
			return
		}
		if h == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, http.StatusOK, priceHintResponse{Count: h.Count, Low: h.Low, Median: h.Median, High: h.High, Regional: h.Regional})
	}
}

type updateCycleRequest struct {
	StartsAt          time.Time  `json:"startsAt"`
	EndsAt            *time.Time `json:"endsAt,omitempty"`
	Capacity          int        `json:"capacity"`
	MinPerReservation int        `json:"minPerReservation"`
	MaxPerReservation int        `json:"maxPerReservation"`
}

func handleUpdateCycle(uc *cycle.UpdateCycle) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		var req updateCycleRequest
		if !readJSON(w, r, &req) {
			return
		}
		c, err := uc.Execute(r.Context(), cycle.UpdateCycleInput{
			CycleID: id, OwnerID: claims.UserID, StartsAt: req.StartsAt, EndsAt: req.EndsAt, Capacity: req.Capacity,
			MinPerReservation: req.MinPerReservation, MaxPerReservation: req.MaxPerReservation,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toCycleResponse(c))
	}
}
