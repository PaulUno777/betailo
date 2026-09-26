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
)

// NewRouter builds the application's top-level router.
//
// GET /me exists specifically to demonstrate RequireAuth protecting a
// route end-to-end; each further business module will add its own
// routes here as it's implemented (see the roadmap in the "Concepts &
// Architecture" document).
func NewRouter(logger *slog.Logger, deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /auth/register", handleRegister(deps.Register))
	mux.HandleFunc("POST /auth/login", handleLogin(deps.Login))
	mux.Handle("GET /me", RequireAuth(deps.Signer)(http.HandlerFunc(handleMe)))

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
