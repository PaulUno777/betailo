# 6. Stateless JWT auth: bcrypt + HMAC, no session store, no refresh tokens

## Status

Accepted

## Context

Phase 1 of the roadmap ("Betailo — Plan de développement") calls for "a
real auth module: registration, login, JWT verified by middleware" —
until now `domain.User` existed but nothing issued or checked
credentials. The MVP has three roles (farmer, buyer, admin) and no
requirement yet for logout-everywhere, token revocation, or
short-lived-access + refresh-token rotation — those are the kind of
thing the post-MVP anti-fraud phase would add if the product needed it,
not something to build ahead of a real requirement.

## Decision

- **Passwords**: bcrypt (`infrastructure/security.BcryptHasher`), cost
  configurable but defaulting to `bcrypt.DefaultCost`. No pepper, no
  custom KDF — bcrypt alone is still the standard baseline for a
  password store at this scale.
- **Tokens**: a single HMAC-SHA256-signed JWT
  (`infrastructure/jwt.Signer`, `github.com/golang-jwt/jwt/v5`)
  carrying `UserID`, `Email`, `Role` and an expiry — no refresh token, no
  session store, no server-side revocation list. `RequireAuth`
  (`interfaces/httpserver`) verifies the signature and expiry and trusts
  the claims directly; it never queries MongoDB. One access token, valid
  until it expires (`ACCESS_TOKEN_TTL`, default 24h) — logout is "the
  client discards the token" and a role or verification-status change
  takes up to the token's remaining TTL to take effect for that user's
  existing token.
- **Self-registration is farmer/buyer only** — `RoleAdmin` is rejected
  by `application/usecase/auth.Register.Execute` before it ever reaches
  persistence. There's no admin-provisioning path in this codebase yet
  (see the roadmap for when one is needed); today, the only way to get
  an admin user is to insert one directly.
- **Login doesn't reveal which half of the credential pair was wrong**:
  an unknown email and a wrong password both return
  `domain.ErrUnauthorized`, mapped to the same 401 with the same body —
  standard practice against user enumeration, at essentially no
  implementation cost.
- **`Register.Execute` doesn't check-then-insert**: it inserts directly
  and relies on `infrastructure/mongodb.UserRepository.Create` turning
  the `users.email` unique-index violation into `domain.ErrEmailTaken`.
  A separate existence check would race two concurrent registrations
  for the same email — the same reasoning ADR-0001 already applied to
  `Cycle.availableQuantity`.
- **`JWT_SECRET` has no default and no fallback**: `internal/app.Initialize`
  fails startup outright if it's unset, rather than signing with an
  empty or hardcoded key. Every other environment variable in this
  codebase degrades to a sensible default when unset; this one
  deliberately doesn't, because a predictable signing key means anyone
  can forge a valid token for any user, including an admin.

## Consequences

**Easier:** verifying a token costs nothing beyond CPU (HMAC + JSON
decode) — no database round trip on every authenticated request, which
matters once `RequireAuth` sits in front of routes that get called far
more often than login itself.

**Harder:** revoking a specific user's access before their token expires
isn't possible without adding a mechanism this design doesn't have (a
deny-list, or moving to short-lived tokens + refresh rotation with a
server-side store). A role change or an admin disabling an account
takes effect only once that user's current token expires. This is an
accepted trade for the MVP's scope, not an oversight — worth revisiting
if `ACCESS_TOKEN_TTL` needs to shrink from 24h for that reason alone,
which would itself be a sign refresh tokens are now warranted.
**Update:** that trigger was pulled — see
[ADR-0008](0008-refresh-token-rotation-httponly-cookie.md), which adds
refresh-token rotation and shrinks `ACCESS_TOKEN_TTL`'s default to 15
minutes. Everything else in this ADR still stands.

**Explicitly not doing:** refresh tokens, a session/token store,
multi-factor auth, or a password-reset flow — none of Phase 1's DoD
requires them, and building them now would be exactly the premature
abstraction ADR-0002 already argues against elsewhere in this codebase.
