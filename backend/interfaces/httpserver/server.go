// Package httpserver is the delivery layer shared by every business
// module: the router, cross-cutting middlewares (logging, auth) and
// generic JSON helpers.
//
// Business handlers will register their routes on the *http.ServeMux
// built here — this package never contains business logic itself, and
// depends on application/usecase types, never on infrastructure/mongodb
// or infrastructure/objectstorage directly (dependencies flow inward:
// delivery -> application -> domain, never the reverse).
package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/domain"
)

// chain applies middlewares to h in the order given — chain(h, A, B)
// means A wraps B wraps h, so A runs first. Only exists to keep the
// route table below readable; it has no behavior of its own.
func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// NewRouter builds the application's top-level router.
//
// Creating a farm, cycle or listing is restricted to farmers
// (RequireRole) — everything else that mutates one of those (update,
// delete, a cycle's status) is protected by RequireAuth alone and
// relies on the usecase's own ownership check instead: a buyer could
// never own a farm to begin with, so a second role gate there would be
// redundant. Reads (GET) need no authentication — farms, cycles and
// published listings are the public marketplace.
func NewRouter(logger *slog.Logger, deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	auth := RequireAuth(deps.Signer)
	farmerOnly := func(h http.Handler) http.Handler { return chain(h, auth, RequireRole(domain.RoleFarmer)) }

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /auth/register", handleRegister(deps.Register))
	mux.HandleFunc("POST /auth/login", handleLogin(deps.Login))
	mux.Handle("GET /me", chain(http.HandlerFunc(handleMe), auth))

	mux.Handle("POST /farms", farmerOnly(handleCreateFarm(deps.CreateFarm)))
	mux.HandleFunc("GET /farms/{id}", handleGetFarm(deps.GetFarm))
	mux.Handle("GET /farms/mine", chain(handleListMyFarms(deps.ListMyFarms), auth))
	mux.Handle("PUT /farms/{id}", chain(handleUpdateFarm(deps.UpdateFarm), auth))
	mux.Handle("DELETE /farms/{id}", chain(handleDeleteFarm(deps.DeleteFarm), auth))

	mux.Handle("POST /cycles", farmerOnly(handleCreateCycle(deps.CreateCycle)))
	mux.HandleFunc("GET /cycles/{id}", handleGetCycle(deps.GetCycle))
	mux.HandleFunc("GET /farms/{farmId}/cycles", handleListCyclesByFarm(deps.ListCyclesByFarm))
	mux.Handle("PATCH /cycles/{id}/status", chain(handleUpdateCycleStatus(deps.UpdateCycleStatus), auth))
	mux.Handle("DELETE /cycles/{id}", chain(handleDeleteCycle(deps.DeleteCycle), auth))

	mux.Handle("POST /listings", farmerOnly(handleCreateListing(deps.CreateListing)))
	mux.HandleFunc("GET /listings/{id}", handleGetListing(deps.GetListing))
	mux.HandleFunc("GET /listings", handleListPublishedListings(deps.ListPublishedListings))
	mux.Handle("PUT /listings/{id}", chain(handleUpdateListing(deps.UpdateListing), auth))
	mux.Handle("DELETE /listings/{id}", chain(handleUnpublishListing(deps.UnpublishListing), auth))

	return withLogging(logger, mux)
}

type healthResponse struct {
	Status string `json:"status"`
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// writeJSON is a small shared helper — every handler in every module
// should respond through it (or an equivalent) so error shapes stay
// consistent across the API.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// withLogging logs one line per request: method, path, status and
// duration. Kept intentionally simple for the MVP — swap for a richer
// middleware (request id, tracing) if the project grows past portfolio
// scope.
func withLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logger.Info("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
