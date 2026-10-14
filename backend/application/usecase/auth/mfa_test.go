package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// fakeAuthTokens keeps one-time tokens in memory.
type fakeAuthTokens struct{ all []*domain.AuthToken }

func (f *fakeAuthTokens) Create(_ context.Context, t *domain.AuthToken) error {
	if t.ID.IsZero() {
		t.ID = bson.NewObjectID()
	}
	f.all = append(f.all, t)
	return nil
}

func (f *fakeAuthTokens) FindByHash(_ context.Context, kind domain.AuthTokenKind, hash string) (*domain.AuthToken, error) {
	for _, t := range f.all {
		if t.Kind == kind && t.TokenHash == hash {
			return t, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeAuthTokens) FindByID(_ context.Context, id bson.ObjectID) (*domain.AuthToken, error) {
	for _, t := range f.all {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeAuthTokens) Update(context.Context, *domain.AuthToken) error { return nil }

func (f *fakeAuthTokens) ListOpen(_ context.Context, kind domain.AuthTokenKind, now time.Time) ([]*domain.AuthToken, error) {
	var out []*domain.AuthToken
	for _, t := range f.all {
		if t.Kind == kind && t.Open(now) {
			out = append(out, t)
		}
	}
	return out, nil
}

// fakeTOTP accepts exactly one code.
type fakeTOTP struct{}

func (fakeTOTP) Generate(account string) (string, string, []byte, error) {
	return "SECRET", "otpauth://totp/Betailo:" + account, []byte("png"), nil
}
func (fakeTOTP) Validate(secret, code string, _ time.Time) bool {
	return secret == "SECRET" && code == "123456"
}

// fakeMailer remembers what was sent.
type fakeMailer struct{ sent []ports.Mail }

func (f *fakeMailer) Send(_ context.Context, m ports.Mail) error {
	f.sent = append(f.sent, m)
	return nil
}

func mfaFixture(t *testing.T) (*SecondFactor, *Login, *fakeUsers, *fakeRefreshTokens, *domain.User) {
	t.Helper()
	users, refresh, tokens := newFakeUsers(), newFakeRefreshTokens(), &fakeAuthTokens{}
	u := registeredUser(t, users, "rita@betailo.test", "correct-horse")
	clock := fakeClock{now: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)}
	sf := &SecondFactor{Users: users, RefreshTokens: refresh, AuthTokens: tokens, Hasher: fakeHasher{}, Signer: &fakeSigner{}, TOTP: fakeTOTP{}, Clock: clock}
	login := &Login{Users: users, RefreshTokens: refresh, Hasher: fakeHasher{}, Signer: &fakeSigner{}, Clock: clock, AuthTokens: tokens}
	return sf, login, users, refresh, u
}

func TestSecondFactor_TOTPEnrolmentThenSignIn(t *testing.T) {
	sf, login, _, refresh, u := mfaFixture(t)
	ctx := context.Background()

	// Signed in without a factor: an ordinary session.
	first, err := login.Execute(ctx, LoginInput{Email: u.Email, Password: "correct-horse", Remember: true})
	if err != nil || first.MFARequired {
		t.Fatalf("login before enrolment = %+v, %v; want a session", first, err)
	}

	setup, err := sf.SetupTOTP(ctx, u.ID)
	if err != nil || setup.Secret == "" || !strings.HasPrefix(setup.URL, "otpauth://") {
		t.Fatalf("SetupTOTP = %+v, %v", setup, err)
	}
	// Not a factor until confirmed.
	if again, _ := login.Execute(ctx, LoginInput{Email: u.Email, Password: "correct-horse"}); again == nil || again.MFARequired {
		t.Fatalf("an unconfirmed secret must not gate sign-in")
	}
	if _, err := sf.ConfirmTOTP(ctx, u.ID, "000000", first.RefreshToken); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("confirm with a wrong code = %v, want unauthorized", err)
	}
	enrolled, err := sf.ConfirmTOTP(ctx, u.ID, "123456", first.RefreshToken)
	if err != nil {
		t.Fatalf("ConfirmTOTP: %v", err)
	}
	if len(enrolled.RecoveryCodes) != domain.RecoveryCodeCount || !enrolled.Status.TOTP || enrolled.AccessToken == "" {
		t.Fatalf("enrolled = %d codes, totp %v", len(enrolled.RecoveryCodes), enrolled.Status.TOTP)
	}
	// The session that enrolled now counts as verified.
	if stored := refresh.byHash[hashRefreshToken(first.RefreshToken)]; stored == nil || !stored.MFA {
		t.Errorf("the enrolling session was not marked as having passed a second factor")
	}

	// From now on the password alone opens no session.
	step, err := login.Execute(ctx, LoginInput{Email: u.Email, Password: "correct-horse", Remember: true})
	if err != nil || !step.MFARequired || step.AccessToken != "" || step.MFAToken == "" {
		t.Fatalf("login after enrolment = %+v, %v; want a pending second step", step, err)
	}
	if _, err := sf.Verify(ctx, VerifyInput{MFAToken: step.MFAToken, Code: "111111"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("verify with a wrong code = %v", err)
	}
	session, err := sf.Verify(ctx, VerifyInput{MFAToken: step.MFAToken, Code: "123456"})
	if err != nil || session.AccessToken == "" || !session.Persistent {
		t.Fatalf("verify = %+v, %v; want a remembered session", session, err)
	}
	if stored := refresh.byHash[hashRefreshToken(session.RefreshToken)]; stored == nil || !stored.MFA {
		t.Errorf("the verified session is not marked MFA")
	}
	// The pending step is spent.
	if _, err := sf.Verify(ctx, VerifyInput{MFAToken: step.MFAToken, Code: "123456"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("reusing a spent second step = %v, want unauthorized", err)
	}

	// A recovery code works once.
	code := enrolled.RecoveryCodes[0]
	for attempt, wantOK := range []bool{true, false} {
		step, _ := login.Execute(ctx, LoginInput{Email: u.Email, Password: "correct-horse"})
		_, err := sf.Verify(ctx, VerifyInput{MFAToken: step.MFAToken, RecoveryCode: strings.ToLower(code)})
		if (err == nil) != wantOK {
			t.Errorf("recovery code, attempt %d: err = %v, want ok=%v", attempt+1, err, wantOK)
		}
	}

	// Removing the factor asks for the password, and drops the codes.
	if _, err := sf.DisableTOTP(ctx, u.ID, "wrong"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("disable with a wrong password = %v", err)
	}
	status, err := sf.DisableTOTP(ctx, u.ID, "correct-horse")
	if err != nil || status.TOTP || status.RecoveryCodes != 0 {
		t.Errorf("disable = %+v, %v", status, err)
	}
}

func TestStaffInvitations_NewAccountAndExistingAccount(t *testing.T) {
	users, refresh, tokens, mailer := newFakeUsers(), newFakeRefreshTokens(), &fakeAuthTokens{}, &fakeMailer{}
	clock := fakeClock{now: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)}
	inviter := registeredUser(t, users, "root@betailo.test", "correct-horse")
	uc := &StaffInvitations{
		Users: users, Farms: noFarms{}, AuthTokens: tokens, RefreshTokens: refresh, Hasher: fakeHasher{}, Signer: &fakeSigner{},
		Mailer: mailer, Clock: clock, PublicURL: "https://betailo.example/",
	}
	ctx := context.Background()

	if _, err := uc.Invite(ctx, inviter.ID, "not-an-email", domain.StaffModerator); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("invite a bad address = %v", err)
	}
	sent, err := uc.Invite(ctx, inviter.ID, " Ngo@Betailo.test ", domain.StaffModerator)
	if err != nil {
		t.Fatalf("Invite: %v", err)
	}
	if !strings.HasPrefix(sent.Link, "https://betailo.example/invitation/") || !sent.Mailed || len(mailer.sent) != 1 || mailer.sent[0].To != "ngo@betailo.test" {
		t.Fatalf("sent = %+v, mails %d", sent, len(mailer.sent))
	}
	raw := strings.TrimPrefix(sent.Link, "https://betailo.example/invitation/")

	if p, err := uc.Preview(ctx, raw); err != nil || p.AccountExists || p.Role != domain.StaffModerator {
		t.Fatalf("Preview = %+v, %v", p, err)
	}
	if _, err := uc.Accept(ctx, AcceptInput{Token: raw, Password: "short"}); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("accept with a short password = %v", err)
	}
	acc, err := uc.Accept(ctx, AcceptInput{Token: raw, DisplayName: "Ngo B.", Password: "long-enough-1"})
	if err != nil || acc.Session == nil || acc.User.StaffRole != domain.StaffModerator {
		t.Fatalf("Accept (new account) = %+v, %v", acc, err)
	}
	if _, err := uc.Accept(ctx, AcceptInput{Token: raw, Password: "long-enough-1"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("accepting twice = %v, want not found", err)
	}
	if _, err := uc.Invite(ctx, inviter.ID, "ngo@betailo.test", domain.StaffFinance); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("inviting a staff member = %v, want conflict", err)
	}

	// An address that already has an account proves it with its password
	// and is not signed in here, so its second factor is never skipped.
	registeredUser(t, users, "kevin@betailo.test", "kevin-password")
	sent, _ = uc.Invite(ctx, inviter.ID, "kevin@betailo.test", domain.StaffSupport)
	raw = strings.TrimPrefix(sent.Link, "https://betailo.example/invitation/")
	if _, err := uc.Accept(ctx, AcceptInput{Token: raw, Password: "not-his"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("accept with the wrong password = %v", err)
	}
	acc, err = uc.Accept(ctx, AcceptInput{Token: raw, Password: "kevin-password"})
	if err != nil || acc.Session != nil || acc.User.StaffRole != domain.StaffSupport {
		t.Fatalf("Accept (existing account) = %+v, %v", acc, err)
	}

	// Cancelling makes the link dead; the list shows what still waits.
	sent, _ = uc.Invite(ctx, inviter.ID, "josiane@betailo.test", domain.StaffFinance)
	if list, _ := uc.List(ctx); len(list) != 1 || list[0].Token.Email != "josiane@betailo.test" {
		t.Errorf("pending invitations = %d", len(list))
	}
	if _, err := uc.Cancel(ctx, sent.Token.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if _, err := uc.Preview(ctx, strings.TrimPrefix(sent.Link, "https://betailo.example/invitation/")); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("a cancelled invitation still previews: %v", err)
	}
}

func TestPasswordReset(t *testing.T) {
	users, refresh, tokens, mailer := newFakeUsers(), newFakeRefreshTokens(), &fakeAuthTokens{}, &fakeMailer{}
	clock := fakeClock{now: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)}
	u := registeredUser(t, users, "root@betailo.test", "old-password")
	uc := &PasswordReset{Users: users, AuthTokens: tokens, RefreshTokens: refresh, Hasher: fakeHasher{}, Mailer: mailer, Clock: clock, PublicURL: "https://betailo.example"}
	ctx := context.Background()

	// An unknown address gets the same silence, and no mail.
	if err := uc.Request(ctx, "nobody@betailo.test"); err != nil || len(mailer.sent) != 0 {
		t.Fatalf("request for an unknown address = %v, mails %d", err, len(mailer.sent))
	}
	if err := uc.Request(ctx, "Root@Betailo.test"); err != nil || len(mailer.sent) != 1 {
		t.Fatalf("request = %v, mails %d", err, len(mailer.sent))
	}
	text := mailer.sent[0].Text
	raw := strings.Fields(text[strings.Index(text, "https://betailo.example/reset-password/"):])[0]
	raw = strings.TrimPrefix(raw, "https://betailo.example/reset-password/")

	if err := uc.Reset(ctx, raw, "short"); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("reset with a short password = %v", err)
	}
	if err := uc.Reset(ctx, raw, "new-password-1"); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if err := (fakeHasher{}).Compare(users.byEmail[u.Email].PasswordHash, "new-password-1"); err != nil {
		t.Errorf("the password did not change")
	}
	if err := uc.Reset(ctx, raw, "another-password"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("a reset link used twice = %v, want not found", err)
	}
}

// noFarms is a farm repository where nobody owns anything.
type noFarms struct{ ports.FarmRepository }

func (noFarms) ListByOwner(context.Context, bson.ObjectID) ([]*domain.Farm, error) { return nil, nil }
