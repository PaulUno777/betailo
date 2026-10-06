# 8. Refresh-token rotation in an httpOnly cookie

## Status

Accepted. Supersedes only the "no refresh tokens" part of ADR-0006 — see
that ADR's Consequences section for the forward link, and the note
below for exactly what else of it still stands.

## Context

ADR-0006 deliberately shipped Phase 1's auth without refresh tokens, a
session store, or MFA, calling them premature for the MVP's scope —
but it named its own escape hatch: *"worth revisiting if
`ACCESS_TOKEN_TTL` needs to shrink from 24h for that reason alone,
which would itself be a sign refresh tokens are now warranted."* That
trigger has now been pulled directly: the product asked for a session
model where a role or suspension change propagates faster than a
24-hour-old token allows, without asking every user to re-login that
often.

Two paths were on the table: adopt Keycloak (or another external IdP),
or extend the existing Go auth module. Keycloak would have meant a new
hosted service (Render has no managed offering, so it's a third
external dependency alongside MongoDB Atlas and Cloudflare R2), a
bcrypt-to-Keycloak user migration with no clean import path, and
replacing `ports.TokenSigner`/`PasswordHasher` and `middleware_auth.go`
wholesale with an OIDC client flow — disproportionate for this MVP's
scale and exactly what ADR-0002's anti-premature-abstraction stance
argues against. Extending the existing module costs one new Mongo
collection and a rotation method behind the same ports pattern
`Login`/`Register` already use, with no user migration and no new
infrastructure.

A second finding shaped the delivery mechanics: despite the "Concepts
& Architecture" document's diagram showing an nginx `/api` proxy in
front of the backend, **the actual deployment is cross-site in
production today**. `frontend/Dockerfile` serves the Angular build
from a stock `nginx:1.27-alpine` with no proxy configuration added, and
`API_BASE_URL` (`frontend/src/app/core/config/api.config.ts`) is
hardcoded to `http://localhost:8080` with no production override
anywhere in the Angular build config. The only thing letting the
deployed frontend reach the deployed backend at all is
`CORS_ALLOWED_ORIGIN` plus real cross-origin CORS handling in
`middleware_cors.go`. Whether to implement the documented proxy (making
the two genuinely same-site) or to fix `API_BASE_URL` for production
and update the architecture document to match reality is a separate,
not-yet-decided follow-up — this ADR's cookie is deliberately
configured so that decision doesn't block on it either way.

## Decision

- **Rotate-on-use, family-wide reuse revocation.** Every refresh token
  belongs to a `FamilyID` started at login. `POST /auth/refresh`
  revokes the token it was called with and issues a new one in the
  same family (`application/usecase/auth.RefreshToken`). Presenting an
  already-revoked token again — the signal that it leaked and is being
  replayed — revokes every token in that family, not just the one
  reused, via `ports.RefreshTokenRepository.RevokeFamily`. Rotation and
  reuse-detection are each unit-tested
  (`application/usecase/auth/refresh_test.go`).
- **Hybrid delivery: Bearer access token, httpOnly-cookie refresh
  token.** The access token stays exactly where ADR-0006 put it — a
  JSON response field, sent back as `Authorization: Bearer`. Only the
  new refresh token moves to an `HttpOnly` cookie
  (`interfaces/httpserver/cookie_config.go`), scoped to `Path=/auth` so
  it's never sent to, or settable on, any other route. This means
  every existing mutating route (`POST /farms`, `/reservations/{id}/pay`,
  etc.) stays exactly as CSRF-safe as it already was — a cross-site
  page can't read or forge a header it was never given — and only the
  two new endpoints need to reason about the cookie at all. A
  pure-cookie access token would have needed CSRF tokens sprayed across
  the entire mutating API surface for comparatively little gain here.
