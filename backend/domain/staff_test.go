package domain

import "testing"

func TestStaffRole_Can(t *testing.T) {
	tests := []struct {
		role StaffRole
		perm Permission
		want bool
	}{
		{StaffSuperAdmin, PermStaffManage, true},
		{StaffSuperAdmin, PermEscrowSettle, true},
		{StaffModerator, PermFarmsReview, true},
		{StaffModerator, PermFarmLimitSet, true},
		{StaffModerator, PermEscrowRead, false},
		{StaffModerator, PermStaffManage, false},
		{StaffFinance, PermEscrowSettle, true},
		{StaffFinance, PermFarmsReview, false},
		{StaffFinance, PermAccountsSuspend, false},
		{StaffSupport, PermAccountsRead, true},
		{StaffSupport, PermEscrowRead, true},
		{StaffSupport, PermEscrowSettle, false},
		{StaffSupport, PermReportsResolve, false},
		{StaffNone, PermOverviewRead, false},
		{StaffRole("intern"), PermOverviewRead, false},
	}
	for _, tt := range tests {
		if got := tt.role.Can(tt.perm); got != tt.want {
			t.Errorf("%q.Can(%q) = %v, want %v", tt.role, tt.perm, got, tt.want)
		}
	}
}

func TestStaffRole_EveryRoleOpensTheOverview(t *testing.T) {
	for _, r := range StaffRoles {
		if !r.Can(PermOverviewRead) {
			t.Errorf("%q can't open the overview: it would land on a refusal after signing in", r)
		}
	}
}

func TestStaffRole_OnlySuperAdminManagesStaff(t *testing.T) {
	got := RolesWith(PermStaffManage)
	if len(got) != 1 || got[0] != StaffSuperAdmin {
		t.Errorf("RolesWith(staff.manage) = %v, want only the super admin", got)
	}
}

func TestStaffRole_PermissionsOfNonStaffIsEmpty(t *testing.T) {
	if got := StaffNone.Permissions(); len(got) != 0 {
		t.Errorf("Permissions() of a non-staff account = %v, want none", got)
	}
	if got := StaffSuperAdmin.Permissions(); len(got) != len(Permissions) {
		t.Errorf("super admin holds %d permissions, want all %d", len(got), len(Permissions))
	}
}
