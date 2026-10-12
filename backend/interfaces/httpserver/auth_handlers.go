package httpserver

import (
	"net/http"

	"github.com/pauluno777/betailo/backend/application/usecase/auth"
	"github.com/pauluno777/betailo/backend/domain"
)

// registerRequest/loginRequest are the wire shape for their endpoints —
// kept separate from application/usecase/auth's Input types so a
// change to the JSON contract never forces a change to the usecase
// layer, and vice versa.
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// userResponse is what a user looks like over the wire — deliberately
// not domain.User itself, so PasswordHash can never leak by omitting a
// json tag by mistake on the domain type.
type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{ID: u.ID.Hex(), Email: u.Email, Role: string(u.Role)}
}

func handleRegister(uc *auth.Register) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRequest
		if !readJSON(w, r, &req) {
			return
		}

		out, err := uc.Execute(r.Context(), auth.RegisterInput{
			Email:    req.Email,
			Password: req.Password,
			Role:     domain.Role(req.Role),
		})
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, toUserResponse(out.User))
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

type loginResponse struct {
	AccessToken string       `json:"accessToken"`
	User        userResponse `json:"user"`
}

func handleLogin(uc *auth.Login, cookies CookieConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if !readJSON(w, r, &req) {
			return
		}

		out, err := uc.Execute(r.Context(), auth.LoginInput{Email: req.Email, Password: req.Password, Remember: req.Remember})
		if err != nil {
			writeError(w, err)
			return
		}

		setRefreshCookie(w, out.RefreshToken, out.RefreshExpiresAt, req.Remember, cookies)
		writeJSON(w, http.StatusOK, loginResponse{
			AccessToken: out.AccessToken,
			User:        toUserResponse(out.User),
		})
	}
}

// refreshResponse deliberately carries only the new access token — a
// refresh call doesn't need to re-send the full user, and the rotated
// refresh token itself never appears in a JSON body, only in the
// Set-Cookie header (see setRefreshCookie).
type refreshResponse struct {
	AccessToken string `json:"accessToken"`
}

func handleRefresh(uc *auth.RefreshToken, cookies CookieConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := readRefreshCookie(r)
		if raw == "" {
			writeError(w, errUnauthorized)
			return
		}

		out, err := uc.Execute(r.Context(), auth.RefreshTokenInput{RawToken: raw})
		if err != nil {
			// Whatever went wrong (expired, reused, unknown, suspended
			// account), the presented cookie is no longer any good —
			// clear it so the frontend's next silent-refresh attempt
			// fails fast instead of retrying the same dead cookie.
			clearRefreshCookie(w, cookies)
			writeError(w, err)
			return
		}

		setRefreshCookie(w, out.RefreshToken, out.RefreshExpiresAt, out.Persistent, cookies)
		writeJSON(w, http.StatusOK, refreshResponse{AccessToken: out.AccessToken})
	}
}

func handleLogout(uc *auth.Logout, cookies CookieConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := readRefreshCookie(r)
		// Best-effort: Logout.Execute is a no-op for an empty or
		// unknown token (see its doc comment), and the cookie is
		// cleared either way, so there's nothing a caller needs to
		// react to here.
		_ = uc.Execute(r.Context(), auth.LogoutInput{RawToken: raw})
		clearRefreshCookie(w, cookies)
		w.WriteHeader(http.StatusNoContent)
	}
}

// meResponse echoes back what RequireAuth verified — this route exists
// specifically to demonstrate the JWT middleware protecting a route
// end-to-end (see the roadmap's Phase 1 requirement), not because a
// buyer or farmer needs a dedicated "who am I" endpoint yet.
type meResponse struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		// Only reachable if this handler is wired without RequireAuth in
		// front of it — a routing bug, not a client-triggerable state.
		writeError(w, errUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, meResponse{
		UserID: claims.UserID.Hex(),
		Email:  claims.Email,
		Role:   string(claims.Role),
	})
}
