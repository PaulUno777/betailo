package domain

// Permission is one thing a member of the staff may do in the back
// office. Routes ask for a permission, never for a role: the table
// below is the only place a role's reach is written down.
type Permission string

const (
	// PermOverviewRead opens the back office's landing page.
	PermOverviewRead Permission = "overview.read"
	// PermAccountsRead lists accounts.
	PermAccountsRead Permission = "accounts.read"
	// PermAccountsSuspend suspends an account.
	PermAccountsSuspend Permission = "accounts.suspend"
	// PermFarmLimitSet raises (or resets) one account's farm limit.
	PermFarmLimitSet Permission = "accounts.farm_limit"
	// PermFarmsRead opens the farm review queue, proof included.
	PermFarmsRead Permission = "farms.read"
	// PermFarmsReview approves, rejects or suspends a farm.
	PermFarmsReview Permission = "farms.review"
	// PermCatalogRead lists every cycle and listing.
	PermCatalogRead Permission = "catalog.read"
	// PermReportsRead opens reports and disputes.
	PermReportsRead Permission = "reports.read"
	// PermReportsResolve acts on a report.
	PermReportsResolve Permission = "reports.resolve"
	// PermEscrowRead lists payments held by the platform.
	PermEscrowRead Permission = "escrow.read"
	// PermEscrowSettle releases or refunds a held payment.
	PermEscrowSettle Permission = "escrow.settle"
	// PermExportsAccounting downloads the accounting exports.
	PermExportsAccounting Permission = "exports.accounting"
	// PermStaffManage grants, changes and revokes staff roles.
	PermStaffManage Permission = "staff.manage"
	// PermAuditRead reads the journal of staff actions.
	PermAuditRead Permission = "audit.read"
)

// Permissions lists every permission in the order the back office
// shows them.
var Permissions = []Permission{
	PermOverviewRead, PermAccountsRead, PermFarmsRead, PermCatalogRead,
	PermFarmsReview, PermFarmLimitSet, PermReportsRead, PermReportsResolve, PermAccountsSuspend,
	PermEscrowRead, PermEscrowSettle, PermExportsAccounting, PermStaffManage, PermAuditRead,
}

// StaffRoles lists the roles staff can hold, most powerful first.
var StaffRoles = []StaffRole{StaffSuperAdmin, StaffModerator, StaffFinance, StaffSupport}

// rolePermissions is who may do what. A super admin may do everything;
// the three others are cut along the platform's jobs: moderators keep
// the marketplace honest, finance moves the money, support reads so it
// can answer people.
var rolePermissions = map[StaffRole]map[Permission]bool{
	StaffModerator: set(
		PermOverviewRead, PermAccountsRead, PermFarmsRead, PermCatalogRead,
		PermFarmsReview, PermFarmLimitSet, PermReportsRead, PermReportsResolve, PermAccountsSuspend,
	),
	StaffFinance: set(
		PermOverviewRead, PermAccountsRead, PermFarmsRead, PermCatalogRead,
		PermEscrowRead, PermEscrowSettle, PermExportsAccounting,
	),
	StaffSupport: set(
		PermOverviewRead, PermAccountsRead, PermFarmsRead, PermCatalogRead,
		PermReportsRead, PermEscrowRead,
	),
}

func set(perms ...Permission) map[Permission]bool {
	out := make(map[Permission]bool, len(perms))
	for _, p := range perms {
		out[p] = true
	}
	return out
}

// IsValid reports whether r is a role staff can hold.
func (r StaffRole) IsValid() bool {
	for _, known := range StaffRoles {
		if r == known {
			return true
		}
	}
	return false
}

// Can reports whether the role holds the permission.
func (r StaffRole) Can(p Permission) bool {
	if r == StaffSuperAdmin {
		return true
	}
	return rolePermissions[r][p]
}

// Permissions returns what the role holds, in display order; nil for
// an account that isn't staff.
func (r StaffRole) Permissions() []Permission {
	if !r.IsStaff() {
		return nil
	}
	out := make([]Permission, 0, len(Permissions))
	for _, p := range Permissions {
		if r.Can(p) {
			out = append(out, p)
		}
	}
	return out
}

// RolesWith returns the roles holding the permission, most powerful
// first: what the back office names when it refuses an action.
func RolesWith(p Permission) []StaffRole {
	var out []StaffRole
	for _, r := range StaffRoles {
		if r.Can(p) {
			out = append(out, r)
		}
	}
	return out
}
