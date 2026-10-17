package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"

	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/application/usecase/farmstats"
	"github.com/pauluno777/betailo/backend/domain"
)

// farmResponse is what a farm looks like over the wire to its owner
// (create, update, "my farms"), contact phone included.
type farmResponse struct {
	ID string `json:"id"`
	// Reference is the code staff and owners quote ("FRM-0212").
	Reference   string   `json:"reference,omitempty"`
	OwnerID     string   `json:"ownerId"`
	Name        string   `json:"name"`
	Location    string   `json:"location"`
	Country     string   `json:"country"`
	Region      string   `json:"region,omitempty"`
	Species     []string `json:"species,omitempty"`
	Description string   `json:"description,omitempty"`
	// Contacts are the farm's phone numbers, each one WhatsApp or call
	// only; withheld from anonymous viewers of the public profile.
	Contacts []farmContactJSON `json:"contacts"`
	// LogoURL, LogoThumbURL and BannerURL are short-lived links to the
	// farm's images; absent when the owner set none.
	LogoURL      string `json:"logoUrl,omitempty"`
	LogoThumbURL string `json:"logoThumbUrl,omitempty"`
	BannerURL    string `json:"bannerUrl,omitempty"`
	// Status is the farm's review state; only "approved" sells.
	Status string `json:"status"`
	// Review is the owner's (and staff's) view of that review. Never
	// part of the public profile.
	Review *farmReviewResponse `json:"review,omitempty"`
}

type farmReviewResponse struct {
	SubmittedAt *time.Time `json:"submittedAt,omitempty"`
	// Submissions is how many times the farm was sent for review.
	Submissions    int        `json:"submissions"`
	IDDocumentType string     `json:"idDocumentType,omitempty"`
	ReviewedAt     *time.Time `json:"reviewedAt,omitempty"`
	AutoApproved   bool       `json:"autoApproved,omitempty"`
	// RejectionCode is the kind of refusal, RejectionReason the
	// message staff wrote.
	RejectionCode   string `json:"rejectionCode,omitempty"`
	RejectionReason string `json:"rejectionReason,omitempty"`
	// ToRedo and Accepted split a refusal into what the owner sends
	// again and what staff found fine (among id_document, photos,
	// details); ReviewerRole is the role of who decided.
	ToRedo       []string `json:"toRedo,omitempty"`
	Accepted     []string `json:"accepted,omitempty"`
	ReviewerRole string   `json:"reviewerRole,omitempty"`
	// PhotoCount is how many farm photos the last submission held.
	PhotoCount int `json:"photoCount"`
	// CanSubmit: the farm is waiting on its owner to (re)submit proof.
	// CanWithdraw: it is in the queue and can be taken back out.
	CanSubmit   bool `json:"canSubmit"`
	CanWithdraw bool `json:"canWithdraw"`
}

type farmContactJSON struct {
	Phone    string `json:"phone"`
	WhatsApp bool   `json:"whatsapp"`
}

func toFarmResponse(ctx context.Context, storage ports.ObjectStorage, f *domain.Farm) farmResponse {
	contacts := make([]farmContactJSON, 0, len(f.Contacts))
	for _, c := range f.Contacts {
		contacts = append(contacts, farmContactJSON{Phone: c.Phone, WhatsApp: c.WhatsApp})
	}
	return farmResponse{
		ID: f.ID.Hex(), Reference: f.Reference, OwnerID: f.OwnerID.Hex(), Name: f.Name, Location: f.Location, Country: f.Country, Region: f.Region, Species: f.Species,
		Description: f.Description, Contacts: contacts, Status: string(f.Status),
		LogoURL: imageURL(ctx, storage, f.LogoKey), LogoThumbURL: avatarURL(ctx, storage, f.LogoKey), BannerURL: imageURL(ctx, storage, f.BannerKey),
	}
}

// imageURL is a short-lived link to the full-size image at key, or ""
// when there is none (or no storage to sign it with).
func imageURL(ctx context.Context, storage ports.ObjectStorage, key string) string {
	if key == "" || storage == nil {
		return ""
	}
	url, err := storage.PresignDownload(ctx, key)
	if err != nil {
		return ""
	}
	return url
}

