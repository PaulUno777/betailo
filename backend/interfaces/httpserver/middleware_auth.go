package httpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// claimsContextKey is an unexported type so no other package can
// collide with this context key — the standard Go idiom for
// request-scoped values (see context.WithValue's own documentation).
type claimsContextKey struct{}

// RequireAuth verifies the request's Authorization: Bearer <token>
// header via signer and, on success, stores the token's claims in the
// request context for downstream handlers to read with ClaimsFromContext.
// Any missing, malformed, expired, or invalid token gets 401 — the
// handler behind this middleware never runs without valid claims
// available.
func RequireAuth(signer ports.TokenSigner) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				writeError(w, errUnauthorized)
				return
			}

			claims, err := signer.ParseAccessToken(r.Context(), token)
			if err != nil {
				writeError(w, errUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext reads the claims RequireAuth stored. The second
// return value is false only if called on a request that never went
// through RequireAuth — a handler wiring bug, not something a client
// can trigger.
func ClaimsFromContext(ctx context.Context) (ports.TokenClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(ports.TokenClaims)
	return claims, ok
}

func bearerToken(r *http.Request) (string, bool) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}
	return token, true
}
