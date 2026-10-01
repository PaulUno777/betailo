package httpserver

import (
	"net/http"
	"time"
)

// refreshCookieName is the one cookie this server sets. Path is /auth
// so it's never sent on, or settable by, any route except the two that
// read it — reducing both blast radius and the size of every other
// request.
const refreshCookieName = "refresh_token"
const refreshCookiePath = "/auth"

// CookieConfig is how the refresh cookie's Secure/SameSite attributes
// reach the handlers below, from COOKIE_SECURE/COOKIE_SAMESITE (see
// internal/app.cookieConfig) — environment-driven for the same reason
// CORS_ALLOWED_ORIGIN is: the frontend and backend are on different
// origins in production today (see docs/adr/0008), which needs
// SameSite=None, but that can flip to Lax with no code change once
// they're made genuinely same-site.
type CookieConfig struct {
	Secure   bool
	SameSite http.SameSite
}

// setRefreshCookie writes the rotated (or newly issued) refresh
// cookie. expiresAt is only honored when persistent is true — "remember
// me: false" gets a session cookie (no Expires/Max-Age at all), the
// same distinction the client used to make with sessionStorage vs
// localStorage.
func setRefreshCookie(w http.ResponseWriter, rawToken string, expiresAt time.Time, persistent bool, cfg CookieConfig) {
	cookie := &http.Cookie{
		Name:     refreshCookieName,
		Value:    rawToken,
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	}
	if persistent {
		cookie.Expires = expiresAt
	}
	http.SetCookie(w, cookie)
}

// clearRefreshCookie overwrites the cookie with an empty, already-
// expired one — the standard way to make a browser drop a cookie, used
// on logout and on any failed refresh.
func clearRefreshCookie(w http.ResponseWriter, cfg CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

// readRefreshCookie returns "" if the cookie is absent — not an error,
// since both handlers that call this treat a missing cookie as just
// another way to fail (unauthorized for refresh, a no-op for logout).
func readRefreshCookie(r *http.Request) string {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
