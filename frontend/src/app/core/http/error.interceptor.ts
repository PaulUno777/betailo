import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { LocaleService } from '../i18n/locale.service';
import { AppError, BackendErrorBody, genericErrorKey, toAppError } from './app-error';

/** Normalizes every failed response into an `AppError` whose `message`
 * is already in the user's language: the backend's refusal code when we
 * have words for it, else the generic sentence for its kind. Nothing
 * downstream ever shows the backend's English developer message. */
export const errorInterceptor: HttpInterceptorFn = (req, next) => {
  const locale = inject(LocaleService);
  const router = inject(Router);
  return next(req).pipe(
    catchError((response: unknown) => {
      if (response instanceof HttpErrorResponse) {
        const body = typeof response.error === 'object' ? (response.error as BackendErrorBody | null) : null;
        // The back office wants a second factor this session hasn't
        // passed: the security page says what to do about it.
        if (body?.code === MFA_REQUIRED && !router.url.startsWith('/account/security')) {
          void router.navigate(['/account/security'], { queryParams: { required: 1 } });
        }
        return throwError(() => localize(locale, toAppError(response.status, body)));
      }
      return throwError(() => response);
    })
  );
};

const MFA_REQUIRED = 'admin.mfa_required';

function localize(locale: LocaleService, err: AppError): AppError {
  const params = err.params as Record<string, string | number> | undefined;
  let text = locale.t(err.message, params);
  if (text === err.message) {
    // A code this client has no words for yet.
    text = locale.t(genericErrorKey(err.kind));
  }
  return { ...err, message: text };
}
