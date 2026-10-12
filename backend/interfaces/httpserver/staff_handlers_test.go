package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pauluno777/betailo/backend/domain"
)

// staffToken registers an account, gives it a staff role straight in
// the store and signs it in.
func staffToken(t *testing.T, router http.Handler, tf *testFakes, role domain.StaffRole) (token, userID string) {
	t.Helper()
	token, userID = registerAndLogin(t, router, uniqueEmail(string(role)), "buyer")
	tf.setStaffRole(userID, role)
	return token, userID
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	return decodeJSON[errorResponse](t, rec).Code
}

// Each staff role reaches what its job needs and is refused the rest,
// by the API, with the roles to ask named in the refusal.
func TestStaff_HTTP_EachRoleReachesOnlyItsOwnRoutes(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	moderator, _ := staffToken(t, router, tf, domain.StaffModerator)
	finance, _ := staffToken(t, router, tf, domain.StaffFinance)
	support, _ := staffToken(t, router, tf, domain.StaffSupport)
	_, targetID := registerAndLogin(t, router, uniqueEmail("member"), "buyer")

	tests := []struct {
		name   string
		token  string
		method string
		path   string
		body   any
		want   int
	}{
		{"moderator opens the overview", moderator, http.MethodGet, "/admin/overview", nil, http.StatusOK},
		{"finance opens the overview", finance, http.MethodGet, "/admin/overview", nil, http.StatusOK},
		{"support opens the overview", support, http.MethodGet, "/admin/overview", nil, http.StatusOK},
		{"support lists accounts", support, http.MethodGet, "/admin/users", nil, http.StatusOK},
		{"support lists farms", support, http.MethodGet, "/admin/farms?queue=todo", nil, http.StatusOK},
		{"support reads escrow", support, http.MethodGet, "/admin/escrow", nil, http.StatusOK},
		{"finance reads escrow", finance, http.MethodGet, "/admin/escrow", nil, http.StatusOK},
		{"moderator reads reports", moderator, http.MethodGet, "/admin/reports", nil, http.StatusOK},
		{"everyone reads the role table", support, http.MethodGet, "/admin/staff", nil, http.StatusOK},

		{"moderator can't read escrow", moderator, http.MethodGet, "/admin/escrow", nil, http.StatusForbidden},
		{"finance can't read reports", finance, http.MethodGet, "/admin/reports", nil, http.StatusForbidden},
		{"support can't suspend", support, http.MethodPost, "/admin/users/" + targetID + "/suspend", nil, http.StatusForbidden},
		{"finance can't suspend", finance, http.MethodPost, "/admin/users/" + targetID + "/suspend", nil, http.StatusForbidden},
		{"support can't raise a farm limit", support, http.MethodPost, "/admin/users/" + targetID + "/farm-limit", farmLimitRequest{Limit: 3, Reason: "x"}, http.StatusForbidden},
		{"finance can't review a farm", finance, http.MethodPost, "/admin/farms/" + targetID + "/review", reviewFarmRequest{Decision: "approve"}, http.StatusForbidden},
		{"support can't settle escrow", support, http.MethodPost, "/admin/escrow/" + targetID + "/release", nil, http.StatusForbidden},
		{"moderator can't grant a role", moderator, http.MethodPost, "/admin/staff", staffRoleRequest{Email: "x@example.com", Role: "support"}, http.StatusForbidden},
		{"moderator can't read the journal", moderator, http.MethodGet, "/admin/audit", nil, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doJSON(t, router, tt.method, tt.path, tt.body, tt.token)
			if rec.Code != tt.want {
				t.Fatalf("%s %s = %d, want %d; body = %s", tt.method, tt.path, rec.Code, tt.want, rec.Body.String())
			}
			if tt.want == http.StatusForbidden {
				out := decodeJSON[errorResponse](t, rec)
				if out.Code != "admin.permission_denied" || out.Params["roles"] == nil {
					t.Errorf("refusal = %+v, want admin.permission_denied naming the roles that may", out)
				}
			}
		})
	}

	// What a moderator may do, done: it lands in the journal under
	// their name and role.
	if rec := doJSON(t, router, http.MethodPost, "/admin/users/"+targetID+"/suspend", nil, moderator); rec.Code != http.StatusOK {
		t.Fatalf("moderator suspending an account = %d; body = %s", rec.Code, rec.Body.String())
	}
	last := tf.audit.actions[len(tf.audit.actions)-1]
	if last.Action != domain.ActionUserSuspend || last.ActorRole != domain.StaffModerator || last.TargetID.Hex() != targetID {
		t.Errorf("journal line = %+v, want the moderator's suspension of the account", last)
	}
	if rec := doJSON(t, router, http.MethodPost, "/admin/users/"+targetID+"/reactivate", nil, moderator); rec.Code != http.StatusOK || decodeJSON[adminUserResponse](t, rec).Suspended {
		t.Fatalf("reactivate = %d %s", rec.Code, rec.Body.String())
	}
}

