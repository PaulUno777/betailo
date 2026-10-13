package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// StaffInvitations invites people onto the staff by e-mail address. An
// invitation is a one-time link: whoever opens it sets up (or proves
// they own) the account with that address, which then holds the role.
// Granting a role to an account that already exists stays possible
// without it (see admin.SetStaffRole).
type StaffInvitations struct {
	Users         ports.UserRepository
	Farms         ports.FarmRepository
	AuthTokens    ports.AuthTokenRepository
	RefreshTokens ports.RefreshTokenRepository
	Hasher        ports.PasswordHasher
	Signer        ports.TokenSigner
	Mailer        ports.Mailer
	Clock         ports.Clock
	// PublicURL is the site's address, for the link in the e-mail.
	PublicURL  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	// Forget, when set, makes a role granted to an existing account
	// apply on its very next request.
	Forget func(userID bson.ObjectID)
}

// Invitation is a pending invitation as the team page lists it.
type Invitation struct {
	Token       *domain.AuthToken
	InviterName string
}

// Sent is an invitation just created or renewed. Link is given once, so
// it can also be passed on by hand; it is never stored.
type Sent struct {
	Token *domain.AuthToken
	Link  string
	// Mailed is false when the e-mail could not be sent: the link is
	// then the only way to deliver the invitation.
	Mailed bool
}

func (uc *StaffInvitations) link(raw string) string {
	return strings.TrimRight(uc.PublicURL, "/") + "/invitation/" + raw
}

// invitable refuses the addresses that can't join the staff this way.
func (uc *StaffInvitations) invitable(ctx context.Context, email string) error {
	u, err := uc.Users.FindByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if u.StaffRole.IsStaff() {
		return domain.Coded("admin.staff_already_member", fmt.Errorf("auth: %s is already on the staff: %w", email, domain.ErrConflict))
	}
	owned, err := uc.Farms.ListByOwner(ctx, u.ID)
	if err != nil {
		return err
	}
	if len(owned) > 0 {
		return domain.CodedWith("admin.staff_owns_farm", map[string]any{"count": len(owned)}, fmt.Errorf("auth: an account that owns a farm can't join the staff: %w", domain.ErrConflict))
	}
	return nil
}

func (uc *StaffInvitations) send(ctx context.Context, t *domain.AuthToken) (*Sent, error) {
	raw, err := newRefreshSecret()
	if err != nil {
		return nil, err
	}
	now := uc.Clock.Now()
	t.TokenHash, t.ExpiresAt = hashRefreshToken(raw), now.Add(domain.InvitationTTL)
	if t.ID.IsZero() {
		t.CreatedAt = now
		err = uc.AuthTokens.Create(ctx, t)
	} else {
		err = uc.AuthTokens.Update(ctx, t)
	}
	if err != nil {
		return nil, err
	}
	link := uc.link(raw)
	mailErr := uc.Mailer.Send(ctx, ports.Mail{
		To:      t.Email,
		Subject: "Invitation à rejoindre l’équipe Betailo",
		Text: fmt.Sprintf("Bonjour,\n\nVous êtes invité à rejoindre l’équipe Betailo avec le rôle « %s ».\n\n"+
			"Ouvrez ce lien pour accepter l’invitation et choisir votre mot de passe :\n%s\n\n"+
			"Le lien est valable 7 jours. Si vous n’attendiez pas ce message, ignorez-le.\n", staffRoleName(t.Role), link),
	})
	return &Sent{Token: t, Link: link, Mailed: mailErr == nil}, nil
}

func staffRoleName(r domain.StaffRole) string {
	switch r {
	case domain.StaffSuperAdmin:
		return "Super admin"
	case domain.StaffModerator:
		return "Modération"
	case domain.StaffFinance:
		return "Finance"
	case domain.StaffSupport:
		return "Support"
	}
	return string(r)
}

// Invite creates an invitation and e-mails its link. An earlier pending
// invitation for the same address is replaced.
func (uc *StaffInvitations) Invite(ctx context.Context, inviter bson.ObjectID, email string, role domain.StaffRole) (*Sent, error) {
	email = normalizeEmail(email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, domain.Coded("auth.email_invalid", fmt.Errorf("auth: a valid e-mail address is required: %w", domain.ErrInvalid))
	}
	if !role.IsValid() {
		return nil, domain.Coded("admin.staff_role_unknown", fmt.Errorf("auth: unknown staff role %q: %w", role, domain.ErrInvalid))
	}
	if err := uc.invitable(ctx, email); err != nil {
		return nil, err
	}
	now := uc.Clock.Now()
	open, err := uc.AuthTokens.ListOpen(ctx, domain.AuthTokenInvitation, now)
	if err != nil {
		return nil, err
	}
	for _, t := range open {
		if t.Email == email {
			t.UsedAt = &now
			if err := uc.AuthTokens.Update(ctx, t); err != nil {
				return nil, err
			}
		}
	}
	return uc.send(ctx, &domain.AuthToken{Kind: domain.AuthTokenInvitation, Email: email, Role: role, CreatedBy: inviter})
}

