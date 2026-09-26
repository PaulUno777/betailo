package httpserver

import (
	"encoding/json"
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
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
}

type loginResponse struct {
	AccessToken string       `json:"accessToken"`
	User        userResponse `json:"user"`
}

func handleLogin(uc *auth.Login) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}

		out, err := uc.Execute(r.Context(), auth.LoginInput{Email: req.Email, Password: req.Password})
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, loginResponse{
			AccessToken: out.AccessToken,
			User:        toUserResponse(out.User),
		})
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
