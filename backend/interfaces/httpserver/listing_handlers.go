package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/pauluno777/betailo/backend/application/usecase/listing"
	"github.com/pauluno777/betailo/backend/domain"
)

// listingResponse is what a listing looks like over the wire.
type listingResponse struct {
	ID          string  `json:"id"`
	CycleID     *string `json:"cycleId,omitempty"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	UnitPrice   float64 `json:"unitPrice"`
	Status      string  `json:"status"`
}

func toListingResponse(l *domain.Listing) listingResponse {
	resp := listingResponse{
		ID: l.ID.Hex(), Title: l.Title, Description: l.Description,
		UnitPrice: l.UnitPrice, Status: string(l.Status),
	}
	if l.CycleID != nil {
		hex := l.CycleID.Hex()
		resp.CycleID = &hex
	}
	return resp
}

type createListingRequest struct {
	CycleID     string  `json:"cycleId"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	UnitPrice   float64 `json:"unitPrice"`
}

func handleCreateListing(uc *listing.CreateListing) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req createListingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}
		cycleID, err := parseObjectIDField(req.CycleID)
		if err != nil {
			writeError(w, errBadRequestBody)
			return
		}

		l, err := uc.Execute(r.Context(), listing.CreateListingInput{
			CycleID: cycleID, OwnerID: claims.UserID, Title: req.Title,
			Description: req.Description, UnitPrice: req.UnitPrice,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toListingResponse(l))
	}
}

func handleGetListing(uc *listing.GetListing) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		l, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toListingResponse(l))
	}
}

func handleListPublishedListings(uc *listing.ListPublishedListings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listings, err := uc.Execute(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]listingResponse, 0, len(listings))
		for _, l := range listings {
			out = append(out, toListingResponse(l))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleUpdateListing(uc *listing.UpdateListing) http.HandlerFunc {
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
		var req createListingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}

		l, err := uc.Execute(r.Context(), listing.UpdateListingInput{
			ListingID: id, OwnerID: claims.UserID, Title: req.Title,
			Description: req.Description, UnitPrice: req.UnitPrice,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toListingResponse(l))
	}
}

func handleUnpublishListing(uc *listing.UnpublishListing) http.HandlerFunc {
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
		if err := uc.Execute(r.Context(), id, claims.UserID); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