// pending loads an invitation still waiting, by its ID.
func (uc *StaffInvitations) pending(ctx context.Context, id bson.ObjectID) (*domain.AuthToken, error) {
	t, err := uc.AuthTokens.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.Kind != domain.AuthTokenInvitation || t.UsedAt != nil {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

// Resend renews an invitation: a new link, seven more days, the old
// link stops working.
func (uc *StaffInvitations) Resend(ctx context.Context, id bson.ObjectID) (*Sent, error) {
	t, err := uc.pending(ctx, id)
	if err != nil {
		return nil, err
	}
	return uc.send(ctx, t)
}

// Cancel withdraws an invitation.
func (uc *StaffInvitations) Cancel(ctx context.Context, id bson.ObjectID) (*domain.AuthToken, error) {
	t, err := uc.pending(ctx, id)
	if err != nil {
		return nil, err
	}
	now := uc.Clock.Now()
	t.UsedAt = &now
	return t, uc.AuthTokens.Update(ctx, t)
}

// List returns the invitations still waiting, newest first.
func (uc *StaffInvitations) List(ctx context.Context) ([]Invitation, error) {
	open, err := uc.AuthTokens.ListOpen(ctx, domain.AuthTokenInvitation, uc.Clock.Now())
	if err != nil {
		return nil, err
	}
	out := make([]Invitation, 0, len(open))
	for _, t := range open {
		inv := Invitation{Token: t}
		if u, err := uc.Users.FindByID(ctx, t.CreatedBy); err == nil {
			inv.InviterName = u.DisplayName
			if inv.InviterName == "" {
				inv.InviterName = u.Email
			}
		}
		out = append(out, inv)
	}
	return out, nil
}

var errInvitationGone = domain.Coded("auth.invitation_invalid", fmt.Errorf("auth: this invitation is no longer valid: %w", domain.ErrNotFound))

func (uc *StaffInvitations) byLink(ctx context.Context, raw string) (*domain.AuthToken, error) {
	if raw == "" {
		return nil, errInvitationGone
	}
	t, err := uc.AuthTokens.FindByHash(ctx, domain.AuthTokenInvitation, hashRefreshToken(raw))
	if err != nil || !t.Open(uc.Clock.Now()) {
		return nil, errInvitationGone
	}
	return t, nil
}

// Preview is what the acceptance page shows before anything is typed.
type Preview struct {
	Email string
	Role  domain.StaffRole
	// AccountExists: the address already has an account; accepting asks
	// for its password instead of creating one.
	AccountExists bool
}

// Preview reads an invitation from its link.
func (uc *StaffInvitations) Preview(ctx context.Context, raw string) (*Preview, error) {
	t, err := uc.byLink(ctx, raw)
	if err != nil {
		return nil, err
	}
	_, err = uc.Users.FindByEmail(ctx, t.Email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	return &Preview{Email: t.Email, Role: t.Role, AccountExists: err == nil}, nil
}

// AcceptInput is what the invited person types.
type AcceptInput struct {
	Token       string
	DisplayName string
	// Password is the new account's password, or the existing account's
	// own password as proof.
	Password string
}

// Accepted is the outcome. Session is set for a new account, which is
// signed in at once; an existing account signs in the usual way, so its
// second factor is never skipped.
type Accepted struct {
	User    *domain.User
	Session *LoginOutput
	// InvitedBy is who sent the invitation, for the journal.
	InvitedBy bson.ObjectID
}

// Accept gives the invited address its role.
func (uc *StaffInvitations) Accept(ctx context.Context, in AcceptInput) (*Accepted, error) {
	t, err := uc.byLink(ctx, in.Token)
	if err != nil {
		return nil, err
	}
	now := uc.Clock.Now()
	out := &Accepted{InvitedBy: t.CreatedBy}

	u, err := uc.Users.FindByEmail(ctx, t.Email)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		if len(in.Password) < minPasswordLength {
			return nil, domain.CodedWith("auth.password_too_short", map[string]any{"min": minPasswordLength}, fmt.Errorf("auth: password must be at least %d characters: %w", minPasswordLength, domain.ErrInvalid))
		}
		name := strings.TrimSpace(in.DisplayName)
		if utf8.RuneCountInString(name) > domain.MaxDisplayNameLen {
			return nil, domain.CodedWith("account.display_name_too_long", map[string]any{"max": domain.MaxDisplayNameLen}, fmt.Errorf("auth: display name too long: %w", domain.ErrInvalid))
		}
		hash, err := uc.Hasher.Hash(in.Password)
		if err != nil {
			return nil, err
		}
		u = &domain.User{Email: t.Email, PasswordHash: hash, DisplayName: name, StaffRole: t.Role, CreatedAt: now}
		if err := uc.Users.Create(ctx, u); err != nil {
			return nil, err
		}
		issuer := sessionIssuer{RefreshTokens: uc.RefreshTokens, Signer: uc.Signer, Clock: uc.Clock, AccessTTL: uc.AccessTTL, RefreshTTL: uc.RefreshTTL}
		if out.Session, err = issuer.issue(ctx, u, false, false); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if err := uc.Hasher.Compare(u.PasswordHash, in.Password); err != nil {
			return nil, domain.Coded("auth.current_password_wrong", fmt.Errorf("auth: wrong password: %w", domain.ErrUnauthorized))
		}
		if u.Suspended {
			return nil, fmt.Errorf("auth: %w", domain.ErrSuspended)
		}
		// The account may have changed since the invitation was sent.
		if err := uc.invitable(ctx, t.Email); err != nil {
			return nil, err
		}
		u.StaffRole, u.FarmLimit, u.FarmLimitNote = t.Role, 0, nil
		if err := uc.Users.Update(ctx, u); err != nil {
			return nil, err
		}
		if uc.Forget != nil {
			uc.Forget(u.ID)
		}
	}

	t.UsedAt, t.UserID = &now, u.ID
	if err := uc.AuthTokens.Update(ctx, t); err != nil {
		return nil, err
	}
	out.User = u
	return out, nil
}
