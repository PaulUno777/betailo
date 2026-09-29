package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/listing"
	"github.com/pauluno777/betailo/backend/domain"
)

// listingResponse is what a listing looks like over the wire.
type listingResponse struct {
	ID          string   `json:"id"`
	CycleID     *string  `json:"cycleId,omitempty"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	UnitPrice   float64  `json:"unitPrice"`
	Unit        string   `json:"unit,omitempty"`
	Status      string   `json:"status"`
	Photos      []string `json:"photos,omitempty"`
}

// toListingResponse resolves Photos (object-storage keys) to signed GET
// URLs on demand, per the domain comment on domain.Listing.Photos — a
// failed presign for one key just drops that photo rather than failing
// the whole response.
func toListingResponse(ctx context.Context, storage ports.ObjectStorage, l *domain.Listing) listingResponse {
	resp := listingResponse{
		ID: l.ID.Hex(), Title: l.Title, Description: l.Description,
		UnitPrice: l.UnitPrice, Unit: l.Unit, Status: string(l.Status),
	}
	if l.CycleID != nil {
		hex := l.CycleID.Hex()
		resp.CycleID = &hex
	}
	for _, key := range l.Photos {
		url, err := storage.PresignDownload(ctx, key)
		if err != nil {
			slog.Error("listing: presign photo download", slog.String("key", key), slog.Any("err", err))
			continue
		}
		resp.Photos = append(resp.Photos, url)
	}
	return resp
}

type createListingRequest struct {
	CycleID     string  `json:"cycleId"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	UnitPrice   float64 `json:"unitPrice"`
	Unit        string  `json:"unit,omitempty"`
}

func handleCreateListing(uc *listing.CreateListing, storage ports.ObjectStorage) http.HandlerFunc {
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
			Description: req.Description, UnitPrice: req.UnitPrice, Unit: req.Unit,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toListingResponse(r.Context(), storage, l))
	}
}

func handleGetListing(uc *listing.GetListing, storage ports.ObjectStorage) http.HandlerFunc {
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
		writeJSON(w, http.StatusOK, toListingResponse(r.Context(), storage, l))
	}
}

func handleUpdateListing(uc *listing.UpdateListing, storage ports.ObjectStorage) http.HandlerFunc {
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
			Description: req.Description, UnitPrice: req.UnitPrice, Unit: req.Unit,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toListingResponse(r.Context(), storage, l))
	}
}

type presignPhotoResponse struct {
	Key       string `json:"key"`
	UploadURL string `json:"uploadUrl"`
}

func handlePresignListingPhoto(uc *listing.PresignListingPhoto) http.HandlerFunc {
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
		out, err := uc.Execute(r.Context(), listing.PresignListingPhotoInput{ListingID: id, OwnerID: claims.UserID})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, presignPhotoResponse{Key: out.Key, UploadURL: out.UploadURL})
	}
}

type setListingPhotosRequest struct {
	Keys []string `json:"keys"`
}

func handleSetListingPhotos(uc *listing.SetListingPhotos, storage ports.ObjectStorage) http.HandlerFunc {
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
		var req setListingPhotosRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}
		l, err := uc.Execute(r.Context(), listing.SetListingPhotosInput{ListingID: id, OwnerID: claims.UserID, Keys: req.Keys})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toListingResponse(r.Context(), storage, l))
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