// toOwnerFarmResponse adds the review to what the owner and staff see.
func toOwnerFarmResponse(ctx context.Context, storage ports.ObjectStorage, f *domain.Farm) farmResponse {
	out := toFarmResponse(ctx, storage, f)
	v := f.Verification
	out.Review = &farmReviewResponse{
		SubmittedAt: v.SubmittedAt, Submissions: v.Submissions, IDDocumentType: string(v.IDDocumentType),
		ReviewedAt: v.ReviewedAt, AutoApproved: v.AutoApproved,
		RejectionCode: string(v.RejectionCode), RejectionReason: v.RejectionReason,
		ReviewerRole: string(v.ReviewedByRole), PhotoCount: len(v.PhotoKeys),
		CanSubmit: f.Status.CanSubmit(), CanWithdraw: f.Status == domain.FarmPending,
	}
	if f.Status == domain.FarmRejected {
		for _, part := range v.RejectionCode.ToRedo() {
			out.Review.ToRedo = append(out.Review.ToRedo, string(part))
		}
		for _, part := range v.RejectionCode.Accepted() {
			out.Review.Accepted = append(out.Review.Accepted, string(part))
		}
	}
	return out
}

// farmProfileResponse is the public farm profile: the farm plus the
// trust signals computed by farm.GetProfile.
type farmProfileResponse struct {
	farmResponse
	TrustScore    float64   `json:"trustScore"`
	Verified      bool      `json:"verified"`
	SalesCount    int       `json:"salesCount"`
	MemberSince   time.Time `json:"memberSince"`
	ReviewAverage float64   `json:"reviewAverage"`
	ReviewCount   int       `json:"reviewCount"`
	OwnerName     string    `json:"ownerName,omitempty"`
	OwnerAvatar   string    `json:"ownerAvatarUrl,omitempty"`
}

// toFarmProfileResponse withholds the contact numbers from anonymous
// callers: a buyer has to sign in to call or message a farmer, which
// keeps phone numbers out of reach of scrapers.
func toFarmProfileResponse(ctx context.Context, storage ports.ObjectStorage, p *farmstats.Profile, signedIn bool) farmProfileResponse {
	base := toFarmResponse(ctx, storage, p.Farm)
	if !signedIn {
		base.Contacts = []farmContactJSON{}
	}
	return farmProfileResponse{
		farmResponse: base,
		TrustScore:   p.TrustScore, Verified: p.Verified, SalesCount: p.SalesCount, MemberSince: p.MemberSince,
		ReviewAverage: p.ReviewAverage, ReviewCount: p.ReviewCount,
		OwnerName: p.OwnerName, OwnerAvatar: avatarURL(ctx, storage, p.OwnerAvatarKey),
	}
}

type farmRequest struct {
	Name        string            `json:"name"`
	Location    string            `json:"location"`
	Country     string            `json:"country,omitempty"`
	Region      string            `json:"region,omitempty"`
	Species     []string          `json:"species,omitempty"`
	Description string            `json:"description,omitempty"`
	Contacts    []farmContactJSON `json:"contacts,omitempty"`
}

func (req farmRequest) contacts() []domain.FarmContact {
	out := make([]domain.FarmContact, 0, len(req.Contacts))
	for _, c := range req.Contacts {
		out = append(out, domain.FarmContact{Phone: c.Phone, WhatsApp: c.WhatsApp})
	}
	return out
}

func handleCreateFarm(uc *farm.CreateFarm, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}

		var req farmRequest
		if !readJSON(w, r, &req) {
			return
		}

		out, err := uc.Execute(r.Context(), farm.CreateFarmInput{
			OwnerID: claims.UserID, Name: req.Name, Location: req.Location, Country: req.Country, Region: req.Region, Species: req.Species,
			Description: req.Description, Contacts: req.contacts(),
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toOwnerFarmResponse(r.Context(), storage, out.Farm))
	}
}

func handleGetFarmProfile(uc *farm.GetProfile, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		p, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		_, signedIn := ClaimsFromContext(r.Context())
		writeJSON(w, http.StatusOK, toFarmProfileResponse(r.Context(), storage, p, signedIn))
	}
}

