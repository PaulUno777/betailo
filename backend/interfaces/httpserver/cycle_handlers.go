package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/application/usecase/cycle"
	"github.com/pauluno777/betailo/backend/domain"
)

// cycleResponse is what a cycle looks like over the wire.
type cycleResponse struct {
	ID                string     `json:"id"`
	FarmID            string     `json:"farmId"`
	Species           string     `json:"species"`
	Type              string     `json:"type"`
	Status            string     `json:"status"`
	Capacity          int        `json:"capacity"`
	AvailableQuantity int        `json:"availableQuantity"`
	StartsAt          time.Time  `json:"startsAt"`
	EndsAt            *time.Time `json:"endsAt,omitempty"`
}

func toCycleResponse(c *domain.Cycle) cycleResponse {
	return cycleResponse{
		ID: c.ID.Hex(), FarmID: c.FarmID.Hex(), Species: c.Species, Type: c.Type,
		Status: string(c.Status), Capacity: c.Capacity, AvailableQuantity: c.AvailableQuantity,
		StartsAt: c.StartsAt, EndsAt: c.EndsAt,
	}
}

type createCycleRequest struct {
	FarmID   string     `json:"farmId"`
	Species  string     `json:"species"`
	Type     string     `json:"type"`
	Capacity int        `json:"capacity"`
	StartsAt time.Time  `json:"startsAt"`
	EndsAt   *time.Time `json:"endsAt,omitempty"`
}

func handleCreateCycle(uc *cycle.CreateCycle) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req createCycleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}
		farmID, err := parseObjectIDField(req.FarmID)
		if err != nil {
			writeError(w, errBadRequestBody)
			return
		}

		c, err := uc.Execute(r.Context(), cycle.CreateCycleInput{
			FarmID: farmID, OwnerID: claims.UserID, Species: req.Species, Type: req.Type,
			Capacity: req.Capacity, StartsAt: req.StartsAt, EndsAt: req.EndsAt,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toCycleResponse(c))
	}
}

func handleGetCycle(uc *cycle.GetCycle) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		c, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toCycleResponse(c))
	}
}

func handleListCyclesByFarm(uc *cycle.ListCyclesByFarm) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		farmID, ok := pathObjectID(w, r, "farmId")
		if !ok {
			return
		}
		cycles, err := uc.Execute(r.Context(), farmID)
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]cycleResponse, 0, len(cycles))
		for _, c := range cycles {
			out = append(out, toCycleResponse(c))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

type updateCycleStatusRequest struct {
	Status string `json:"status"`
}

func handleUpdateCycleStatus(uc *cycle.UpdateCycleStatus) http.HandlerFunc {
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
		var req updateCycleStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}

		c, err := uc.Execute(r.Context(), cycle.UpdateCycleStatusInput{
			CycleID: id, OwnerID: claims.UserID, Status: domain.CycleStatus(req.Status),
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toCycleResponse(c))
	}
}

func handleDeleteCycle(uc *cycle.DeleteCycle) http.HandlerFunc {
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
