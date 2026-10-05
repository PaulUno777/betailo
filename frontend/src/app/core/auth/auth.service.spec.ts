import { TestBed } from '@angular/core/testing';
import { Observable, Subject, of, throwError } from 'rxjs';
import { User } from '../../domain/models';
import { AuthApi, LoginResult, RefreshResult } from '../../data-access/auth/auth.api';
import { AuthService } from './auth.service';

const user: User = { id: 'u1', email: 'buyer@betailo.test', role: 'buyer' };

/** A minimal AuthApi double — AuthService never needs register()/me(),
 * and refresh()/logout() are each scripted per test rather than making
 * a real HTTP call. */
class FakeAuthApi {
  refreshCalls = 0;
  refreshResult: Observable<RefreshResult> = of({ accessToken: 'refreshed-token' });
  logoutCalls = 0;

  login(): Observable<LoginResult> {
    throw new Error('not used by these tests');
  }

  refresh(): Observable<RefreshResult> {
    this.refreshCalls++;
    return this.refreshResult;
  }

  logout(): Observable<void> {
    this.logoutCalls++;
    return of(undefined);
  }
}

describe('AuthService', () => {
  let fakeAuthApi: FakeAuthApi;

  beforeEach(() => {
    localStorage.clear();
    fakeAuthApi = new FakeAuthApi();
    TestBed.configureTestingModule({ providers: [{ provide: AuthApi, useValue: fakeAuthApi }] });
  });

  it('starts signed out when nothing is stored', () => {
    const auth = TestBed.inject(AuthService);
    expect(auth.isAuthenticated()).toBeFalse();
    expect(auth.user()).toBeNull();
    expect(auth.role()).toBeNull();
    expect(auth.token()).toBeNull();
  });

  it('exposes the session after setSession and persists only the user, never the token', () => {
    const auth = TestBed.inject(AuthService);
    auth.setSession('token-123', user);

    expect(auth.isAuthenticated()).toBeTrue();
    expect(auth.token()).toBe('token-123');
    expect(auth.role()).toBe('buyer');
    expect(JSON.parse(localStorage.getItem('betailo.user')!)).toEqual(user);
    // The actual security property this phase exists for: the access
    // token is never written to any browser storage an XSS payload
    // could read.
    expect(localStorage.getItem('betailo.user')).not.toContain('token-123');
    expect(sessionStorage.length).toBe(0);
  });

  it('does not restore the access token on a fresh instance — only the cached display user', () => {
    TestBed.inject(AuthService).setSession('token-456', user);

    TestBed.resetTestingModule();
    TestBed.configureTestingModule({ providers: [{ provide: AuthApi, useValue: fakeAuthApi }] });
    const restored = TestBed.inject(AuthService);

    expect(restored.token()).toBeNull();
    expect(restored.isAuthenticated()).toBeFalse();
    expect(restored.user()).toEqual(user);
  });

  it('clears local state and calls the server logout on signOut', () => {
    const auth = TestBed.inject(AuthService);
    auth.setSession('token-789', user);
    auth.signOut();

    expect(auth.isAuthenticated()).toBeFalse();
    expect(auth.user()).toBeNull();
    expect(localStorage.getItem('betailo.user')).toBeNull();
    expect(fakeAuthApi.logoutCalls).toBe(1);
  });

  it('signOut clears local state even if the server logout call fails', () => {
    fakeAuthApi.logout = () => {
      fakeAuthApi.logoutCalls++;
      return throwError(() => new Error('network error'));
    };
    const auth = TestBed.inject(AuthService);
    auth.setSession('token-abc', user);
    auth.signOut();

    expect(auth.isAuthenticated()).toBeFalse();
  });

  describe('refreshSession', () => {
    it('sets a new access token on success', (done) => {
      fakeAuthApi.refreshResult = of({ accessToken: 'fresh-token' });
      const auth = TestBed.inject(AuthService);

      auth.refreshSession().subscribe((token) => {
        expect(token).toBe('fresh-token');
        expect(auth.token()).toBe('fresh-token');
        expect(auth.isAuthenticated()).toBeTrue();
        done();
      });
    });

    it('clears the session and resolves to null on failure', (done) => {
      fakeAuthApi.refreshResult = throwError(() => new Error('expired'));
      const auth = TestBed.inject(AuthService);
      auth.setSession('stale-token', user);

      auth.refreshSession().subscribe((token) => {
        expect(token).toBeNull();
        expect(auth.isAuthenticated()).toBeFalse();
        expect(auth.user()).toBeNull();
        done();
      });
    });

    it('shares one in-flight call across concurrent callers (single-flight)', (done) => {
      // A Subject, not `of(...)`, so both calls below genuinely overlap
      // before either resolves — matching a real network call, which
      // never resolves synchronously within the same call that issued
      // it. `of(...)` would make both calls fully resolve (including
      // AuthService's own cleanup) before the second one even starts,
      // defeating the point of this test.
      const pending = new Subject<RefreshResult>();
      fakeAuthApi.refreshResult = pending.asObservable();
      const auth = TestBed.inject(AuthService);

      let completed = 0;
      const onDone = () => {
        completed++;
        if (completed === 2) {
          expect(fakeAuthApi.refreshCalls).toBe(1);
          done();
        }
      };

      auth.refreshSession().subscribe(onDone);
      auth.refreshSession().subscribe(onDone);

      pending.next({ accessToken: 'new-token' });
      pending.complete();
    });

    it('starts a fresh call on the next refresh cycle', (done) => {
      const auth = TestBed.inject(AuthService);
      // Not nested inside the first subscribe's callback: RxJS delivers
      // `next` before `complete` (and finalize runs on complete), so a
      // call made from inside the first `next` handler would still see
      // the stale in-flight observable. Calling it after the first
      // `.subscribe()` returns means the whole synchronous emit ->
      // complete -> finalize chain has already run.
      auth.refreshSession().subscribe();
      auth.refreshSession().subscribe(() => {
        expect(fakeAuthApi.refreshCalls).toBe(2);
        done();
      });
    });
  });
});