func handleListMyFarms(uc *farm.ListMyFarms, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		farms, err := uc.Execute(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]farmResponse, 0, len(farms))
		for _, f := range farms {
			out = append(out, toOwnerFarmResponse(r.Context(), storage, f))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// farmCardResponse is one farm on its owner's "Mes fermes" page.
type farmCardResponse struct {
	farmResponse
	// CoverURL is a short-lived link to the farm's first proof photo.
	CoverURL      string  `json:"coverUrl,omitempty"`
	ActiveCycles  int     `json:"activeCycles"`
	SalesCount    int     `json:"salesCount"`
	TrustScore    float64 `json:"trustScore"`
	ReviewAverage float64 `json:"reviewAverage"`
	ReviewCount   int     `json:"reviewCount"`
}

func handleListMyFarmCards(uc *farm.MyFarmCards, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		cards, err := uc.Execute(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]farmCardResponse, 0, len(cards))
		for _, c := range cards {
			row := farmCardResponse{
				farmResponse: toOwnerFarmResponse(r.Context(), storage, c.Farm), ActiveCycles: c.ActiveCycles, SalesCount: c.SalesCount,
				TrustScore: c.TrustScore, ReviewAverage: c.ReviewAverage, ReviewCount: c.ReviewCount,
			}
			if c.CoverKey != "" && storage != nil {
				if url, err := storage.PresignDownload(r.Context(), c.CoverKey); err == nil {
					row.CoverURL = url
				}
			}
			out = append(out, row)
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleUpdateFarm(uc *farm.UpdateFarm, storage ports.ObjectStorage) http.HandlerFunc {
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
		var req farmRequest
		if !readJSON(w, r, &req) {
			return
		}

		f, err := uc.Execute(r.Context(), farm.UpdateFarmInput{
			FarmID: id, OwnerID: claims.UserID, Name: req.Name, Location: req.Location, Country: req.Country, Region: req.Region, Species: req.Species,
			Description: req.Description, Contacts: req.contacts(),
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toOwnerFarmResponse(r.Context(), storage, f))
	}
}

func handleDeleteFarm(uc *farm.DeleteFarm) http.HandlerFunc {
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

type farmProofUploadResponse struct {
	Key       string `json:"key"`
	UploadURL string `json:"uploadUrl"`
}

func handlePresignFarmProof(uc *farm.PresignFarmProof) http.HandlerFunc {
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
		out, err := uc.Execute(r.Context(), id, claims.UserID, req.Size)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, farmProofUploadResponse{Key: out.Key, UploadURL: out.UploadURL})
	}
}

type submitFarmRequest struct {
	IDDocumentType    string   `json:"idDocumentType"`
	IDDocumentKey     string   `json:"idDocumentKey"`
	IDDocumentBackKey string   `json:"idDocumentBackKey"`
	PhotoKeys         []string `json:"photoKeys"`
	// Certified is the owner's declaration that the farm is theirs
	// and the photos recent.
	Certified bool `json:"certified"`
}

func handleWithdrawFarmSubmission(uc *farm.WithdrawFarmSubmission, storage ports.ObjectStorage) http.HandlerFunc {
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
		f, err := uc.Execute(r.Context(), id, claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toOwnerFarmResponse(r.Context(), storage, f))
	}
}

func handleSubmitFarmForReview(uc *farm.SubmitFarmForReview, storage ports.ObjectStorage) http.HandlerFunc {
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
		var req submitFarmRequest
		if !readJSON(w, r, &req) {
			return
		}
		f, err := uc.Execute(r.Context(), farm.SubmitFarmForReviewInput{
			FarmID: id, OwnerID: claims.UserID, IDDocumentType: domain.IDDocumentType(req.IDDocumentType),
			IDDocumentKey: req.IDDocumentKey, IDDocumentBackKey: req.IDDocumentBackKey, PhotoKeys: req.PhotoKeys, Certified: req.Certified,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toOwnerFarmResponse(r.Context(), storage, f))
	}
}

type setBrandingRequest struct {
	Key string `json:"key"`
}

func handlePresignFarmBranding(uc *farm.PresignBranding) http.HandlerFunc {
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
		out, err := uc.Execute(r.Context(), id, claims.UserID, req.Size, req.ThumbSize)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, presignPhotoResponse{Key: out.Key, UploadURL: out.UploadURL, ThumbUploadURL: out.ThumbUploadURL})
	}
}

// handleSetFarmBranding serves `PUT /farms/{id}/logo` and
// `PUT /farms/{id}/banner`: an empty key removes the image.
func handleSetFarmBranding(uc *farm.SetBranding, image farm.BrandingImage, storage ports.ObjectStorage) http.HandlerFunc {
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
		var req setBrandingRequest
		if !readJSON(w, r, &req) {
			return
		}
		f, err := uc.Execute(r.Context(), id, claims.UserID, image, req.Key)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toOwnerFarmResponse(r.Context(), storage, f))
	}
}
