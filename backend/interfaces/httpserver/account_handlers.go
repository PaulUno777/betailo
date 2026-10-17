package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/account"
	"github.com/pauluno777/betailo/backend/application/usecase/auth"
	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/domain"
)

// profileResponse is the signed-in user's own account.
type profileResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	StaffRole   string    `json:"staffRole,omitempty"`
	DisplayName string    `json:"displayName,omitempty"`
	AvatarURL   string    `json:"avatarUrl,omitempty"`
	Phone       string    `json:"phone,omitempty"`
	City        string    `json:"city,omitempty"`
	TrustScore  float64   `json:"trustScore"`
	MemberSince time.Time `json:"memberSince"`
}

func toProfileResponse(ctx context.Context, storage ports.ObjectStorage, u *domain.User) profileResponse {
	return profileResponse{
		ID: u.ID.Hex(), Email: u.Email, StaffRole: string(u.StaffRole), DisplayName: u.DisplayName,
		AvatarURL: avatarURL(ctx, storage, u.AvatarKey), Phone: u.Phone, City: u.City,
		TrustScore: u.TrustScore, MemberSince: u.CreatedAt,
	}
}

// avatarURL is where a browser downloads a profile photo's small copy,
// or "" when the account has none.
func avatarURL(ctx context.Context, storage ports.ObjectStorage, key string) string {
	if key == "" || storage == nil {
		return ""
	}
	url, err := storage.PresignDownload(ctx, domain.ThumbKey(key))
	if err != nil {
		return ""
	}
	return url
}

// meResponse is `GET /me`: the profile plus what the account can do,
// decided here so the client only draws it. Farms is every farm the
// account owns with its review status; CanAddFarm says whether it may
// open another (its limit, and staff accounts never sell).
type meResponse struct {
	profileResponse
	Farms     []farmResponse `json:"farms"`
	FarmLimit int            `json:"farmLimit"`
	// FarmContactLimit is how many contact numbers each farm may list.
	FarmContactLimit int  `json:"farmContactLimit"`
	CanAddFarm       bool `json:"canAddFarm"`
	// FarmLimitRaisedAt is set when staff gave this account its own limit.
	FarmLimitRaisedAt *time.Time `json:"farmLimitRaisedAt,omitempty"`
	// Permissions is what the account's staff role lets it do in the
	// back office (empty for everyone else); IsRoot marks the
	// platform's root account.
	Permissions []string `json:"permissions"`
	IsRoot      bool     `json:"isRoot,omitempty"`
}

func handleGetProfile(uc *account.GetProfile, access *farm.SellerAccess, storage ports.ObjectStorage) http.HandlerFunc {
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
		seller, err := access.Execute(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		farms := make([]farmResponse, 0, len(seller.Farms))
		for _, f := range seller.Farms {
			farms = append(farms, toOwnerFarmResponse(r.Context(), storage, f))
		}
		permissions := []string{}
		for _, p := range u.StaffRole.Permissions() {
			permissions = append(permissions, string(p))
		}
		out := meResponse{
			profileResponse: toProfileResponse(r.Context(), storage, u), Farms: farms, FarmLimit: seller.FarmLimit, FarmContactLimit: seller.ContactLimit, CanAddFarm: seller.CanAdd,
			Permissions: permissions, IsRoot: u.Root,
		}
		if u.FarmLimit > 0 && u.FarmLimitNote != nil {
			out.FarmLimitRaisedAt = &u.FarmLimitNote.SetAt
		}
		writeJSON(w, http.StatusOK, out)
	}
}

type updateProfileRequest struct {
	DisplayName string `json:"displayName"`
	Phone       string `json:"phone"`
	City        string `json:"city"`
}

func handleUpdateProfile(uc *account.UpdateProfile, storage ports.ObjectStorage) http.HandlerFunc {
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
		writeJSON(w, http.StatusOK, toProfileResponse(r.Context(), storage, u))
	}
}

func handlePresignAvatar(uc *account.PresignAvatar) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req presignPhotoRequest
		if !readJSON(w, r, &req) {
			return
		}
		out, err := uc.Execute(r.Context(), claims.UserID, req.Size, req.ThumbSize)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, presignPhotoResponse{Key: out.Key, UploadURL: out.UploadURL, ThumbUploadURL: out.ThumbUploadURL})
	}
}

type setAvatarRequest struct {
	Key string `json:"key"`
}

func handleSetAvatar(uc *account.SetAvatar, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req setAvatarRequest
		if !readJSON(w, r, &req) {
			return
		}
		if req.Key == "" {
			writeError(w, domain.Coded("photo.key_required", fmt.Errorf("account: no photo key: %w", domain.ErrInvalid)))
			return
		}
		u, err := uc.Execute(r.Context(), claims.UserID, req.Key)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toProfileResponse(r.Context(), storage, u))
	}
}

func handleRemoveAvatar(uc *account.SetAvatar, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		u, err := uc.Execute(r.Context(), claims.UserID, "")
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toProfileResponse(r.Context(), storage, u))
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
		out, err := login.SessionFor(r.Context(), u, req.Remember, claims.MFA)
		if err != nil {
			writeError(w, err)
			return
		}
		setRefreshCookie(w, out.RefreshToken, out.RefreshExpiresAt, req.Remember, cookies)
		writeJSON(w, http.StatusOK, loginResponse{AccessToken: out.AccessToken, User: toUserResponse(out.User)})
	}
}
