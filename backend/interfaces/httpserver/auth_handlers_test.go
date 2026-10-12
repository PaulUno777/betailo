package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func doJSON(t *testing.T, router http.Handler, method, path string, body any, bearer string) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", rec.Body.String(), err)
	}
	return out
}

// doJSONWithCookie is doJSON for the two endpoints (/auth/refresh,
// /auth/logout) that authenticate via the refresh cookie instead of a
// bearer token.
func doJSONWithCookie(t *testing.T, router http.Handler, method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBuffer(nil))
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// refreshCookieFrom finds the refresh_token cookie among a response's
// Set-Cookie headers, failing the test if there isn't exactly one.
func refreshCookieFrom(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			return c
		}
	}
	t.Fatalf("no %q cookie in response (Set-Cookie headers: %v)", refreshCookieName, rec.Result().Header.Values("Set-Cookie"))
	return nil
}

func TestRegister_HTTP_Success(t *testing.T) {
	router := newTestRouter()

	rec := doJSON(t, router, http.MethodPost, "/auth/register", registerRequest{
		Email:    "farmer@example.com",
		Password: "correct-horse-battery",
		Role:     "farmer",
	}, "")

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	got := decodeJSON[userResponse](t, rec)
	if got.Email != "farmer@example.com" || got.Role != "farmer" || got.ID == "" {
		t.Errorf("body = %+v, want a populated farmer user", got)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("correct-horse-battery")) {
		t.Error("response body leaks the plaintext password")
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("passwordHash")) {
		t.Error("response body leaks the password hash field")
	}
}

func TestRegister_HTTP_DuplicateEmail(t *testing.T) {
	router := newTestRouter()
	in := registerRequest{Email: "buyer@example.com", Password: "correct-horse-battery", Role: "buyer"}

	first := doJSON(t, router, http.MethodPost, "/auth/register", in, "")
	if first.Code != http.StatusCreated {
		t.Fatalf("first registration status = %d, want %d", first.Code, http.StatusCreated)
	}

	second := doJSON(t, router, http.MethodPost, "/auth/register", in, "")
	if second.Code != http.StatusConflict {
		t.Fatalf("second registration status = %d, want %d; body = %s", second.Code, http.StatusConflict, second.Body.String())
	}
}

func TestRegister_HTTP_InvalidBody(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString("{not json"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRegister_HTTP_RejectsAdminRole(t *testing.T) {
	router := newTestRouter()

	rec := doJSON(t, router, http.MethodPost, "/auth/register", registerRequest{
		Email: "wannabe@example.com", Password: "correct-horse-battery", Role: "admin",
	}, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d — self-registering as admin must be rejected", rec.Code, http.StatusBadRequest)
	}
}

func TestLoginAndMe_HTTP_FullFlow(t *testing.T) {
	router := newTestRouter()

	reg := doJSON(t, router, http.MethodPost, "/auth/register", registerRequest{
		Email: "farmer@example.com", Password: "correct-horse-battery", Role: "farmer",
	}, "")
	if reg.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d", reg.Code, http.StatusCreated)
	}

	loginRec := doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{
		Email: "farmer@example.com", Password: "correct-horse-battery",
	}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body = %s", loginRec.Code, http.StatusOK, loginRec.Body.String())
	}
	login := decodeJSON[loginResponse](t, loginRec)
	if login.AccessToken == "" {
		t.Fatal("login response has no accessToken")
	}
	if login.User.Email != "farmer@example.com" || login.User.Role != "farmer" {
		t.Errorf("login.User = %+v, want the registered farmer", login.User)
	}

	// The token from login must work as a bearer token on a protected
	// route — this is the middleware end-to-end.
	meRec := doJSON(t, router, http.MethodGet, "/me", nil, login.AccessToken)
	if meRec.Code != http.StatusOK {
		t.Fatalf("GET /me status = %d, want %d; body = %s", meRec.Code, http.StatusOK, meRec.Body.String())
	}
	me := decodeJSON[profileResponse](t, meRec)
	if me.Email != "farmer@example.com" || me.Role != "farmer" || me.ID != login.User.ID {
		t.Errorf("GET /me = %+v, want it to describe the logged-in farmer", me)
	}
}

