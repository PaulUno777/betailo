package httpserver

import (
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/application/usecase/account"
	"github.com/pauluno777/betailo/backend/application/usecase/auth"
	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/domain"
)

// profileResponse is the signed-in user's own account.
type profileResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	DisplayName string    `json:"displayName,omitempty"`
	Phone       string    `json:"phone,omitempty"`
	City        string    `json:"city,omitempty"`
	Verified    bool      `json:"verified"`
	TrustScore  float64   `json:"trustScore"`
	MemberSince time.Time `json:"memberSince"`
}

func toProfileResponse(u *domain.User) profileResponse {
	return profileResponse{
		ID: u.ID.Hex(), Email: u.Email, Role: string(u.Role), DisplayName: u.DisplayName, Phone: u.Phone, City: u.City,
		Verified: u.IsVerified, TrustScore: u.TrustScore, MemberSince: u.CreatedAt,
	}
}

func handleGetProfile(uc *account.GetProfile) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		u, err := uc.Execute(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toProfileResponse(u))
	}
}

type updateProfileRequest struct {
	DisplayName string `json:"displayName"`
	Phone       string `json:"phone"`
	City        string `json:"city"`
}

func handleUpdateProfile(uc *account.UpdateProfile) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req updateProfileRequest
		if !readJSON(w, r, &req) {
			return
		}
		u, err := uc.Execute(r.Context(), account.UpdateProfileInput{
			UserID: claims.UserID, DisplayName: req.DisplayName, Phone: req.Phone, City: req.City,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toProfileResponse(u))
	}
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	Remember        bool   `json:"remember"`
}

// handleChangePassword changes the password (ending every session),
// then signs this device straight back in with the new one, so the user
// who just changed it isn't the one person bounced to the login page.
func handleChangePassword(uc *auth.ChangePassword, login *auth.Login, cookies CookieConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req changePasswordRequest
		if !readJSON(w, r, &req) {
			return
		}
		u, err := uc.Execute(r.Context(), auth.ChangePasswordInput{UserID: claims.UserID, Current: req.CurrentPassword, New: req.NewPassword})
		if err != nil {
			writeError(w, err)
			return
		}
		out, err := login.Execute(r.Context(), auth.LoginInput{Email: u.Email, Password: req.NewPassword, Remember: req.Remember})
		if err != nil {
			writeError(w, err)
			return
		}
		setRefreshCookie(w, out.RefreshToken, out.RefreshExpiresAt, req.Remember, cookies)
		writeJSON(w, http.StatusOK, loginResponse{AccessToken: out.AccessToken, User: toUserResponse(out.User)})
	}
}

type becomeSellerRequest struct {
	Farm farmRequest `json:"farm"`
}

func handleBecomeSeller(uc *farm.BecomeSeller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req becomeSellerRequest
		if !readJSON(w, r, &req) {
			return
		}
		f, err := uc.Execute(r.Context(), farm.BecomeSellerInput{
			UserID: claims.UserID, FarmName: req.Farm.Name, Location: req.Farm.Location, Species: req.Farm.Species,
			Description: req.Farm.Description, ContactPhone: req.Farm.ContactPhone,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toFarmResponse(f))
	}
}
