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

func TestAccount_HTTP_BuyerBecomesSeller(t *testing.T) {
	router := newTestRouter()
	email := uniqueEmail("buyer")
	token, _ := registerAndLogin(t, router, email, "buyer")

	// A buyer can't create a farm directly...
	if rec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "X", Location: "Y"}, token); rec.Code != http.StatusForbidden {
		t.Fatalf("buyer POST /farms status = %d, want 403", rec.Code)
	}
	// ...but can become a seller, which creates the farm and switches the role.
	rec := doJSON(t, router, http.MethodPost, "/me/become-seller", becomeSellerRequest{Farm: farmRequest{Name: "Ferme Etoga", Location: "Douala", ContactPhone: "690123456"}}, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("become seller status = %d; body = %s", rec.Code, rec.Body.String())
	}
	if again := doJSON(t, router, http.MethodPost, "/me/become-seller", becomeSellerRequest{Farm: farmRequest{Name: "Bis", Location: "Douala"}}, token); again.Code != http.StatusConflict {
		t.Errorf("second become-seller status = %d, want 409", again.Code)
	}

	// A fresh sign-in carries the farmer role.
	login := decodeJSON[loginResponse](t, doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{Email: email, Password: "correct-horse-battery"}, ""))
	if login.User.Role != "farmer" {
		t.Fatalf("role after becoming a seller = %q, want farmer", login.User.Role)
	}
	if farms := decodeJSON[[]farmResponse](t, doJSON(t, router, http.MethodGet, "/farms/mine", nil, login.AccessToken)); len(farms) != 1 || farms[0].Name != "Ferme Etoga" {
		t.Errorf("farms/mine = %+v, want the new farm", farms)
	}
}
