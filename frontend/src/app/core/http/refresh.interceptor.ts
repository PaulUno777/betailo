import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, switchMap, throwError } from 'rxjs';
import { AuthService } from '../auth/auth.service';

/** Requests a 401 here must never trigger a refresh attempt — login,
 * register, the second sign-in step and a password reset legitimately
 * 401 on a bad password, code or link (there is no session yet to
 * refresh), and excluding /auth/refresh itself is the loop guard: a
 * failed refresh must be allowed to actually fail, not try to refresh
 * its own failure. */
const REFRESH_EXEMPT = ['/auth/login', '/auth/register', '/auth/refresh', '/auth/mfa/', '/auth/password/'];

/** On a 401 from anything else, attempt one silent refresh (see
 * AuthService.refreshSession — single-flight, so concurrent 401s share
 * it) and retry the original request once with the new token. Must sit
 * AFTER errorInterceptor in app.config.ts's interceptor array so it
 * sees the raw HttpErrorResponse before errorInterceptor turns it into
 * an AppError — see that file's comment for why interceptor order maps
 * to array order in reverse for the response path. */
export const refreshInterceptor: HttpInterceptorFn = (req, next) => {
  if (REFRESH_EXEMPT.some((path) => req.url.includes(path))) {
    return next(req);
  }

  const auth = inject(AuthService);

  return next(req).pipe(
    catchError((err: unknown) => {
      if (!(err instanceof HttpErrorResponse) || err.status !== 401) {
        return throwError(() => err);
      }
      return auth.refreshSession().pipe(
        switchMap((newToken) => {
          if (!newToken) {
            return throwError(() => err);
          }
          return next(req.clone({ setHeaders: { Authorization: `Bearer ${newToken}` } }));
        })
      );
    })
  );
};
