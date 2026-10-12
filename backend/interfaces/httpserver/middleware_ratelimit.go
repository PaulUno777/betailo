package httpserver

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimit is one token-bucket policy: Burst requests at once, refilled
// at PerMinute. The zero value disables limiting, which is what tests
// that hammer /auth/register from a single httptest address rely on.
type RateLimit struct {
	PerMinute float64
	Burst     int
}

func (l RateLimit) enabled() bool { return l.PerMinute > 0 && l.Burst > 0 }

// RateLimitConfig groups the per-route policies NewRouter applies to the
// credential endpoints — the ones worth brute-forcing (login), spamming
// (register), or hammering (refresh).
type RateLimitConfig struct {
	Login    RateLimit
	Register RateLimit
	Refresh  RateLimit
	// TrustProxyHeaders makes the client key the first X-Forwarded-For
	// hop instead of RemoteAddr. Only safe behind a proxy that
	// overwrites the header (Render does); otherwise any client could
	// pick its own key and dodge the limit.
	TrustProxyHeaders bool
}

// DefaultRateLimits are the production policies.
func DefaultRateLimits(trustProxyHeaders bool) RateLimitConfig {
	return RateLimitConfig{
		Login:             RateLimit{PerMinute: 10, Burst: 10},
		Register:          RateLimit{PerMinute: 5, Burst: 5},
		Refresh:           RateLimit{PerMinute: 30, Burst: 30},
		TrustProxyHeaders: trustProxyHeaders,
	}
}

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// rateLimiter is an in-memory per-key token bucket. It runs in a single
// process: a multi-instance deployment would need a shared store, but
// one API instance is all this deployment has. Idle buckets are swept
// opportunistically on the request path, so there's no background
// goroutine to leak.
type rateLimiter struct {
	policy RateLimit
	now    func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

const rateLimiterIdleTTL = 10 * time.Minute

func newRateLimiter(policy RateLimit, now func() time.Time) *rateLimiter {
	return &rateLimiter{policy: policy, now: now, buckets: map[string]*bucket{}}
}

// allow reports whether key may proceed and, if not, how long until one
// token is available.
func (l *rateLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	perSecond := l.policy.PerMinute / 60
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(l.policy.Burst), lastSeen: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(float64(l.policy.Burst), b.tokens+now.Sub(b.lastSeen).Seconds()*perSecond)
	b.lastSeen = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / perSecond * float64(time.Second))
	return false, wait
}

func (l *rateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < rateLimiterIdleTTL {
		return
	}
	for k, b := range l.buckets {
		if now.Sub(b.lastSeen) > rateLimiterIdleTTL {
			delete(l.buckets, k)
		}
	}
	l.lastSweep = now
}

// rateLimited wraps next with policy, keyed by client IP. A disabled
// policy returns next unchanged.
func rateLimited(policy RateLimit, trustProxy bool, now func() time.Time, next http.Handler) http.Handler {
	if !policy.enabled() {
		return next
	}
	limiter := newRateLimiter(policy, now)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, wait := limiter.allow(clientIP(r, trustProxy))
		if !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
			writeJSON(w, http.StatusTooManyRequests, errorResponse{Error: "too many requests"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			first, _, _ := strings.Cut(fwd, ",")
			if ip := strings.TrimSpace(first); ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
