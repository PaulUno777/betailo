package httpserver

import (
	"fmt"
	"net/http"
	"testing"
)

// registerAndLogin registers a new user with role and returns their
// access token and user ID — a shared setup step for every test in
// this package that needs an authenticated caller.
func registerAndLogin(t *testing.T, router http.Handler, email, role string) (token, userID string) {
	t.Helper()

	regRec := doJSON(t, router, http.MethodPost, "/auth/register", registerRequest{
		Email: email, Password: "correct-horse-battery", Role: role,
	}, "")
	if regRec.Code != http.StatusCreated {
		t.Fatalf("register(%s) status = %d, want %d; body = %s", email, regRec.Code, http.StatusCreated, regRec.Body.String())
	}
	user := decodeJSON[userResponse](t, regRec)

	loginRec := doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{
		Email: email, Password: "correct-horse-battery",
	}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login(%s) status = %d, want %d; body = %s", email, loginRec.Code, http.StatusOK, loginRec.Body.String())
	}
	login := decodeJSON[loginResponse](t, loginRec)

	return login.AccessToken, user.ID
}

// uniqueEmail avoids collisions between subtests sharing one router's
// in-memory user store.
var emailCounter int

func uniqueEmail(prefix string) string {
	emailCounter++
	return fmt.Sprintf("%s-%d@example.com", prefix, emailCounter)
}
