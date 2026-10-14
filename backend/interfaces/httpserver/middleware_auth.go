package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/admin"
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

// StaffRoleSource gives an account's staff role as it is stored right
// now. A token keeps the role it was signed with until it expires; the
// back office asks the source instead, so a role granted or revoked
// since applies at once. Nil falls back to the token's claim.
type StaffRoleSource interface {
	StaffRole(ctx context.Context, userID bson.ObjectID) (domain.StaffRole, error)
}

// actorContextKey holds the staff member RequirePermission let through.
type actorContextKey struct{}

// RequirePermission must sit behind RequireAuth (it reads the claims
// that middleware stores). It lets the request through only if the
// account's staff role holds permission p, and otherwise answers 403
// naming the permission and the roles that hold it, so the client can
// say who to ask. A valid token without the permission is a 403, never
// the 401 an invalid token gets.
func RequirePermission(roles StaffRoleSource, p domain.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeError(w, errForbidden)
				return
			}
			role := claims.StaffRole
			if roles != nil {
				live, err := roles.StaffRole(r.Context(), claims.UserID)
				if err != nil {
					writeError(w, err)
					return
				}
				role = live
			}
			if !role.IsStaff() {
				writeError(w, errForbidden)
				return
			}
			if !role.Can(p) {
				allowed := domain.RolesWith(p)
				names := make([]string, 0, len(allowed))
				for _, a := range allowed {
					names = append(names, string(a))
				}
				writeError(w, domain.CodedWith("admin.permission_denied", map[string]any{"permission": string(p), "roles": names}, errForbidden))
				return
			}
			ctx := context.WithValue(r.Context(), actorContextKey{}, admin.Actor{ID: claims.UserID, Role: role})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// actorFromContext is the staff member behind a back-office request.
// False only on a route that isn't behind RequirePermission: a wiring
// bug, not something a client can trigger.
func actorFromContext(ctx context.Context) (admin.Actor, bool) {
	actor, ok := ctx.Value(actorContextKey{}).(admin.Actor)
	return actor, ok
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