- **`SameSite=None; Secure`, both environment-driven.** Given the
  verified cross-site finding above, the cookie needs `SameSite=None`
  to be sent at all; `Secure` is required alongside it and costs
  nothing (it works over `http://localhost` in every modern browser
  too). Both are read from `COOKIE_SAMESITE`/`COOKIE_SECURE`
  (`internal/app.cookieConfig`) specifically so a future same-site
  deployment can flip to `Lax` with a config change, not a code change.
- **`Access-Control-Allow-Credentials: true`, unconditionally.**
  Added to `middleware_cors.go`. Safe without further conditions here
  because `CORS_ALLOWED_ORIGIN` is already always one exact configured
  origin, never `*` — the one precondition browsers require before
  they'll honor a credentialed cross-origin request at all.
- **"Remember me" lives on the server record, not just the cookie.**
  `ports.RefreshToken.Persistent` carries the login's `Remember` choice
  through every rotation in the family. The cookie's `Expires` isn't
  re-sent by the browser on the next request, so without this the bit
  would silently become "always persistent" after the first silent
  refresh. `Remember: false` gets a session-only cookie (no
  `Expires`/`Max-Age`) — the same distinction `auth.service.ts` used to
  make with `sessionStorage` vs `localStorage`, moved server-side.
- **Cleanup via Mongo's TTL index, not a worker.** `refresh_tokens`
  gets a `expiresAt` index with `expireAfterSeconds: 0`
  (`infrastructure/mongodb/bootstrap.go`) — Mongo's own background
  thread deletes expired documents, so no new entry in
  `internal/worker` was needed.
- **Access token stays short, refresh token stays long.**
  `ACCESS_TOKEN_TTL` default drops to 15 minutes; `REFRESH_TOKEN_TTL`
  defaults to 30 days. This is the actual payoff: a role or suspension
  change now takes effect within one access-token lifetime (≤15 min,
  via the fresh `UserRepository.FindByID` read in
  `RefreshToken.Execute`) instead of waiting out the old 24h token.
- **Frontend stops persisting the access token client-side at all.**
  `auth.service.ts` now keeps it only in an in-memory signal — the
  actual security win, since an XSS payload can no longer read it from
  `localStorage`/`sessionStorage`. A silent refresh at bootstrap
  (reading the httpOnly cookie the browser already has) restores the
  session on page load.

## Consequences

**Easier:** a suspended farmer or buyer, or a changed role, is locked
out within minutes rather than up to a day. A compromised access token
self-expires fast (15 min). Logout now actually revokes something
server-side, instead of just discarding a client-held token that
remains valid until it expires on its own.

**Harder:** the backend now owns a small amount of session state (one
Mongo collection) it didn't have before — a deliberate, bounded cost,
not the kind of stateful session store ADR-0006 was avoiding wholesale.
A reused/stolen refresh token now forces a real re-login for that
device, where before there was nothing to steal beyond the access
token itself (which self-expired in ≤24h regardless).

**Explicitly not doing:** Passkeys/WebAuthn and two-factor auth for
farmers (both separate future phases — `ports.RefreshTokenRepository`
and the hybrid cookie delivery here are designed not to need rework
when either lands); "sign out everywhere" (`RevokeAllForUser` doesn't
exist on the port yet because no caller does, consistent with
ADR-0002 — the natural next method once a password-change flow or an
account-security page exists); recording IP/user-agent per
refresh-token record for a future "active sessions" page (same
reasoning); and resolving the nginx-proxy-vs-hardcoded-`API_BASE_URL`
discrepancy in production (a separate, not-yet-decided follow-up this
ADR's env-driven `SameSite` is designed to outlive either way).

## Note on ADR-0006

Bcrypt password hashing, the single HMAC-SHA256 signing secret with no
default, self-registration restricted to farmer/buyer, and the
enumeration-safe "same error for unknown email or wrong password" all
remain exactly as ADR-0006 decided them — only its "no refresh tokens,
no session store" clause is superseded, by this ADR.
