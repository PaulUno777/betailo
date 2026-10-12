package httpserver

import (
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/application/usecase/farmstats"
	"github.com/pauluno777/betailo/backend/domain"
)

// farmResponse is what a farm looks like over the wire to its owner
// (create, update, "my farms"), contact phone included.
type farmResponse struct {
	ID           string   `json:"id"`
	OwnerID      string   `json:"ownerId"`
	Name         string   `json:"name"`
	Location     string   `json:"location"`
	Country      string   `json:"country"`
	Region       string   `json:"region,omitempty"`
	Species      []string `json:"species,omitempty"`
	Description  string   `json:"description,omitempty"`
	ContactPhone string   `json:"contactPhone,omitempty"`
}

func toFarmResponse(f *domain.Farm) farmResponse {
	return farmResponse{
		ID: f.ID.Hex(), OwnerID: f.OwnerID.Hex(), Name: f.Name, Location: f.Location, Country: f.Country, Region: f.Region, Species: f.Species,
		Description: f.Description, ContactPhone: f.ContactPhone,
	}
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
		writeJSON(w, http.StatusCreated, toFarmResponse(out.Farm))
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
			out = append(out, toFarmResponse(f))
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
