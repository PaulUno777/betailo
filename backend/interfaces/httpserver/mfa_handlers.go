package httpserver

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/application/usecase/admin"
	"github.com/pauluno777/betailo/backend/application/usecase/auth"
	"github.com/pauluno777/betailo/backend/domain"
)

type passkeyResponse struct {
	// ID is the credential's identifier, base64url: what the client
	// sends back to remove it.
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type mfaStatusResponse struct {
	TOTP          bool              `json:"totp"`
	Passkeys      []passkeyResponse `json:"passkeys"`
	RecoveryCodes int               `json:"recoveryCodes"`
	Enabled       bool              `json:"enabled"`
}

func toMFAStatusResponse(s *auth.MFAStatus) mfaStatusResponse {
	out := mfaStatusResponse{TOTP: s.TOTP, Passkeys: make([]passkeyResponse, 0, len(s.Passkeys)), RecoveryCodes: s.RecoveryCodes}
	for _, p := range s.Passkeys {
		out.Passkeys = append(out.Passkeys, passkeyResponse{ID: base64.RawURLEncoding.EncodeToString(p.ID), Name: p.Name, CreatedAt: p.CreatedAt})
	}
	out.Enabled = out.TOTP || len(out.Passkeys) > 0
	return out
}

func handleMFAStatus(uc *auth.SecondFactor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		s, err := uc.Status(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		out := toMFAStatusResponse(s)
		writeJSON(w, http.StatusOK, struct {
			mfaStatusResponse
			// SessionVerified: this session passed a second factor.
			SessionVerified bool `json:"sessionVerified"`
			// PasskeysAvailable: this deployment can register passkeys.
			PasskeysAvailable bool `json:"passkeysAvailable"`
		}{out, claims.MFA, uc.PasskeysAvailable()})
	}
}

type totpSetupResponse struct {
	Secret string `json:"secret"`
	// URL is the otpauth:// address an authenticator app reads; QR is
	// that address as a QR code, a PNG data URL ready for an <img>.
	URL string `json:"url"`
	QR  string `json:"qr"`
}

func handleTOTPSetup(uc *auth.SecondFactor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		s, err := uc.SetupTOTP(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, totpSetupResponse{Secret: s.Secret, URL: s.URL, QR: "data:image/png;base64," + base64.StdEncoding.EncodeToString(s.QR)})
	}
}

// enrolledResponse follows the addition of a factor. RecoveryCodes is
// shown once; AccessToken replaces the client's, for a session that now
// counts as verified.
type enrolledResponse struct {
	RecoveryCodes []string          `json:"recoveryCodes,omitempty"`
	AccessToken   string            `json:"accessToken"`
	Status        mfaStatusResponse `json:"status"`
}

func toEnrolledResponse(e *auth.Enrolled) enrolledResponse {
	return enrolledResponse{RecoveryCodes: e.RecoveryCodes, AccessToken: e.AccessToken, Status: toMFAStatusResponse(e.Status)}
}

type codeRequest struct {
	Code string `json:"code"`
}

func handleTOTPConfirm(uc *auth.SecondFactor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req codeRequest
		if !readJSON(w, r, &req) {
			return
		}
		e, err := uc.ConfirmTOTP(r.Context(), claims.UserID, req.Code, readRefreshCookie(r))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toEnrolledResponse(e))
	}
}

type passwordRequest struct {
	Password string `json:"password"`
}

func handleTOTPDisable(uc *auth.SecondFactor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req passwordRequest
		if !readJSON(w, r, &req) {
			return
		}
		s, err := uc.DisableTOTP(r.Context(), claims.UserID, req.Password)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toMFAStatusResponse(s))
	}
}

func handleRecoveryCodes(uc *auth.SecondFactor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req passwordRequest
		if !readJSON(w, r, &req) {
			return
		}
		codes, err := uc.RegenerateRecoveryCodes(r.Context(), claims.UserID, req.Password)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string][]string{"recoveryCodes": codes})
	}
}

