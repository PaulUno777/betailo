package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	me := decodeJSON[meResponse](t, meRec)
	if me.Email != "farmer@example.com" || me.Role != "farmer" || me.UserID != login.User.ID {
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
	// forge a token instead: tamper with the last character to prove a
	// modified signature is rejected.
	tampered := login.AccessToken[:len(login.AccessToken)-1] + "x"
	rec := doJSON(t, router, http.MethodGet, "/me", nil, tampered)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d for a tampered token", rec.Code, http.StatusUnauthorized)
	}
}
