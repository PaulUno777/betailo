import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, catchError, finalize, map, of, shareReplay } from 'rxjs';
import { AuthApi } from '../../data-access/auth/auth.api';
import { Role } from '../../domain/enums';
import { User } from '../../domain/models';

const USER_STORAGE_KEY = 'betailo.user';

/** Only ever the non-sensitive display user, never a token — see this
 * service's class doc for why the access token itself is never
 * persisted client-side. */
function readStoredUser(): User | null {
  try {
    const raw = localStorage.getItem(USER_STORAGE_KEY);
    return raw ? (JSON.parse(raw) as User) : null;
  } catch {
    return null;
  }
}

function persistUser(user: User | null): void {
  try {
    if (user) {
      localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(user));
    } else {
      localStorage.removeItem(USER_STORAGE_KEY);
    }
  } catch {
    // Storage can throw (private mode, quota) — losing the cached
    // display user is harmless, it's re-fetched on next login/refresh.
  }
}

/** The single source of truth for "who is signed in" — narrow, readonly
 * signals out (ISP: consumers read `isAuthenticated()`/`role()`, never a
 * god session object), one place that owns the token's lifecycle so
 * `authInterceptor`/`refreshInterceptor` and every guard agree with
 * each other.
 *
 * The access token lives ONLY in memory (never `localStorage` or
 * `sessionStorage`) — see docs/adr/0008: that's the actual security
 * win of moving to a refresh-token session, since an XSS payload can
 * no longer read it off the page. `user` (non-sensitive — no token, no
 * password) stays cached in `localStorage` purely so a reload can
 * render the signed-in UI instantly instead of flashing a logged-out
 * state while the silent refresh below is still in flight; the real
 * session is always re-established from the httpOnly refresh cookie,
 * via `refreshSession()`, called once at bootstrap (see
 * `app.config.ts`'s app initializer) and again by `refreshInterceptor`
 * on any 401. */
@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly authApi = inject(AuthApi);

  private readonly tokenSignal = signal<string | null>(null);
  private readonly userSignal = signal<User | null>(readStoredUser());

  /** Multiple concurrent 401s (or the bootstrap initializer racing a
   * route guard) share this one in-flight refresh instead of each
   * firing its own `/auth/refresh` call. Reset to null once the
   * request completes, so the next refresh cycle starts a fresh one. */
  private refreshInFlight: Observable<string | null> | null = null;

  readonly user = this.userSignal.asReadonly();
  readonly token = this.tokenSignal.asReadonly();
  readonly role = computed<Role | null>(() => this.userSignal()?.role ?? null);
  readonly isAuthenticated = computed(() => this.tokenSignal() !== null);

  /** Called after a successful login/register-then-login. The server
   * has already set the refresh cookie (see docs/adr/0008) — this just
   * records the access token and the display user. */
  setSession(accessToken: string, user: User): void {
    this.tokenSignal.set(accessToken);
    this.userSignal.set(user);
    persistUser(user);
  }

  /** Called by `refreshInterceptor`/the bootstrap initializer once a
   * silent refresh succeeds — only the access token changes; `user`
   * is left as whatever the last login or refresh already set. */
  private setAccessToken(accessToken: string): void {
    this.tokenSignal.set(accessToken);
  }

  private clearSession(): void {
    this.tokenSignal.set(null);
    this.userSignal.set(null);
    persistUser(null);
  }

  /** Single-flight silent refresh against the httpOnly cookie. Never
   * errors to its caller — a failed refresh (no cookie, expired,
   * revoked) resolves to `null` and clears local session state, which
   * is exactly what both call sites (the bootstrap initializer and
   * `refreshInterceptor`) want: "is there now a usable token, yes or
   * no," never a thrown error to handle. */
  refreshSession(): Observable<string | null> {
    if (!this.refreshInFlight) {
      this.refreshInFlight = this.authApi.refresh().pipe(
        map((res) => {
          this.setAccessToken(res.accessToken);
          return res.accessToken;
        }),
        catchError(() => {
          this.clearSession();
          return of(null);
        }),
        finalize(() => {
          this.refreshInFlight = null;
        }),
        shareReplay(1)
      );
    }
    return this.refreshInFlight;
  }

  /** Clears local state immediately (never block the UI on the
   * network) and best-effort revokes the server-side refresh-token
   * family. A failed logout call is not worth surfacing to the user —
   * they're signed out locally either way, and a stale cookie that
   * never got revoked is cleaned up by Mongo's TTL index regardless
   * (docs/adr/0008). */
  signOut(): void {
    this.clearSession();
    this.authApi
      .logout()
      .pipe(catchError(() => of(undefined)))
      .subscribe();
  }
}
