package admin

import (
	"context"
	"fmt"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ListStaff backs "Équipe & rôles": every account holding a staff
// role, the root first, then by role.
type ListStaff struct {
	Users ports.UserRepository
}

func (uc *ListStaff) Execute(ctx context.Context) ([]*domain.User, error) {
	users, err := uc.Users.List(ctx)
	if err != nil {
		return nil, err
	}
	rank := map[domain.StaffRole]int{}
	for i, r := range domain.StaffRoles {
		rank[r] = i
	}
	out := make([]*domain.User, 0)
	for _, u := range users {
		if u.StaffRole.IsStaff() {
			out = append(out, u)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Root != b.Root {
			return a.Root
		}
		if rank[a.StaffRole] != rank[b.StaffRole] {
			return rank[a.StaffRole] < rank[b.StaffRole]
		}
		return a.CreatedAt.Before(b.CreatedAt)
	})
	return out, nil
}

// SetStaffRole grants a staff role to an existing account, changes it,
// or revokes it (role StaffNone). There is no invitation: the person
// signs up like anyone else, then a super admin gives their account a
// role.
//
// What it refuses, whoever asks:
//   - the root account's role never changes (see auth.EnsureRootAdmin);
//   - nobody changes their own role, so a super admin can't lock
//     themselves out by accident;
//   - the last super admin keeps the role;
//   - an account that owns a farm doesn't join the staff: staff don't
//     sell on the marketplace they police.
type SetStaffRole struct {
	Users ports.UserRepository
	Farms ports.FarmRepository
	// Sessions, when set, makes the change apply on the account's very
	// next request.
	Sessions SessionForgetter
}

type SetStaffRoleInput struct {
	Actor Actor
	// UserID or Email names the account; UserID wins when both are set.
	UserID bson.ObjectID
	Email  string
	Role   domain.StaffRole
}

type SetStaffRoleOutput struct {
	User *domain.User
	// Previous is the role the account held before.
	Previous domain.StaffRole
}

func (uc *SetStaffRole) Execute(ctx context.Context, in SetStaffRoleInput) (*SetStaffRoleOutput, error) {
	if in.Role != domain.StaffNone && !in.Role.IsValid() {
		return nil, domain.Coded("admin.staff_role_unknown", fmt.Errorf("admin: unknown staff role %q: %w", in.Role, domain.ErrInvalid))
	}
	u, err := findAccount(ctx, uc.Users, in.UserID, in.Email)
	if err != nil {
		return nil, err
	}
	previous := u.StaffRole
	switch {
	case u.Root:
		return nil, domain.Coded("admin.root_protected", fmt.Errorf("admin: the root account's role can't change: %w", domain.ErrForbidden))
	case u.ID == in.Actor.ID:
		return nil, domain.Coded("admin.self_action", fmt.Errorf("admin: nobody changes their own role: %w", domain.ErrForbidden))
	case previous == in.Role:
		return &SetStaffRoleOutput{User: u, Previous: previous}, nil
	}

	if previous == domain.StaffSuperAdmin {
		others, err := uc.otherSuperAdmins(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		if others == 0 {
			return nil, domain.Coded("admin.last_super_admin", fmt.Errorf("admin: the last super admin keeps the role: %w", domain.ErrConflict))
		}
	}
	if !previous.IsStaff() {
		owned, err := uc.Farms.ListByOwner(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		if len(owned) > 0 {
			return nil, domain.CodedWith("admin.staff_owns_farm", map[string]any{"count": len(owned)}, fmt.Errorf("admin: an account that owns a farm can't join the staff: %w", domain.ErrConflict))
		}
	}

	u.StaffRole = in.Role
	if in.Role.IsStaff() {
		// A limit raised for a seller means nothing on a staff account.
		u.FarmLimit, u.FarmLimitNote = 0, nil
	}
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	if uc.Sessions != nil {
		uc.Sessions.Forget(u.ID)
	}
	return &SetStaffRoleOutput{User: u, Previous: previous}, nil
}

func (uc *SetStaffRole) otherSuperAdmins(ctx context.Context, except bson.ObjectID) (int, error) {
	users, err := uc.Users.List(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, u := range users {
		if u.ID != except && u.StaffRole == domain.StaffSuperAdmin && !u.Suspended {
			n++
		}
	}
	return n, nil
}
