import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { AuthService } from '../auth/auth.service';
import { API_BASE_URL } from '../config/api.config';
import { authInterceptor } from './auth.interceptor';

describe('authInterceptor', () => {
  let http: HttpClient;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([authInterceptor])),
        provideHttpClientTesting(),
        { provide: API_BASE_URL, useValue: 'https://api.betailo.test' },
        { provide: AuthService, useValue: { token: () => 'secret-token' } }
      ]
    });
    http = TestBed.inject(HttpClient);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpMock.verify());

  it('sends the session to our own API', () => {
    http.get('https://api.betailo.test/me').subscribe();
    const req = httpMock.expectOne('https://api.betailo.test/me');
    expect(req.request.headers.get('Authorization')).toBe('Bearer secret-token');
    expect(req.request.withCredentials).toBeTrue();
    req.flush({});
  });

  it('never sends it to a presigned storage URL', () => {
    const url = 'https://bucket.r2.cloudflarestorage.com/listings/a/b.jpg?X-Amz-Signature=x';
    http.put(url, new Blob()).subscribe();
    const req = httpMock.expectOne(url);
    expect(req.request.headers.has('Authorization')).toBeFalse();
    expect(req.request.withCredentials).toBeFalse();
    req.flush(null);
  });
});
