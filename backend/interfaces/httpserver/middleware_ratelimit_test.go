package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeNow struct{ t time.Time }

func (f *fakeNow) now() time.Time { return f.t }

func TestRateLimiter_BurstThenRefill(t *testing.T) {
	clk := &fakeNow{t: time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)}
	l := newRateLimiter(RateLimit{PerMinute: 60, Burst: 3}, clk.now)

	for i := range 3 {
		if ok, _ := l.allow("1.2.3.4"); !ok {
			t.Fatalf("request %d within burst was refused", i+1)
		}
	}
	ok, wait := l.allow("1.2.3.4")
	if ok {
		t.Fatal("request past the burst was allowed")
	}
	if wait <= 0 || wait > time.Second {
		t.Fatalf("wait = %v, want (0, 1s] at 1 token/s", wait)
	}
	if ok, _ := l.allow("5.6.7.8"); !ok {
		t.Fatal("a different client shares the first client's bucket")
	}

	clk.t = clk.t.Add(time.Second)
	if ok, _ := l.allow("1.2.3.4"); !ok {
		t.Fatal("one token should have refilled after one second")
	}
}

func TestRateLimiter_SweepsIdleBuckets(t *testing.T) {
	clk := &fakeNow{t: time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)}
	l := newRateLimiter(RateLimit{PerMinute: 60, Burst: 1}, clk.now)
	l.allow("1.2.3.4")

	clk.t = clk.t.Add(rateLimiterIdleTTL + time.Minute)
	l.allow("5.6.7.8")
	if _, ok := l.buckets["1.2.3.4"]; ok {
		t.Fatal("idle bucket should have been swept")
	}
}

func TestRateLimited_Returns429WithRetryAfter(t *testing.T) {
	clk := &fakeNow{t: time.Now()}
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := rateLimited(RateLimit{PerMinute: 1, Burst: 1}, false, clk.now, ok)

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/auth/login", nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first request status = %d, want %d", first.Code, http.StatusNoContent)
	}

	second := httptest.NewRecorder()
	h.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/auth/login", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if got := second.Header().Get("Retry-After"); got != "60" {
		t.Fatalf("Retry-After = %q, want %q", got, "60")
	}
}

func TestRateLimited_ZeroPolicyIsDisabled(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := rateLimited(RateLimit{}, false, time.Now, ok)
	for i := range 100 {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/auth/register", nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("request %d status = %d, want %d", i+1, rec.Code, http.StatusNoContent)
		}
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:5555"
	r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")

	if got := clientIP(r, false); got != "10.0.0.1" {
		t.Errorf("untrusted proxy: clientIP = %q, want RemoteAddr host", got)
	}
	if got := clientIP(r, true); got != "203.0.113.7" {
		t.Errorf("trusted proxy: clientIP = %q, want first X-Forwarded-For hop", got)
	}
}

func TestReadJSON_OversizedBodyIs413(t *testing.T) {
	router := newTestRouter()
	body := `{"email":"` + strings.Repeat("a", maxRequestBodyBytes) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestReadJSON_MalformedBodyIs400(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