// A super admin builds the team from existing accounts; the role
// applies to sessions that are already open, both ways.
func TestStaff_HTTP_GrantChangeAndRevokeARole(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	root := adminToken(t, router, tf)
	email := uniqueEmail("rita")
	memberToken, memberID := registerAndLogin(t, router, email, "buyer")

	if rec := doJSON(t, router, http.MethodGet, "/admin/overview", nil, memberToken); rec.Code != http.StatusForbidden {
		t.Fatalf("before any role, overview = %d, want 403", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodPost, "/admin/staff", staffRoleRequest{Email: email, Role: "janitor"}, root); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown role = %d, want 400", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodPost, "/admin/staff", staffRoleRequest{Email: "nobody@example.com", Role: "support"}, root); rec.Code != http.StatusNotFound || errorCode(t, rec) != "admin.account_not_found" {
		t.Fatalf("role for an unknown account = %d %s, want 404 admin.account_not_found", rec.Code, rec.Body.String())
	}

	grant := doJSON(t, router, http.MethodPost, "/admin/staff", staffRoleRequest{Email: email, Role: "support"}, root)
	if grant.Code != http.StatusOK || decodeJSON[adminUserResponse](t, grant).StaffRole != "support" {
		t.Fatalf("grant = %d %s", grant.Code, grant.Body.String())
	}
	// The token was signed before the role existed: it works at once.
	if rec := doJSON(t, router, http.MethodGet, "/admin/users", nil, memberToken); rec.Code != http.StatusOK {
		t.Fatalf("with the support role, listing accounts = %d, want 200", rec.Code)
	}
	me := decodeJSON[meResponse](t, doJSON(t, router, http.MethodGet, "/me", nil, memberToken))
	if me.StaffRole != "support" || len(me.Permissions) == 0 || me.CanAddFarm {
		t.Fatalf("GET /me for staff = role %q, permissions %v, canAddFarm %v", me.StaffRole, me.Permissions, me.CanAddFarm)
	}

	change := doJSON(t, router, http.MethodPost, "/admin/staff/"+memberID+"/role", staffRoleRequest{Role: "moderator"}, root)
	if change.Code != http.StatusOK || decodeJSON[adminUserResponse](t, change).StaffRole != "moderator" {
		t.Fatalf("change = %d %s", change.Code, change.Body.String())
	}
	staff := decodeJSON[staffResponse](t, doJSON(t, router, http.MethodGet, "/admin/staff", nil, root))
	if len(staff.Members) != 2 || len(staff.Roles) != len(domain.StaffRoles) || len(staff.Permissions) != len(domain.Permissions) {
		t.Fatalf("staff = %d members, %d roles, %d permissions", len(staff.Members), len(staff.Roles), len(staff.Permissions))
	}

	revoke := doJSON(t, router, http.MethodPost, "/admin/staff/"+memberID+"/role", staffRoleRequest{Role: ""}, root)
	if revoke.Code != http.StatusOK || decodeJSON[adminUserResponse](t, revoke).StaffRole != "" {
		t.Fatalf("revoke = %d %s", revoke.Code, revoke.Body.String())
	}
	// ...and the same token is locked out at once.
	if rec := doJSON(t, router, http.MethodGet, "/admin/users", nil, memberToken); rec.Code != http.StatusForbidden {
		t.Fatalf("after the role was revoked, listing accounts = %d, want 403", rec.Code)
	}
	if !tf.audit.has(domain.ActionStaffGrant) || !tf.audit.has(domain.ActionStaffRevoke) {
		t.Error("granting and revoking a role weren't both journaled")
	}
	journal := doJSON(t, router, http.MethodGet, "/admin/audit", nil, root)
	if journal.Code != http.StatusOK || len(decodeJSON[[]adminActionResponse](t, journal)) < 3 {
		t.Fatalf("journal = %d %s", journal.Code, journal.Body.String())
	}
}

