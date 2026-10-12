package httpserver

import (
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/application/usecase/cycle"
	"github.com/pauluno777/betailo/backend/domain"
)

// cycleResponse is what a cycle looks like over the wire.
type cycleResponse struct {
	ID                string     `json:"id"`
	Reference         string     `json:"reference"`
	FarmID            string     `json:"farmId"`
	Species           string     `json:"species"`
	Type              string     `json:"type"`
	Status            string     `json:"status"`
	Capacity          int        `json:"capacity"`
	AvailableQuantity int        `json:"availableQuantity"`
	StartsAt          time.Time  `json:"startsAt"`
	EndsAt            *time.Time `json:"endsAt,omitempty"`
	MinPerReservation int        `json:"minPerReservation,omitempty"`
	MaxPerReservation int        `json:"maxPerReservation,omitempty"`
}

func toCycleResponse(c *domain.Cycle) cycleResponse {
	return cycleResponse{
		ID: c.ID.Hex(), Reference: c.Reference, FarmID: c.FarmID.Hex(), Species: c.Species, Type: c.Type,
		Status: string(c.Status), Capacity: c.Capacity, AvailableQuantity: c.AvailableQuantity,
		StartsAt: c.StartsAt, EndsAt: c.EndsAt,
		MinPerReservation: c.MinPerReservation, MaxPerReservation: c.MaxPerReservation,
	}
}

type timelineStepResponse struct {
	Kind   string    `json:"kind"`
	Date   time.Time `json:"date"`
	Title  string    `json:"title,omitempty"`
	Detail string    `json:"detail,omitempty"`
	State  string    `json:"state"`
}

func toTimelineResponse(steps []domain.TimelineStep) []timelineStepResponse {
	out := make([]timelineStepResponse, 0, len(steps))
	for _, s := range steps {
		out = append(out, timelineStepResponse{Kind: string(s.Kind), Date: s.Date, Title: s.Title, Detail: s.Detail, State: string(s.State)})
	}
	return out
}

// cycleDetailResponse is a single cycle with its server-built timeline.
type cycleDetailResponse struct {
	cycleResponse
	Timeline []timelineStepResponse `json:"timeline"`
}

type healthEventRequest struct {
	Title  string    `json:"title"`
	Detail string    `json:"detail,omitempty"`
	Date   time.Time `json:"date"`
}

func handleAddHealthEvent(uc *cycle.AddHealthEvent) http.HandlerFunc {
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
		var req healthEventRequest
		if !readJSON(w, r, &req) {
			return
		}
		c, err := uc.Execute(r.Context(), cycle.AddHealthEventInput{
			CycleID: id, OwnerID: claims.UserID, Title: req.Title, Detail: req.Detail, Date: req.Date,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toCycleResponse(c))
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
		if !readJSON(w, r, &req) {
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
		v, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, cycleDetailResponse{cycleResponse: toCycleResponse(v.Cycle), Timeline: toTimelineResponse(v.Timeline)})
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
		if !readJSON(w, r, &req) {
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
