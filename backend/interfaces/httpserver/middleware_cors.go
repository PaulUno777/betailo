package httpserver

import (
	"net/http"
	"os"
	"strings"
)

// withCORS lets a browser-hosted frontend (Angular dev server, or the
// deployed static build on its own origin) call this API cross-origin.
// The allowed origin comes from CORS_ALLOWED_ORIGIN so the same binary
// serves a different frontend origin in dev vs. production without a
// code change — the same environment-variable-only pattern
// connectStorage/newTokenSigner already use.
//
// CORS_ALLOWED_ORIGIN may list several origins, comma-separated (the
// dev server on localhost and the same machine's LAN address, so a
// phone on the network can try the app). A request from a listed origin
// gets that origin back; any other gets none, and the browser blocks it.
func withCORS(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range AllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGIN")) {
		allowed[o] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		if origin := r.Header.Get("Origin"); allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")
			// Safe: the origin echoed above is always one exact configured
			// value, never "*" — the one precondition browsers require
			// before they'll honor credentialed (cookie-bearing)
			// cross-origin requests. Needed for /auth/refresh and
			// /auth/logout's cookie; harmless for every other route, which
			// carries no cookie to begin with (see cookie_config.go's Path).
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AllowedOrigins parses CORS_ALLOWED_ORIGIN: comma-separated origins,
// trailing slashes dropped. Empty means the Angular dev server,
// http://localhost:4200. "*" is ignored: browsers refuse it on requests
// that carry the session cookie.
func AllowedOrigins(raw string) []string {
	var out []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimRight(strings.TrimSpace(o), "/"); o != "" && o != "*" {
			out = append(out, o)
		}
	}
	if len(out) == 0 {
		return []string{"http://localhost:4200"}
	}
	return out
}
