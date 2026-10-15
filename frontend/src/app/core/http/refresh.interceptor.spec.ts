import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Observable, Subject, of } from 'rxjs';
import { AuthApi, RefreshResult } from '../../data-access/auth/auth.api';
import { AuthService } from '../auth/auth.service';
import { refreshInterceptor } from './refresh.interceptor';

/** A minimal AuthService double — only refreshSession() is relevant to
 * this interceptor; everything else about a real session is out of
 * scope here. */
class FakeAuthService {
  calls = 0;
  result: Observable<string | null> = of('new-token');

  refreshSession(): Observable<string | null> {
    this.calls++;
    return this.result;
  }
}

describe('refreshInterceptor', () => {
  let http: HttpClient;
  let httpMock: HttpTestingController;
  let fakeAuth: FakeAuthService;

  beforeEach(() => {
    fakeAuth = new FakeAuthService();
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([refreshInterceptor])),
        provideHttpClientTesting(),
        { provide: AuthService, useValue: fakeAuth }
      ]
    });
    http = TestBed.inject(HttpClient);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpMock.verify());

  it('passes through a successful request untouched', (done) => {
    http.get('/listings').subscribe((body) => {
      expect(body).toEqual({ ok: true });
      expect(fakeAuth.calls).toBe(0);
      done();
    });
    httpMock.expectOne('/listings').flush({ ok: true });
  });

  it('on a 401, refreshes once and retries with the new token', (done) => {
    http.get('/reservations/mine').subscribe((body) => {
      expect(body).toEqual({ ok: true });
      expect(fakeAuth.calls).toBe(1);
      done();
    });

    const first = httpMock.expectOne('/reservations/mine');
    first.flush('unauthorized', { status: 401, statusText: 'Unauthorized' });

    const retry = httpMock.expectOne((req) => req.url === '/reservations/mine' && req.headers.get('Authorization') === 'Bearer new-token');
    retry.flush({ ok: true });
  });

  it('propagates the original error when the refresh itself fails', (done) => {
    fakeAuth.result = of(null);

    http.get('/reservations/mine').subscribe({
      next: () => fail('expected an error'),
      error: (err) => {
        expect(err.status).toBe(401);
        expect(fakeAuth.calls).toBe(1);
        done();
      }
    });

    httpMock.expectOne('/reservations/mine').flush('unauthorized', { status: 401, statusText: 'Unauthorized' });
    httpMock.expectNone((req) => req.headers.has('Authorization'));
  });

  it('never attempts a refresh for the sign-in calls or /auth/refresh itself', (done) => {
    const paths = ['/auth/login', '/auth/register', '/auth/refresh', '/auth/mfa/verify', '/auth/mfa/passkey/verify', '/auth/password/reset'];
    let remaining = paths.length;
    const checkDone = () => {
      remaining--;
      if (remaining === 0) {
        expect(fakeAuth.calls).toBe(0);
        done();
      }
    };

    for (const path of paths) {
      http.post(path, {}).subscribe({ error: checkDone });
    }

    for (const path of paths) {
      httpMock.expectOne(path).flush('unauthorized', { status: 401, statusText: 'Unauthorized' });
    }
  });

  it('shares one refresh across requests that 401 concurrently (end-to-end, with the real AuthService)', (done) => {
    // This one exercises the real AuthService instead of the fake
    // above — single-flight dedup is AuthService.refreshSession()'s
    // responsibility (see auth.service.spec.ts for that in isolation),
    // so proving it holds through this interceptor needs the real
    // thing, not a fake that would just count every call.
    let refreshApiCalls = 0;
    // A Subject, not of(...): both /a and /b must still be waiting on
    // the same in-flight refresh when it resolves, which requires it
    // to not resolve synchronously the instant the first one asks for
    // it — see the equivalent note in auth.service.spec.ts.
    const pending = new Subject<RefreshResult>();
    const fakeAuthApi = {
      refresh: (): Observable<RefreshResult> => {
        refreshApiCalls++;
        return pending.asObservable();
      },
      logout: () => of(undefined)
    };

    TestBed.resetTestingModule();
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([refreshInterceptor])),
        provideHttpClientTesting(),
        { provide: AuthApi, useValue: fakeAuthApi }
      ]
    });
    const realHttp = TestBed.inject(HttpClient);
    const realHttpMock = TestBed.inject(HttpTestingController);

    let settled = 0;
    const onSettle = () => {
      settled++;
      if (settled === 2) {
        expect(refreshApiCalls).toBe(1);
        done();
      }
    };

    realHttp.get('/a').subscribe(onSettle);
    realHttp.get('/b').subscribe(onSettle);

    realHttpMock.expectOne('/a').flush('unauthorized', { status: 401, statusText: 'Unauthorized' });
    realHttpMock.expectOne('/b').flush('unauthorized', { status: 401, statusText: 'Unauthorized' });

    // Both /a and /b are now waiting on the one shared refresh call —
    // resolve it once.
    pending.next({ accessToken: 'new-token' });
    pending.complete();

    realHttpMock.expectOne((req) => req.url === '/a').flush({ ok: true });
    realHttpMock.expectOne((req) => req.url === '/b').flush({ ok: true });
    realHttpMock.verify();
  });
});
