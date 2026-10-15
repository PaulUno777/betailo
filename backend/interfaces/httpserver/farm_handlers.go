package httpserver

import (
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
	Reference    string   `json:"reference,omitempty"`
	OwnerID      string   `json:"ownerId"`
	Name         string   `json:"name"`
	Location     string   `json:"location"`
	Country      string   `json:"country"`
	Region       string   `json:"region,omitempty"`
	Species      []string `json:"species,omitempty"`
	Description  string   `json:"description,omitempty"`
	ContactPhone string   `json:"contactPhone,omitempty"`
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

func toFarmResponse(f *domain.Farm) farmResponse {
	return farmResponse{
		ID: f.ID.Hex(), Reference: f.Reference, OwnerID: f.OwnerID.Hex(), Name: f.Name, Location: f.Location, Country: f.Country, Region: f.Region, Species: f.Species,
		Description: f.Description, ContactPhone: f.ContactPhone, Status: string(f.Status),
	}
}

// toOwnerFarmResponse adds the review to what the owner and staff see.
func toOwnerFarmResponse(f *domain.Farm) farmResponse {
	out := toFarmResponse(f)
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
}

// toFarmProfileResponse withholds the contact phone from anonymous
// callers: a buyer has to sign in to call or message a farmer, which
// keeps phone numbers out of reach of scrapers.
func toFarmProfileResponse(p *farmstats.Profile, signedIn bool) farmProfileResponse {
	base := toFarmResponse(p.Farm)
	if !signedIn {
		base.ContactPhone = ""
	}
	return farmProfileResponse{
		farmResponse: base,
		TrustScore:   p.TrustScore, Verified: p.Verified, SalesCount: p.SalesCount, MemberSince: p.MemberSince,
		ReviewAverage: p.ReviewAverage, ReviewCount: p.ReviewCount,
	}
}

type farmRequest struct {
	Name         string   `json:"name"`
	Location     string   `json:"location"`
	Country      string   `json:"country,omitempty"`
	Region       string   `json:"region,omitempty"`
	Species      []string `json:"species,omitempty"`
	Description  string   `json:"description,omitempty"`
	ContactPhone string   `json:"contactPhone,omitempty"`
}

func handleCreateFarm(uc *farm.CreateFarm) http.HandlerFunc {
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
			Description: req.Description, ContactPhone: req.ContactPhone,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toOwnerFarmResponse(out.Farm))
	}
}

func handleGetFarmProfile(uc *farm.GetProfile) http.HandlerFunc {
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
		writeJSON(w, http.StatusOK, toFarmProfileResponse(p, signedIn))
	}
}

func handleListMyFarms(uc *farm.ListMyFarms) http.HandlerFunc {
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
			out = append(out, toOwnerFarmResponse(f))
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
				farmResponse: toOwnerFarmResponse(c.Farm), ActiveCycles: c.ActiveCycles, SalesCount: c.SalesCount,
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

func handleUpdateFarm(uc *farm.UpdateFarm) http.HandlerFunc {
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
			Description: req.Description, ContactPhone: req.ContactPhone,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toFarmResponse(f))
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

func handleWithdrawFarmSubmission(uc *farm.WithdrawFarmSubmission) http.HandlerFunc {
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
		writeJSON(w, http.StatusOK, toOwnerFarmResponse(f))
	}
}

func handleSubmitFarmForReview(uc *farm.SubmitFarmForReview) http.HandlerFunc {
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
		writeJSON(w, http.StatusOK, toOwnerFarmResponse(f))
	}
}