// The guard rails around the team: the root account is untouchable,
// nobody edits themselves, there is always a super admin left, and a
// seller doesn't join the staff.
func TestStaff_HTTP_ProtectedAccounts(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	rootToken, rootID := registerAndLogin(t, router, uniqueEmail("root"), "buyer")
	tf.markRoot(rootID)
	otherToken, otherID := staffToken(t, router, tf, domain.StaffSuperAdmin)
	moderator, moderatorID := staffToken(t, router, tf, domain.StaffModerator)

	setRole := func(token, id, role string) *httptest.ResponseRecorder {
		return doJSON(t, router, http.MethodPost, "/admin/staff/"+id+"/role", staffRoleRequest{Role: role}, token)
	}
	if rec := setRole(otherToken, rootID, "support"); rec.Code != http.StatusForbidden || errorCode(t, rec) != "admin.root_protected" {
		t.Fatalf("demoting the root = %d %s, want 403 admin.root_protected", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, router, http.MethodPost, "/admin/users/"+rootID+"/suspend", nil, otherToken); rec.Code != http.StatusForbidden || errorCode(t, rec) != "admin.root_protected" {
		t.Fatalf("suspending the root = %d %s, want 403 admin.root_protected", rec.Code, rec.Body.String())
	}
	if rec := setRole(otherToken, otherID, ""); rec.Code != http.StatusForbidden || errorCode(t, rec) != "admin.self_action" {
		t.Fatalf("revoking one's own role = %d %s, want 403 admin.self_action", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, router, http.MethodPost, "/admin/users/"+otherID+"/suspend", nil, otherToken); rec.Code != http.StatusForbidden || errorCode(t, rec) != "admin.self_action" {
		t.Fatalf("suspending oneself = %d %s, want 403 admin.self_action", rec.Code, rec.Body.String())
	}
	// A moderator suspends members, not colleagues.
	if rec := doJSON(t, router, http.MethodPost, "/admin/users/"+otherID+"/suspend", nil, moderator); rec.Code != http.StatusForbidden || errorCode(t, rec) != "admin.staff_needs_super_admin" {
		t.Fatalf("moderator suspending a super admin = %d %s, want 403 admin.staff_needs_super_admin", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, router, http.MethodPost, "/admin/users/"+moderatorID+"/suspend", nil, rootToken); rec.Code != http.StatusOK {
		t.Fatalf("root suspending a moderator = %d; body = %s", rec.Code, rec.Body.String())
	}

	// A seller stays a seller.
	sellerToken, sellerID := registerAndLogin(t, router, uniqueEmail("seller"), "farmer")
	if rec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Douala"}, sellerToken); rec.Code != http.StatusCreated {
		t.Fatalf("create farm = %d", rec.Code)
	}
	if rec := setRole(rootToken, sellerID, "support"); rec.Code != http.StatusConflict || errorCode(t, rec) != "admin.staff_owns_farm" {
		t.Fatalf("a farm owner joining the staff = %d %s, want 409 admin.staff_owns_farm", rec.Code, rec.Body.String())
	}

	// With the root gone from the picture, the last super admin stays.
	solo, tfSolo := newTestRouterAndFakes()
	aToken, _ := staffToken(t, solo, tfSolo, domain.StaffSuperAdmin)
	_, bID := staffToken(t, solo, tfSolo, domain.StaffSuperAdmin)
	if rec := doJSON(t, solo, http.MethodPost, "/admin/staff/"+bID+"/role", staffRoleRequest{Role: "finance"}, aToken); rec.Code != http.StatusOK {
		t.Fatalf("demoting one of two super admins = %d; body = %s", rec.Code, rec.Body.String())
	}
	tfSolo.setStaffRole(bID, domain.StaffSuperAdmin)
	bToken := decodeJSON[loginResponse](t, doJSON(t, solo, http.MethodPost, "/auth/login", loginRequest{Email: emailForUserID(tfSolo, bID), Password: "correct-horse-battery"}, "")).AccessToken
	aID := ""
	for _, u := range tfSolo.users.byEmail {
		if u.StaffRole == domain.StaffSuperAdmin && u.ID.Hex() != bID {
			aID = u.ID.Hex()
		}
	}
	if rec := doJSON(t, solo, http.MethodPost, "/admin/staff/"+aID+"/role", staffRoleRequest{Role: ""}, bToken); rec.Code != http.StatusOK {
		t.Fatalf("revoking the other super admin = %d; body = %s", rec.Code, rec.Body.String())
	}
	// b is now alone; nobody else can demote them, and they can't
	// demote themselves (self_action), so the platform keeps one.
	if rec := doJSON(t, solo, http.MethodPost, "/admin/staff/"+bID+"/role", staffRoleRequest{Role: ""}, bToken); rec.Code != http.StatusForbidden {
		t.Fatalf("the last super admin revoking themselves = %d, want 403", rec.Code)
	}
}

// An exception to the farm limit is explained, attributed, never below
// what the account owns, and can be ended.
func TestStaff_HTTP_FarmLimitExceptions(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	tf.createFarm.MaxFarms = 1
	moderator, _ := staffToken(t, router, tf, domain.StaffModerator)
	email := uniqueEmail("coop")
	sellerToken, sellerID := registerAndLogin(t, router, email, "farmer")
	create := func(name string) int {
		return doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: name, Location: "Bafang"}, sellerToken).Code
	}
	if code := create("Site 1"); code != http.StatusCreated {
		t.Fatalf("first farm = %d", code)
	}

	raise := doJSON(t, router, http.MethodPost, "/admin/farm-limits", farmLimitRequest{Email: email, Limit: 3, Reason: "Coopérative, trois sites"}, moderator)
	if raise.Code != http.StatusOK {
		t.Fatalf("exception by email = %d; body = %s", raise.Code, raise.Body.String())
	}
	if got := decodeJSON[adminUserResponse](t, raise); got.FarmLimit != 3 || got.FarmLimitNote == nil || got.FarmLimitNote.Reason != "Coopérative, trois sites" || got.FarmCount != 1 {
		t.Fatalf("after the exception = %+v (note %+v)", got, got.FarmLimitNote)
	}
	if create("Site 2") != http.StatusCreated || create("Site 3") != http.StatusCreated {
		t.Fatal("the account should open farms up to its exception")
	}

	limits := decodeJSON[farmLimitsResponse](t, doJSON(t, router, http.MethodGet, "/admin/farm-limits", nil, moderator))
	if limits.Default != 1 || len(limits.Exceptions) != 1 || limits.Exceptions[0].FarmLimitNote.SetByRole != "moderator" || limits.Exceptions[0].FarmCount != 3 {
		t.Fatalf("limits = %+v", limits)
	}

	lower := doJSON(t, router, http.MethodPost, "/admin/users/"+sellerID+"/farm-limit", farmLimitRequest{Limit: 2, Reason: "Erreur"}, moderator)
	if lower.Code != http.StatusConflict || errorCode(t, lower) != "admin.farm_limit_below_owned" {
		t.Fatalf("limit below the farms owned = %d %s, want 409 admin.farm_limit_below_owned", lower.Code, lower.Body.String())
	}
	// Ending the exception needs no reason and removes no farm.
	reset := doJSON(t, router, http.MethodPost, "/admin/users/"+sellerID+"/farm-limit", farmLimitRequest{Limit: 0}, moderator)
	if reset.Code != http.StatusOK || decodeJSON[adminUserResponse](t, reset).FarmLimitNote != nil {
		t.Fatalf("ending the exception = %d %s", reset.Code, reset.Body.String())
	}
	if farms := decodeJSON[[]farmResponse](t, doJSON(t, router, http.MethodGet, "/farms/mine", nil, sellerToken)); len(farms) != 3 {
		t.Errorf("farms after the exception ended = %d, want the 3 already opened", len(farms))
	}
	if create("Site 4") != http.StatusConflict {
		t.Error("back on the default, a fourth farm should be refused")
	}
}
