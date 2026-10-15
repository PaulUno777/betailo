package httpserver

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestAllowedOrigins(t *testing.T) {
	cases := map[string][]string{
		"":                            {"http://localhost:4200"},
		"*":                           {"http://localhost:4200"},
		"https://betailo.vercel.app/": {"https://betailo.vercel.app"},
		"http://localhost:4200, http://192.168.1.13:4200": {"http://localhost:4200", "http://192.168.1.13:4200"},
	}
	for raw, want := range cases {
		if got := AllowedOrigins(raw); !reflect.DeepEqual(got, want) {
			t.Errorf("AllowedOrigins(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestWithCORS_EchoesOnlyListedOrigins(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGIN", "http://localhost:4200,http://192.168.1.13:4200")
	h := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))

	for origin, want := range map[string]string{
		"http://192.168.1.13:4200": "http://192.168.1.13:4200",
		"http://localhost:4200":    "http://localhost:4200",
		"https://evil.example":     "",
	} {
		req := httptest.NewRequest(http.MethodOptions, "/auth/register", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != want {
			t.Errorf("origin %s: Allow-Origin = %q, want %q", origin, got, want)
		}
		if want != "" && rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Errorf("origin %s: credentials not allowed", origin)
		}
	}
}
