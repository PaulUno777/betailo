package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// claimsContextKey is an unexported type so no other package can
// collide with this context key — the standard Go idiom for
// request-scoped values (see context.WithValue's own documentation).
type claimsContextKey struct{}

// AccountChecker says whether a signed-in account may still act. A
// token stays valid until it expires; this is what ends the sessions of
// an account suspended (or deleted) since. Nil skips the check.
type AccountChecker interface {
	CheckActive(ctx context.Context, userID bson.ObjectID) error
}

// RequireAuth verifies the request's Authorization: Bearer <token>
// header via signer and, on success, stores the token's claims in the
// request context for downstream handlers to read with ClaimsFromContext.
// Any missing, malformed, expired, or invalid token gets 401 — the
// handler behind this middleware never runs without valid claims
// available. With accounts set, a suspended account also gets 401, so
// the client's refresh fails and it signs out.
func RequireAuth(signer ports.TokenSigner, accounts AccountChecker) func(http.Handler) http.Handler {
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
			if accounts != nil {
				if err := accounts.CheckActive(r.Context(), claims.UserID); err != nil {
					if errors.Is(err, domain.ErrSuspended) || errors.Is(err, domain.ErrUnauthorized) {
						writeError(w, domain.Coded("account_suspended", errUnauthorized))
					} else {
						writeError(w, err)
					}
					return
				}
			}

			ctx := context.WithValue(r.Context(), claimsContextKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole must sit behind RequireAuth (it reads the claims that
// middleware stores) and rejects the request with 403 if the
// authenticated user's role doesn't match. Used to keep, for instance,
// farm/cycle/listing creation restricted to farmers — a buyer's valid

// OptionalAuth is RequireAuth for public routes that personalise their
// response when the caller is signed in (a favorited flag, a farmer's
// contact phone). No Authorization header means an anonymous request.
// A header that's present but invalid (typically an expired access
// token) is still a 401, so the client's refresh-and-retry flow kicks in
// instead of silently serving the anonymous view to a signed-in user.
func OptionalAuth(signer ports.TokenSigner, accounts AccountChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.TrimSpace(r.Header.Get("Authorization")) == "" {
				next.ServeHTTP(w, r)
				return
			}
			RequireAuth(signer, accounts)(next).ServeHTTP(w, r)
		})
	}
}

// token still gets 403, not the 401 an invalid token would.
func RequireRole(role domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok || claims.Role != role {
				writeError(w, errForbidden)
				return
			}
			next.ServeHTTP(w, r)
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
