package httpserver

import (
	"net/http"
	"testing"
)

func TestAccount_HTTP_ProfileEdit(t *testing.T) {
	router := newTestRouter()
	token, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	rec := doJSON(t, router, http.MethodPatch, "/me", updateProfileRequest{DisplayName: "  Aïcha Etoga ", Phone: "699 12 34 56", City: "Douala"}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /me status = %d; body = %s", rec.Code, rec.Body.String())
	}
	p := decodeJSON[profileResponse](t, rec)
	if p.DisplayName != "Aïcha Etoga" || p.Phone != "+237699123456" || p.City != "Douala" {
		t.Errorf("profile = %+v, want trimmed name, E.164 phone and city", p)
	}
	if got := decodeJSON[profileResponse](t, doJSON(t, router, http.MethodGet, "/me", nil, token)); got.DisplayName != "Aïcha Etoga" {
		t.Errorf("GET /me after edit = %+v", got)
	}
	for _, bad := range []updateProfileRequest{{DisplayName: ""}, {DisplayName: "Ok", Phone: "123"}} {
		if rec := doJSON(t, router, http.MethodPatch, "/me", bad, token); rec.Code != http.StatusBadRequest {
			t.Errorf("PATCH /me %+v status = %d, want 400", bad, rec.Code)
		}
	}
}

func TestAccount_HTTP_ChangePasswordEndsOtherSessionsButKeepsThisOne(t *testing.T) {
	router := newTestRouter()
	email := uniqueEmail("buyer")
	token, _ := registerAndLogin(t, router, email, "buyer")
	// A second device's session, via its refresh cookie.
	other := doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{Email: email, Password: "correct-horse-battery"}, "")
	otherCookie := refreshCookieFrom(t, other)

	if wrong := doJSON(t, router, http.MethodPost, "/me/password", changePasswordRequest{CurrentPassword: "nope-nope-nope", NewPassword: "brand-new-secret"}, token); wrong.Code != http.StatusUnauthorized {
		t.Fatalf("wrong current password status = %d, want 401", wrong.Code)
	}
	rec := doJSON(t, router, http.MethodPost, "/me/password", changePasswordRequest{CurrentPassword: "correct-horse-battery", NewPassword: "brand-new-secret"}, token)
	if rec.Code != http.StatusOK || decodeJSON[loginResponse](t, rec).AccessToken == "" {
		t.Fatalf("change password status = %d; body = %s", rec.Code, rec.Body.String())
	}
	if refresh := doJSONWithCookie(t, router, http.MethodPost, "/auth/refresh", otherCookie); refresh.Code != http.StatusUnauthorized {
		t.Errorf("the other device's session survived the password change: status = %d", refresh.Code)
	}
	if old := doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{Email: email, Password: "correct-horse-battery"}, ""); old.Code != http.StatusUnauthorized {
		t.Errorf("old password still works: status = %d", old.Code)
	}
}
