// Package admin implements the back office: platform-wide oversight
// (accounts, farms, cycles, listings, escrow, reports, staff) that
// every other module deliberately scopes to "the caller's own farm" or
// "public data". Who may call what is decided at the HTTP layer, one
// permission per route (see domain.Permission); the rules that depend
// on who is acting and on whom (nobody touches the root account, staff
// are suspended only by a super admin) are enforced here.
package admin

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Actor is the staff member behind an action: their role is the one
// stored on their account right now, not the one in their token.
type Actor struct {
	ID   bson.ObjectID
	Role domain.StaffRole
}

// AccountItem is an account as the back office lists it.
type AccountItem struct {
	User      *domain.User
	FarmCount int
}

// ListUsers backs the back office's "Comptes" page.
type ListUsers struct {
	Users ports.UserRepository
	Farms ports.FarmRepository
}

func (uc *ListUsers) Execute(ctx context.Context) ([]AccountItem, error) {
	users, err := uc.Users.List(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := farmCounts(ctx, uc.Farms)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(users, func(i, j int) bool { return users[i].CreatedAt.After(users[j].CreatedAt) })
	out := make([]AccountItem, 0, len(users))
	for _, u := range users {
		out = append(out, AccountItem{User: u, FarmCount: counts[u.ID]})
	}
	return out, nil
}

// farmCounts is how many farms each account owns, in one read.
func farmCounts(ctx context.Context, farms ports.FarmRepository) (map[bson.ObjectID]int, error) {
	all, err := farms.ListByStatus(ctx, "")
	if err != nil {
		return nil, err
	}
	counts := make(map[bson.ObjectID]int, len(all))
	for _, f := range all {
		counts[f.OwnerID]++
	}
	return counts, nil
}

// MaxFarmLimit caps what staff can grant one account.
const MaxFarmLimit = 50

// SetFarmLimit lets staff make an exception to the platform's number
// of farms per account (MAX_FARMS_PER_ACCOUNT), for one account. An
// exception is always explained, attributed, and never below what the
// account already owns. Limit 0 ends the exception: the account is back
// on the platform default, which removes nothing it has, only stops new
// farms.
type SetFarmLimit struct {
	Users ports.UserRepository
	Farms ports.FarmRepository
	Clock ports.Clock
}

type SetFarmLimitInput struct {
	Actor Actor
	// UserID or Email names the account; UserID wins when both are set.
	UserID bson.ObjectID
	Email  string
	Limit  int
	Reason string
}

func (uc *SetFarmLimit) Execute(ctx context.Context, in SetFarmLimitInput) (*AccountItem, error) {
	if in.Limit < 0 || in.Limit > MaxFarmLimit {
		return nil, domain.CodedWith("admin.farm_limit_range", map[string]any{"max": MaxFarmLimit}, fmt.Errorf("admin: farm limit must be between 0 and %d: %w", MaxFarmLimit, domain.ErrInvalid))
	}
	reason := strings.TrimSpace(in.Reason)
	if in.Limit > 0 && reason == "" {
		return nil, domain.Coded("admin.farm_limit_reason_required", fmt.Errorf("admin: a farm limit exception needs a reason: %w", domain.ErrInvalid))
	}
	if utf8.RuneCountInString(reason) > domain.MaxFarmLimitReasonLen {
		return nil, domain.CodedWith("admin.reason_too_long", map[string]any{"max": domain.MaxFarmLimitReasonLen}, fmt.Errorf("admin: reason must be at most %d characters: %w", domain.MaxFarmLimitReasonLen, domain.ErrInvalid))
	}

	u, err := findAccount(ctx, uc.Users, in.UserID, in.Email)
	if err != nil {
		return nil, err
	}
	if u.StaffRole.IsStaff() {
		return nil, domain.Coded("admin.farm_limit_staff", fmt.Errorf("admin: staff accounts don't own farms: %w", domain.ErrConflict))
	}
	owned, err := uc.Farms.ListByOwner(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	if in.Limit > 0 && in.Limit < len(owned) {
		return nil, domain.CodedWith("admin.farm_limit_below_owned", map[string]any{"owned": len(owned)}, fmt.Errorf("admin: the account already owns %d farm(s): %w", len(owned), domain.ErrConflict))
	}

	u.FarmLimit, u.FarmLimitNote = in.Limit, nil
	if in.Limit > 0 {
		u.FarmLimitNote = &domain.FarmLimitNote{Reason: reason, SetBy: in.Actor.ID, SetAt: uc.Clock.Now()}
	}
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	return &AccountItem{User: u, FarmCount: len(owned)}, nil
}

// findAccount resolves an account by ID or, failing that, by email.
func findAccount(ctx context.Context, users ports.UserRepository, id bson.ObjectID, email string) (*domain.User, error) {
	if !id.IsZero() {
		return users.FindByID(ctx, id)
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, domain.Coded("admin.account_required", fmt.Errorf("admin: no account given: %w", domain.ErrInvalid))
	}
	u, err := users.FindByEmail(ctx, email)
	if err != nil {
		return nil, domain.Coded("admin.account_not_found", fmt.Errorf("admin: no account with email %s: %w", email, err))
	}
	return u, nil
}

// FarmLimitException is one account whose limit staff changed.
type FarmLimitException struct {
	AccountItem
	// SetByName and SetByRole are who granted it, as they are today.
	SetByName string
	SetByRole domain.StaffRole
}

// FarmLimits is the platform default and every exception to it.
type FarmLimits struct {
	Default    int
	Max        int
	Exceptions []FarmLimitException
}

// ListFarmLimits backs the "Limite de fermes" card.
type ListFarmLimits struct {
	Users ports.UserRepository
	Farms ports.FarmRepository
	// Default is MAX_FARMS_PER_ACCOUNT as read at startup.
	Default int
}

func (uc *ListFarmLimits) Execute(ctx context.Context) (*FarmLimits, error) {
	users, err := uc.Users.List(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := farmCounts(ctx, uc.Farms)
	if err != nil {
		return nil, err
	}
	byID := make(map[bson.ObjectID]*domain.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	out := &FarmLimits{Default: uc.Default, Max: MaxFarmLimit, Exceptions: []FarmLimitException{}}
	if out.Default <= 0 {
		out.Default = domain.DefaultMaxFarmsPerAccount
	}
	for _, u := range users {
		if u.FarmLimit <= 0 {
			continue
		}
		ex := FarmLimitException{AccountItem: AccountItem{User: u, FarmCount: counts[u.ID]}}
		if u.FarmLimitNote != nil {
			if by := byID[u.FarmLimitNote.SetBy]; by != nil {
				ex.SetByName, ex.SetByRole = displayName(by), by.StaffRole
			}
		}
		out.Exceptions = append(out.Exceptions, ex)
	}
	sort.SliceStable(out.Exceptions, func(i, j int) bool {
		a, b := out.Exceptions[i].User.FarmLimitNote, out.Exceptions[j].User.FarmLimitNote
		if a == nil || b == nil {
			return a != nil
		}
		return a.SetAt.After(b.SetAt)
	})
	return out, nil
}

// displayName is what the back office calls an account.
func displayName(u *domain.User) string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Email
}

// SuspendUser is "Suspendre le compte": it blocks the account from
// signing in and from creating anything (see domain.User.Suspended); it
// does not affect reservations or escrow already in flight.
type SuspendUser struct {
	Users ports.UserRepository
	// Sessions, when set, is told so the suspension applies on the
	// user's very next request instead of after the gate's cache delay.
	Sessions SessionForgetter
}

// SessionForgetter drops what the auth layer cached about an account.
type SessionForgetter interface {
	Forget(userID bson.ObjectID)
}

func (uc *SuspendUser) Execute(ctx context.Context, actor Actor, userID bson.ObjectID) (*domain.User, error) {
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := checkSuspendable(actor, u); err != nil {
		return nil, err
	}
	return setSuspended(ctx, uc.Users, uc.Sessions, u, true)
}

// ReactivateUser lifts a suspension.
type ReactivateUser struct {
	Users    ports.UserRepository
	Sessions SessionForgetter
}

func (uc *ReactivateUser) Execute(ctx context.Context, actor Actor, userID bson.ObjectID) (*domain.User, error) {
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.StaffRole.IsStaff() && actor.Role != domain.StaffSuperAdmin {
		return nil, domain.Coded("admin.staff_needs_super_admin", fmt.Errorf("admin: only a super admin acts on a staff account: %w", domain.ErrForbidden))
	}
	return setSuspended(ctx, uc.Users, uc.Sessions, u, false)
}

func setSuspended(ctx context.Context, users ports.UserRepository, sessions SessionForgetter, u *domain.User, suspended bool) (*domain.User, error) {
	u.Suspended = suspended
	if err := users.Update(ctx, u); err != nil {
		return nil, err
	}
	if sessions != nil {
		sessions.Forget(u.ID)
	}
	return u, nil
}

// checkSuspendable is who may suspend whom: never the root account,
// never yourself, and a member of the staff only by a super admin.
func checkSuspendable(actor Actor, target *domain.User) error {
	switch {
	case target.Root:
		return domain.Coded("admin.root_protected", fmt.Errorf("admin: the root account can't be suspended: %w", domain.ErrForbidden))
	case target.ID == actor.ID:
		return domain.Coded("admin.self_action", fmt.Errorf("admin: nobody suspends their own account: %w", domain.ErrForbidden))
	case target.StaffRole.IsStaff() && actor.Role != domain.StaffSuperAdmin:
		return domain.Coded("admin.staff_needs_super_admin", fmt.Errorf("admin: only a super admin acts on a staff account: %w", domain.ErrForbidden))
	}
	return nil
}