// passkeyOptionsResponse carries the browser's credentials-API options
// as the passkey service produced them, untouched.
type passkeyOptionsResponse struct {
	Options   json.RawMessage `json:"options"`
	FlowToken string          `json:"flowToken,omitempty"`
}

func handlePasskeyBegin(uc *auth.SecondFactor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		setup, err := uc.BeginPasskey(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, passkeyOptionsResponse{Options: setup.Options, FlowToken: setup.FlowToken})
	}
}

type passkeyFinishRequest struct {
	FlowToken string `json:"flowToken"`
	Name      string `json:"name"`
	// Credential is the browser's answer, as its JSON form.
	Credential json.RawMessage `json:"credential"`
}

func handlePasskeyFinish(uc *auth.SecondFactor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req passkeyFinishRequest
		if !readJSON(w, r, &req) {
			return
		}
		e, err := uc.FinishPasskey(r.Context(), claims.UserID, req.FlowToken, req.Name, req.Credential, readRefreshCookie(r))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toEnrolledResponse(e))
	}
}

type passkeyRemoveRequest struct {
	ID       string `json:"id"`
	Password string `json:"password"`
}

func handlePasskeyRemove(uc *auth.SecondFactor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req passkeyRemoveRequest
		if !readJSON(w, r, &req) {
			return
		}
		id, err := base64.RawURLEncoding.DecodeString(req.ID)
		if err != nil {
			writeError(w, domain.ErrInvalid)
			return
		}
		s, err := uc.RemovePasskey(r.Context(), claims.UserID, id, req.Password)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toMFAStatusResponse(s))
	}
}

// --- the second step of a sign-in ---

type mfaVerifyRequest struct {
	MFAToken     string          `json:"mfaToken"`
	Code         string          `json:"code,omitempty"`
	RecoveryCode string          `json:"recoveryCode,omitempty"`
	Credential   json.RawMessage `json:"credential,omitempty"`
}

func writeSession(w http.ResponseWriter, out *auth.LoginOutput, persistent bool, cookies CookieConfig) {
	setRefreshCookie(w, out.RefreshToken, out.RefreshExpiresAt, persistent, cookies)
	writeJSON(w, http.StatusOK, loginResponse{AccessToken: out.AccessToken, User: toUserResponse(out.User)})
}

func handleMFAVerify(uc *auth.SecondFactor, cookies CookieConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mfaVerifyRequest
		if !readJSON(w, r, &req) {
			return
		}
		out, err := uc.Verify(r.Context(), auth.VerifyInput{MFAToken: req.MFAToken, Code: req.Code, RecoveryCode: req.RecoveryCode})
		if err != nil {
			writeError(w, err)
			return
		}
		writeSession(w, out, out.Persistent, cookies)
	}
}

func handleMFAPasskeyOptions(uc *auth.SecondFactor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mfaVerifyRequest
		if !readJSON(w, r, &req) {
			return
		}
		options, err := uc.BeginPasskeyLogin(r.Context(), req.MFAToken)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, passkeyOptionsResponse{Options: options})
	}
}

func handleMFAPasskeyVerify(uc *auth.SecondFactor, cookies CookieConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mfaVerifyRequest
		if !readJSON(w, r, &req) {
			return
		}
		out, err := uc.FinishPasskeyLogin(r.Context(), req.MFAToken, req.Credential)
		if err != nil {
			writeError(w, err)
			return
		}
		writeSession(w, out, out.Persistent, cookies)
	}
}

// --- staff invitations ---

type invitationResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	InviterName string    `json:"inviterName,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
	// Link is only present right after sending or renewing: it is never
	// stored, so it can't be read back later.
	Link   string `json:"link,omitempty"`
	Mailed *bool  `json:"mailed,omitempty"`
}

func toInvitationResponse(t *domain.AuthToken) invitationResponse {
	return invitationResponse{ID: t.ID.Hex(), Email: t.Email, Role: string(t.Role), CreatedAt: t.CreatedAt, ExpiresAt: t.ExpiresAt}
}

func sentResponse(s *auth.Sent) invitationResponse {
	out := toInvitationResponse(s.Token)
	out.Link, out.Mailed = s.Link, &s.Mailed
	return out
}

type inviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func handleInviteStaff(uc *auth.StaffInvitations, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		var req inviteRequest
		if !readJSON(w, r, &req) {
			return
		}
		sent, err := uc.Invite(r.Context(), actor.ID, req.Email, domain.StaffRole(req.Role))
		if err != nil {
			writeError(w, err)
			return
		}
		journal.Record(r.Context(), actor, admin.Entry{Action: domain.ActionStaffInvite, TargetType: "invitation", TargetID: sent.Token.ID, TargetLabel: sent.Token.Email, Detail: req.Role})
		writeJSON(w, http.StatusCreated, sentResponse(sent))
	}
}

func handleResendInvitation(uc *auth.StaffInvitations, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		sent, err := uc.Resend(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		journal.Record(r.Context(), actor, admin.Entry{Action: domain.ActionStaffInvite, TargetType: "invitation", TargetID: sent.Token.ID, TargetLabel: sent.Token.Email, Detail: string(sent.Token.Role)})
		writeJSON(w, http.StatusOK, sentResponse(sent))
	}
}

func handleCancelInvitation(uc *auth.StaffInvitations, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		t, err := uc.Cancel(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		journal.Record(r.Context(), actor, admin.Entry{Action: domain.ActionStaffInviteCancel, TargetType: "invitation", TargetID: t.ID, TargetLabel: t.Email})
		w.WriteHeader(http.StatusNoContent)
	}
}

type invitationPreviewResponse struct {
	Email         string `json:"email"`
	Role          string `json:"role"`
	AccountExists bool   `json:"accountExists"`
}

func handlePreviewInvitation(uc *auth.StaffInvitations) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := uc.Preview(r.Context(), r.PathValue("token"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, invitationPreviewResponse{Email: p.Email, Role: string(p.Role), AccountExists: p.AccountExists})
	}
}

type acceptInvitationRequest struct {
	Token       string `json:"token"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

// acceptedResponse: a new account is signed in at once (AccessToken and
// User are set); an existing one is told to sign in the usual way.
type acceptedResponse struct {
	AccessToken string        `json:"accessToken,omitempty"`
	User        *userResponse `json:"user,omitempty"`
	SignIn      bool          `json:"signIn"`
}

func handleAcceptInvitation(uc *auth.StaffInvitations, journal *admin.Journal, cookies CookieConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req acceptInvitationRequest
		if !readJSON(w, r, &req) {
			return
		}
		acc, err := uc.Accept(r.Context(), auth.AcceptInput{Token: req.Token, DisplayName: req.DisplayName, Password: req.Password})
		if err != nil {
			writeError(w, err)
			return
		}
		// The role was granted by whoever sent the invitation.
		journal.Record(r.Context(), admin.Actor{ID: acc.InvitedBy}, admin.Entry{
			Action: domain.ActionStaffGrant, TargetType: "user", TargetID: acc.User.ID, TargetLabel: acc.User.Email, Detail: string(acc.User.StaffRole),
		})
		if acc.Session == nil {
			writeJSON(w, http.StatusOK, acceptedResponse{SignIn: true})
			return
		}
		setRefreshCookie(w, acc.Session.RefreshToken, acc.Session.RefreshExpiresAt, false, cookies)
		user := toUserResponse(acc.User)
		writeJSON(w, http.StatusOK, acceptedResponse{AccessToken: acc.Session.AccessToken, User: &user})
	}
}

// --- password reset ---

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

func handleForgotPassword(uc *auth.PasswordReset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req forgotPasswordRequest
		if !readJSON(w, r, &req) {
			return
		}
		if err := uc.Request(r.Context(), req.Email); err != nil {
			writeError(w, err)
			return
		}
		// The same answer whether or not the address has an account.
		w.WriteHeader(http.StatusAccepted)
	}
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func handleResetPassword(uc *auth.PasswordReset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req resetPasswordRequest
		if !readJSON(w, r, &req) {
			return
		}
		if err := uc.Reset(r.Context(), req.Token, req.Password); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
