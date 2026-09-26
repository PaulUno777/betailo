package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/domain"
)

// farmResponse is what a farm looks like over the wire.
type farmResponse struct {
	ID       string   `json:"id"`
	OwnerID  string   `json:"ownerId"`
	Name     string   `json:"name"`
	Location string   `json:"location"`
	Species  []string `json:"species,omitempty"`
}

func toFarmResponse(f *domain.Farm) farmResponse {
	return farmResponse{
		ID: f.ID.Hex(), OwnerID: f.OwnerID.Hex(), Name: f.Name, Location: f.Location, Species: f.Species,
	}
}

type farmRequest struct {
	Name     string   `json:"name"`
	Location string   `json:"location"`
	Species  []string `json:"species,omitempty"`
}

func handleCreateFarm(uc *farm.CreateFarm) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}

		var req farmRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}

		out, err := uc.Execute(r.Context(), farm.CreateFarmInput{
			OwnerID: claims.UserID, Name: req.Name, Location: req.Location, Species: req.Species,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toFarmResponse(out.Farm))
	}
}

func handleGetFarm(uc *farm.GetFarm) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		f, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toFarmResponse(f))
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}

		f, err := uc.Execute(r.Context(), farm.UpdateFarmInput{
			FarmID: id, OwnerID: claims.UserID, Name: req.Name, Location: req.Location, Species: req.Species,
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
