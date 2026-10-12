import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { AuthService } from '../auth/auth.service';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const token = inject(AuthService).token();
  // withCredentials on every request, not just /auth/*: harmless
  // elsewhere since the refresh cookie's Path restricts it to /auth
  // anyway (see cookie_config.go), and simpler than threading the flag
  // through every ApiClient call site and every httpResource() URL.
  const withCreds = req.clone({ withCredentials: true });
  if (!token) {
    return next(withCreds);
  }
  return next(withCreds.clone({ setHeaders: { Authorization: `Bearer ${token}` } }));
};