func TestLogin_HTTP_WrongPassword(t *testing.T) {
	router := newTestRouter()
	doJSON(t, router, http.MethodPost, "/auth/register", registerRequest{
		Email: "a@example.com", Password: "correct-horse-battery", Role: "buyer",
	}, "")

	rec := doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{
		Email: "a@example.com", Password: "wrong-password",
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLogin_HTTP_UnknownEmail(t *testing.T) {
	router := newTestRouter()

	rec := doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{
		Email: "nobody@example.com", Password: "correct-horse-battery",
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (never reveal the account doesn't exist)", rec.Code, http.StatusUnauthorized)
	}
}

func TestMe_HTTP_NoToken(t *testing.T) {
	router := newTestRouter()

	rec := doJSON(t, router, http.MethodGet, "/me", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMe_HTTP_MalformedToken(t *testing.T) {
	router := newTestRouter()

	rec := doJSON(t, router, http.MethodGet, "/me", nil, "not-a-real-token")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMe_HTTP_MissingBearerScheme(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "some-token-without-bearer-prefix")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMe_HTTP_TokenFromDifferentSecretRejected(t *testing.T) {
	router := newTestRouter()
	doJSON(t, router, http.MethodPost, "/auth/register", registerRequest{
		Email: "a@example.com", Password: "correct-horse-battery", Role: "buyer",
	}, "")
	loginRec := doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{
		Email: "a@example.com", Password: "correct-horse-battery",
	}, "")
	login := decodeJSON[loginResponse](t, loginRec)

	// A second router, built with its own newTestRouter() call, uses a
	// signer with the same hardcoded test secret in this suite, so
	// forge a token instead: change the signature's first character to
	// prove a modified signature is rejected. Not the last one: in a
	// 43-char base64url HS256 signature it carries two padding bits the
	// decoder ignores, so some substitutions decode to the same bytes.
	sigStart := strings.LastIndex(login.AccessToken, ".") + 1
	swap := byte('A')
	if login.AccessToken[sigStart] == swap {
		swap = 'B'
	}
	tampered := login.AccessToken[:sigStart] + string(swap) + login.AccessToken[sigStart+1:]
	rec := doJSON(t, router, http.MethodGet, "/me", nil, tampered)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d for a tampered token", rec.Code, http.StatusUnauthorized)
	}
}

func TestLogin_HTTP_SetsRefreshCookie(t *testing.T) {
	router := newTestRouter()
	doJSON(t, router, http.MethodPost, "/auth/register", registerRequest{
		Email: "farmer@example.com", Password: "correct-horse-battery", Role: "farmer",
	}, "")

	rec := doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{
		Email: "farmer@example.com", Password: "correct-horse-battery", Remember: true,
	}, "")
	cookie := refreshCookieFrom(t, rec)

	if cookie.Value == "" {
		t.Error("cookie value is empty, want the raw refresh secret")
	}
	if !cookie.HttpOnly {
		t.Error("HttpOnly = false, want true")
	}
	if cookie.SameSite != http.SameSiteNoneMode {
		t.Errorf("SameSite = %v, want %v (newTestRouter wires CookieConfig with None)", cookie.SameSite, http.SameSiteNoneMode)
	}
	if cookie.Path != refreshCookiePath {
		t.Errorf("Path = %q, want %q", cookie.Path, refreshCookiePath)
	}
	if cookie.Expires.IsZero() {
		t.Error("Expires is zero, want it set (Remember: true)")
	}

	// The access token in the JSON body is unaffected by any of this.
	login := decodeJSON[loginResponse](t, rec)
	if login.AccessToken == "" {
		t.Error("login response has no accessToken")
	}
}

func TestLogin_HTTP_NotRemembered_SessionCookie(t *testing.T) {
	router := newTestRouter()
	doJSON(t, router, http.MethodPost, "/auth/register", registerRequest{
		Email: "buyer@example.com", Password: "correct-horse-battery", Role: "buyer",
	}, "")

	rec := doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{
		Email: "buyer@example.com", Password: "correct-horse-battery", Remember: false,
	}, "")
	cookie := refreshCookieFrom(t, rec)

	if !cookie.Expires.IsZero() {
		t.Errorf("Expires = %v, want zero (Remember: false should be a session-only cookie)", cookie.Expires)
	}
}

// TestRefreshLogoutFlow_HTTP exercises the full lifecycle through the
// real router: login, rotate via refresh, detect reuse of the
// now-stale cookie (and confirm it kills the token refresh issued
// too), then logout.
func TestRefreshLogoutFlow_HTTP(t *testing.T) {
	router := newTestRouter()
	doJSON(t, router, http.MethodPost, "/auth/register", registerRequest{
		Email: "farmer@example.com", Password: "correct-horse-battery", Role: "farmer",
	}, "")
	loginRec := doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{
		Email: "farmer@example.com", Password: "correct-horse-battery", Remember: true,
	}, "")
	cookie1 := refreshCookieFrom(t, loginRec)

	// Rotate.
	refreshRec := doJSONWithCookie(t, router, http.MethodPost, "/auth/refresh", cookie1)
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want %d; body = %s", refreshRec.Code, http.StatusOK, refreshRec.Body.String())
	}
	refreshed := decodeJSON[refreshResponse](t, refreshRec)
	if refreshed.AccessToken == "" {
		t.Fatal("refresh response has no accessToken")
	}
	cookie2 := refreshCookieFrom(t, refreshRec)
	if cookie2.Value == cookie1.Value {
		t.Fatal("refresh did not rotate the cookie value")
	}

	// The rotated-away cookie1 is now dead.
	reuseRec := doJSONWithCookie(t, router, http.MethodPost, "/auth/refresh", cookie1)
	if reuseRec.Code != http.StatusUnauthorized {
		t.Fatalf("reuse status = %d, want %d", reuseRec.Code, http.StatusUnauthorized)
	}

	// And reuse detection took the whole family down: cookie2, never
	// itself reused, is now also dead.
	afterReuseRec := doJSONWithCookie(t, router, http.MethodPost, "/auth/refresh", cookie2)
	if afterReuseRec.Code != http.StatusUnauthorized {
		t.Fatalf("post-reuse status with the other valid cookie = %d, want %d", afterReuseRec.Code, http.StatusUnauthorized)
	}

	// Fresh login, then logout — the cookie it issued must stop
	// working for refresh afterward.
	loginRec2 := doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{
		Email: "farmer@example.com", Password: "correct-horse-battery", Remember: true,
	}, "")
	cookie3 := refreshCookieFrom(t, loginRec2)

	logoutRec := doJSONWithCookie(t, router, http.MethodPost, "/auth/logout", cookie3)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want %d", logoutRec.Code, http.StatusNoContent)
	}

	postLogoutRec := doJSONWithCookie(t, router, http.MethodPost, "/auth/refresh", cookie3)
	if postLogoutRec.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout status = %d, want %d", postLogoutRec.Code, http.StatusUnauthorized)
	}
}

func TestRefresh_HTTP_NoCookie(t *testing.T) {
	router := newTestRouter()
	rec := doJSONWithCookie(t, router, http.MethodPost, "/auth/refresh", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLogout_HTTP_NoCookie(t *testing.T) {
	router := newTestRouter()
	rec := doJSONWithCookie(t, router, http.MethodPost, "/auth/logout", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d (logout with no cookie is a no-op, not an error)", rec.Code, http.StatusNoContent)
	}
}
