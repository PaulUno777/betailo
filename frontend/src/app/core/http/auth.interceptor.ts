import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { AuthService } from '../auth/auth.service';
import { API_BASE_URL } from '../config/api.config';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  // Only our own API gets the session. Presigned object-storage URLs
  // (photo uploads and downloads) must go out bare: they already carry
  // their own signature, S3-style storage rejects a second auth
  // mechanism, and the access token must never leak to a third party.
  if (!req.url.startsWith(inject(API_BASE_URL))) {
    return next(req);
  }
  const token = inject(AuthService).token();
  // withCredentials on every API request, not just /auth/*: harmless
  // elsewhere since the refresh cookie's Path restricts it to /auth
  // anyway (see cookie_config.go), and simpler than threading the flag
  // through every ApiClient call site and every httpResource() URL.
  const withCreds = req.clone({ withCredentials: true });
  if (!token) {
    return next(withCreds);
  }
  return next(withCreds.clone({ setHeaders: { Authorization: `Bearer ${token}` } }));
};
