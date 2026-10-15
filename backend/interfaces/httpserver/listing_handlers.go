package httpserver

import (
	"context"
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
	// Thumbs are the photos' small copies, index for index, for cards
	// and grids. A photo uploaded before small copies existed has none:
	// its thumb URL then answers 404 and the client falls back to the
	// photo.
	Thumbs []string `json:"thumbs,omitempty"`

	Breed             string `json:"breed,omitempty"`
	AgeWeeks          int    `json:"ageWeeks,omitempty"`
	DeliveryAvailable bool   `json:"deliveryAvailable"`
}

// toListingResponse resolves Photos (object-storage keys) to signed GET
// URLs on demand, per the domain comment on domain.Listing.Photos — a
// failed presign for one key just drops that photo rather than failing
// the whole response.
func toListingResponse(ctx context.Context, storage ports.ObjectStorage, l *domain.Listing) listingResponse {
	resp := listingResponse{
		ID: l.ID.Hex(), Title: l.Title, Description: l.Description,
		UnitPrice: l.UnitPrice, Unit: l.Unit, Status: string(l.Status),
		Breed: l.Breed, AgeWeeks: l.AgeWeeks, DeliveryAvailable: l.DeliveryAvailable,
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
		thumb, err := storage.PresignDownload(ctx, domain.ThumbKey(key))
		if err != nil {
			thumb = url
		}
		resp.Photos = append(resp.Photos, url)
		resp.Thumbs = append(resp.Thumbs, thumb)
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
		if !readJSON(w, r, &req) {
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
		claims, _ := ClaimsFromContext(r.Context())
		l, err := uc.Execute(r.Context(), id, claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toListingResponse(r.Context(), storage, l))
	}
}

// listingDetailResponse is the listing page in one payload. See
// listing.Detail.
type listingDetailResponse struct {
	Listing       listingResponse       `json:"listing"`
	Cycle         cycleDetailResponse   `json:"cycle"`
	Farm          farmProfileResponse   `json:"farm"`
	ReviewSummary reviewSummaryResponse `json:"reviewSummary"`
	TopReviews    []reviewResponse      `json:"topReviews"`
	Favorited     bool                  `json:"favorited"`
	IsOwner       bool                  `json:"isOwner"`
	CanReserve    bool                  `json:"canReserve"`
}

func handleGetListingDetail(uc *listing.GetDetail, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		claims, signedIn := ClaimsFromContext(r.Context())
		d, err := uc.Execute(r.Context(), id, claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		reviews := make([]reviewResponse, 0, len(d.TopReviews))
		for _, rv := range d.TopReviews {
			reviews = append(reviews, toReviewResponse(r.Context(), storage, rv))
		}
		writeJSON(w, http.StatusOK, listingDetailResponse{
			Listing:       toListingResponse(r.Context(), storage, d.Listing),
			Cycle:         cycleDetailResponse{cycleResponse: toCycleResponse(d.Cycle), Timeline: toTimelineResponse(d.Timeline)},
			Farm:          toFarmProfileResponse(d.Farm, signedIn),
			ReviewSummary: toReviewSummaryResponse(d.ReviewSummary),
			TopReviews:    reviews,
			Favorited:     d.Favorited,
			IsOwner:       d.IsOwner,
			CanReserve:    d.CanReserve,
		})
	}
}

// updateListingRequest edits a listing. cycleId is accepted (old
// clients send the create shape) but ignored: a listing never moves.
type updateListingRequest struct {
	CycleID           string  `json:"cycleId,omitempty"`
	Title             string  `json:"title"`
	Description       string  `json:"description,omitempty"`
	UnitPrice         float64 `json:"unitPrice"`
	Unit              string  `json:"unit,omitempty"`
	Breed             *string `json:"breed,omitempty"`
	AgeWeeks          *int    `json:"ageWeeks,omitempty"`
	DeliveryAvailable *bool   `json:"deliveryAvailable,omitempty"`
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
		var req updateListingRequest
		if !readJSON(w, r, &req) {
			return
		}

		l, err := uc.Execute(r.Context(), listing.UpdateListingInput{
			ListingID: id, OwnerID: claims.UserID, Title: req.Title,
			Description: req.Description, UnitPrice: req.UnitPrice, Unit: req.Unit,
			Breed: req.Breed, AgeWeeks: req.AgeWeeks, DeliveryAvailable: req.DeliveryAvailable,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toListingResponse(r.Context(), storage, l))
	}
}

// presignPhotoRequest declares the byte sizes of a photo and of its
// small copy, both already resized in the browser; the upload URLs only
// accept exactly those sizes.
type presignPhotoRequest struct {
	Size      int64 `json:"size"`
	ThumbSize int64 `json:"thumbSize"`
}

type presignPhotoResponse struct {
	Key            string `json:"key"`
	UploadURL      string `json:"uploadUrl"`
	ThumbUploadURL string `json:"thumbUploadUrl"`
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
		var req presignPhotoRequest
		if !readJSON(w, r, &req) {
			return
		}
		out, err := uc.Execute(r.Context(), listing.PresignListingPhotoInput{ListingID: id, OwnerID: claims.UserID, Size: req.Size, ThumbSize: req.ThumbSize})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, presignPhotoResponse{Key: out.Key, UploadURL: out.UploadURL, ThumbUploadURL: out.ThumbUploadURL})
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
		if !readJSON(w, r, &req) {
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
