package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// SecondFactor is every use case around an account's second factors:
// setting them up from a signed-in session, and the second step of a
// sign-in. One struct because they share everything.
type SecondFactor struct {
	Users         ports.UserRepository
	RefreshTokens ports.RefreshTokenRepository
	AuthTokens    ports.AuthTokenRepository
	Hasher        ports.PasswordHasher
	Signer        ports.TokenSigner
	TOTP          ports.TOTP
	Passkeys      ports.Passkeys
	Clock         ports.Clock
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

func (uc *SecondFactor) sessions() sessionIssuer {
	return sessionIssuer{RefreshTokens: uc.RefreshTokens, Signer: uc.Signer, Clock: uc.Clock, AccessTTL: uc.AccessTTL, RefreshTTL: uc.RefreshTTL}
}

var errBadCode = domain.Coded("auth.mfa_code_invalid", fmt.Errorf("auth: wrong second-factor code: %w", domain.ErrUnauthorized))

// MFAStatus is what an account's security page shows.
type MFAStatus struct {
	TOTP          bool
	Passkeys      []domain.Passkey
	RecoveryCodes int
}

func statusOf(u *domain.User) *MFAStatus {
	return &MFAStatus{TOTP: u.MFA.HasTOTP(), Passkeys: u.MFA.Passkeys, RecoveryCodes: len(u.MFA.RecoveryHashes)}
}

// PasskeysAvailable reports whether this deployment can offer passkeys.
func (uc *SecondFactor) PasskeysAvailable() bool { return uc.Passkeys != nil }

var errNoPasskeys = domain.Coded("auth.passkeys_unavailable", fmt.Errorf("auth: passkeys are not available on this deployment: %w", domain.ErrConflict))

// Status returns the account's second factors.
func (uc *SecondFactor) Status(ctx context.Context, userID bson.ObjectID) (*MFAStatus, error) {
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return statusOf(u), nil
}

// TOTPSetup is what the owner needs to add the account to their app.
type TOTPSetup struct {
	Secret string
	URL    string
	// QR is URL drawn as a QR code, PNG.
	QR []byte
}

// SetupTOTP starts (or restarts) the authenticator-app enrolment. The
// secret only becomes a factor once ConfirmTOTP sees a code from it.
func (uc *SecondFactor) SetupTOTP(ctx context.Context, userID bson.ObjectID) (*TOTPSetup, error) {
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.MFA.HasTOTP() {
		return nil, domain.Coded("auth.mfa_totp_already_set", fmt.Errorf("auth: an authenticator app is already set up: %w", domain.ErrConflict))
	}
	secret, url, qr, err := uc.TOTP.Generate(u.Email)
	if err != nil {
		return nil, err
	}
	u.MFA.TOTPSecret, u.MFA.TOTPConfirmedAt = secret, nil
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	return &TOTPSetup{Secret: secret, URL: url, QR: qr}, nil
}

// Enrolled is the result of adding a factor: fresh recovery codes when
// the account had none, and an access token for a session that now
// counts as having passed a second factor.
type Enrolled struct {
	RecoveryCodes []string
	AccessToken   string
	Status        *MFAStatus
}

// ConfirmTOTP turns the pending secret into a factor on a right code.
func (uc *SecondFactor) ConfirmTOTP(ctx context.Context, userID bson.ObjectID, code, rawRefresh string) (*Enrolled, error) {
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.MFA.TOTPSecret == "" || u.MFA.HasTOTP() {
		return nil, domain.Coded("auth.mfa_totp_not_pending", fmt.Errorf("auth: no authenticator setup in progress: %w", domain.ErrConflict))
	}
	now := uc.Clock.Now()
	if !uc.TOTP.Validate(u.MFA.TOTPSecret, strings.TrimSpace(code), now) {
		return nil, errBadCode
	}
	u.MFA.TOTPConfirmedAt = &now
	return uc.finishEnrolment(ctx, u, rawRefresh)
}

// finishEnrolment saves the account with its new factor, hands out
// recovery codes when it has none, and upgrades the session.
func (uc *SecondFactor) finishEnrolment(ctx context.Context, u *domain.User, rawRefresh string) (*Enrolled, error) {
	out := &Enrolled{}
	if len(u.MFA.RecoveryHashes) == 0 {
		codes, hashes, err := newRecoveryCodes()
		if err != nil {
			return nil, err
		}
		u.MFA.RecoveryHashes, out.RecoveryCodes = hashes, codes
	}
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	token, err := uc.sessions().upgrade(ctx, u, rawRefresh)
	if err != nil {
		return nil, err
	}
	out.AccessToken, out.Status = token, statusOf(u)
	return out, nil
}

// confirmPassword is asked before a factor is removed or the recovery
// codes replaced: a borrowed open session must not be enough.
func (uc *SecondFactor) confirmPassword(ctx context.Context, userID bson.ObjectID, password string) (*domain.User, error) {
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := uc.Hasher.Compare(u.PasswordHash, password); err != nil {
		return nil, domain.Coded("auth.current_password_wrong", fmt.Errorf("auth: wrong password: %w", domain.ErrUnauthorized))
	}
	return u, nil
}

// DisableTOTP removes the authenticator app. With no factor left the
// recovery codes go too.
func (uc *SecondFactor) DisableTOTP(ctx context.Context, userID bson.ObjectID, password string) (*MFAStatus, error) {
	u, err := uc.confirmPassword(ctx, userID, password)
	if err != nil {
		return nil, err
	}
	u.MFA.TOTPSecret, u.MFA.TOTPConfirmedAt = "", nil
	return uc.saveFactors(ctx, u)
}

func (uc *SecondFactor) saveFactors(ctx context.Context, u *domain.User) (*MFAStatus, error) {
	if !u.MFA.Enabled() {
		u.MFA.RecoveryHashes = nil
	}
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	return statusOf(u), nil
}

// RegenerateRecoveryCodes replaces the recovery codes; the old ones stop
// working.
func (uc *SecondFactor) RegenerateRecoveryCodes(ctx context.Context, userID bson.ObjectID, password string) ([]string, error) {
	u, err := uc.confirmPassword(ctx, userID, password)
	if err != nil {
		return nil, err
	}
	if !u.MFA.Enabled() {
		return nil, domain.Coded("auth.mfa_not_enabled", fmt.Errorf("auth: no second factor to recover: %w", domain.ErrConflict))
	}
	codes, hashes, err := newRecoveryCodes()
	if err != nil {
		return nil, err
	}
	u.MFA.RecoveryHashes = hashes
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	return codes, nil
}

// PasskeySetup is a passkey registration in progress: the options the
// browser needs, and the token that names the ceremony for its finish.
type PasskeySetup struct {
	Options   []byte
	FlowToken string
}

// BeginPasskey starts registering a passkey for the signed-in account.
func (uc *SecondFactor) BeginPasskey(ctx context.Context, userID bson.ObjectID) (*PasskeySetup, error) {
	if uc.Passkeys == nil {
		return nil, errNoPasskeys
	}
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(u.MFA.Passkeys) >= domain.MaxPasskeys {
		return nil, domain.CodedWith("auth.passkey_limit", map[string]any{"max": domain.MaxPasskeys}, fmt.Errorf("auth: at most %d passkeys: %w", domain.MaxPasskeys, domain.ErrConflict))
	}
	options, session, err := uc.Passkeys.BeginRegistration(u)
	if err != nil {
		return nil, err
	}
	raw, err := newRefreshSecret()
	if err != nil {
		return nil, err
	}
	now := uc.Clock.Now()
	if err := uc.AuthTokens.Create(ctx, &domain.AuthToken{
		Kind: domain.AuthTokenPasskeySetup, TokenHash: hashRefreshToken(raw), UserID: u.ID, Data: session,
		CreatedAt: now, ExpiresAt: now.Add(domain.MFAChallengeTTL),
	}); err != nil {
		return nil, err
	}
	return &PasskeySetup{Options: options, FlowToken: raw}, nil
}

// openToken loads a one-time token of the given kind that is still
// usable; anything else is the same refusal.
func (uc *SecondFactor) openToken(ctx context.Context, kind domain.AuthTokenKind, raw string) (*domain.AuthToken, error) {
	expired := domain.Coded("auth.mfa_challenge_expired", fmt.Errorf("auth: this step has expired, start again: %w", domain.ErrUnauthorized))
	if raw == "" {
		return nil, expired
	}
	t, err := uc.AuthTokens.FindByHash(ctx, kind, hashRefreshToken(raw))
	if err != nil || !t.Open(uc.Clock.Now()) {
		return nil, expired
	}
	return t, nil
}

func (uc *SecondFactor) consume(ctx context.Context, t *domain.AuthToken) error {
	now := uc.Clock.Now()
	t.UsedAt = &now
	return uc.AuthTokens.Update(ctx, t)
}

// FinishPasskey stores the passkey the browser created.
func (uc *SecondFactor) FinishPasskey(ctx context.Context, userID bson.ObjectID, flowToken, name string, response []byte, rawRefresh string) (*Enrolled, error) {
	if uc.Passkeys == nil {
		return nil, errNoPasskeys
	}
	t, err := uc.openToken(ctx, domain.AuthTokenPasskeySetup, flowToken)
	if err != nil {
		return nil, err
	}
	if t.UserID != userID {
		return nil, fmt.Errorf("auth: not this account's passkey setup: %w", domain.ErrForbidden)
	}
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > domain.MaxPasskeyNameLen {
		return nil, fmt.Errorf("auth: passkey name too long: %w", domain.ErrInvalid)
	}
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	key, err := uc.Passkeys.FinishRegistration(u, t.Data, response)
	if err != nil {
		return nil, domain.Coded("auth.passkey_rejected", fmt.Errorf("auth: passkey registration refused (%v): %w", err, domain.ErrInvalid))
	}
	if err := uc.consume(ctx, t); err != nil {
		return nil, err
	}
	key.Name, key.CreatedAt = name, uc.Clock.Now()
	u.MFA.Passkeys = append(u.MFA.Passkeys, *key)
	return uc.finishEnrolment(ctx, u, rawRefresh)
}

// RemovePasskey deletes one passkey, by its position in the list the
// status returned (IDs are opaque bytes; the index is what the page has).
func (uc *SecondFactor) RemovePasskey(ctx context.Context, userID bson.ObjectID, credentialID []byte, password string) (*MFAStatus, error) {
	u, err := uc.confirmPassword(ctx, userID, password)
	if err != nil {
		return nil, err
	}
	kept := u.MFA.Passkeys[:0]
	found := false
	for _, p := range u.MFA.Passkeys {
		if subtle.ConstantTimeCompare(p.ID, credentialID) == 1 {
			found = true
			continue
		}
		kept = append(kept, p)
	}
	if !found {
		return nil, domain.ErrNotFound
	}
	u.MFA.Passkeys = kept
	return uc.saveFactors(ctx, u)
}

// challengeUser loads the pending sign-in and its account.
func (uc *SecondFactor) challengeUser(ctx context.Context, mfaToken string) (*domain.AuthToken, *domain.User, error) {
	t, err := uc.openToken(ctx, domain.AuthTokenMFAChallenge, mfaToken)
	if err != nil {
		return nil, nil, err
	}
	u, err := uc.Users.FindByID(ctx, t.UserID)
	if err != nil {
		return nil, nil, err
	}
	if u.Suspended {
		return nil, nil, fmt.Errorf("auth: %w", domain.ErrSuspended)
	}
	return t, u, nil
}

// VerifyInput is the second step of a sign-in with a typed code: one
// from the authenticator app, or a recovery code.
type VerifyInput struct {
	MFAToken     string
	Code         string
	RecoveryCode string
}

// Verify finishes a sign-in whose password step asked for a second
// factor. A recovery code works once.
func (uc *SecondFactor) Verify(ctx context.Context, in VerifyInput) (*LoginOutput, error) {
	t, u, err := uc.challengeUser(ctx, in.MFAToken)
	if err != nil {
		return nil, err
	}
	switch {
	case in.RecoveryCode != "":
		hash := hashRefreshToken(normalizeRecoveryCode(in.RecoveryCode))
		kept, used := u.MFA.RecoveryHashes[:0:0], false
		for _, h := range u.MFA.RecoveryHashes {
			if !used && subtle.ConstantTimeCompare([]byte(h), []byte(hash)) == 1 {
				used = true
				continue
			}
			kept = append(kept, h)
		}
		if !used {
			return nil, errBadCode
		}
		u.MFA.RecoveryHashes = kept
		if err := uc.Users.Update(ctx, u); err != nil {
			return nil, err
		}
	case u.MFA.HasTOTP() && uc.TOTP.Validate(u.MFA.TOTPSecret, strings.TrimSpace(in.Code), uc.Clock.Now()):
	default:
		return nil, errBadCode
	}
	if err := uc.consume(ctx, t); err != nil {
		return nil, err
	}
	return uc.sessions().issue(ctx, u, t.Persistent, true)
}

// BeginPasskeyLogin returns the options the browser needs to sign the
// pending sign-in with a passkey.
func (uc *SecondFactor) BeginPasskeyLogin(ctx context.Context, mfaToken string) ([]byte, error) {
	if uc.Passkeys == nil {
		return nil, errNoPasskeys
	}
	t, u, err := uc.challengeUser(ctx, mfaToken)
	if err != nil {
		return nil, err
	}
	if len(u.MFA.Passkeys) == 0 {
		return nil, domain.Coded("auth.mfa_no_passkey", fmt.Errorf("auth: this account has no passkey: %w", domain.ErrConflict))
	}
	options, session, err := uc.Passkeys.BeginLogin(u)
	if err != nil {
		return nil, err
	}
	t.Data = session
	if err := uc.AuthTokens.Update(ctx, t); err != nil {
		return nil, err
	}
	return options, nil
}

// FinishPasskeyLogin finishes the sign-in with the passkey's signature.
func (uc *SecondFactor) FinishPasskeyLogin(ctx context.Context, mfaToken string, response []byte) (*LoginOutput, error) {
	if uc.Passkeys == nil {
		return nil, errNoPasskeys
	}
	t, u, err := uc.challengeUser(ctx, mfaToken)
	if err != nil {
		return nil, err
	}
	if len(t.Data) == 0 {
		return nil, errBadCode
	}
	signed, err := uc.Passkeys.FinishLogin(u, t.Data, response)
	if err != nil {
		return nil, errBadCode
	}
	for i, p := range u.MFA.Passkeys {
		if subtle.ConstantTimeCompare(p.ID, signed.ID) == 1 {
			u.MFA.Passkeys[i].Data = signed.Data
		}
	}
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	if err := uc.consume(ctx, t); err != nil {
		return nil, err
	}
	return uc.sessions().issue(ctx, u, t.Persistent, true)
}

// newRecoveryCodes returns RecoveryCodeCount codes shaped "ABCDE-FGHIJ"
// and their hashes. Ten base32 characters are 50 bits each: enough for
// a code that can only be tried through a rate-limited endpoint.
func newRecoveryCodes() (codes, hashes []string, err error) {
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	for i := 0; i < domain.RecoveryCodeCount; i++ {
		buf := make([]byte, 7)
		if _, err := rand.Read(buf); err != nil {
			return nil, nil, fmt.Errorf("auth: generate recovery code: %w", err)
		}
		raw := enc.EncodeToString(buf)[:10]
		codes = append(codes, raw[:5]+"-"+raw[5:])
		hashes = append(hashes, hashRefreshToken(raw))
	}
	return codes, hashes, nil
}

// normalizeRecoveryCode drops what a person adds when retyping a code.
func normalizeRecoveryCode(code string) string {
	return strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(strings.TrimSpace(code)))
}
