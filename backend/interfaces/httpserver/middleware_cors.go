package httpserver

import (
	"net/http"
	"os"
)

// withCORS lets a browser-hosted frontend (Angular dev server, or the
// deployed static build on its own origin) call this API cross-origin.
// The allowed origin comes from CORS_ALLOWED_ORIGIN so the same binary
// serves a different frontend origin in dev vs. production without a
// code change — the same environment-variable-only pattern
// connectStorage/newTokenSigner already use.
func withCORS(next http.Handler) http.Handler {
	origin := os.Getenv("CORS_ALLOWED_ORIGIN")
	if origin == "" {
		origin = "http://localhost:4200"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
