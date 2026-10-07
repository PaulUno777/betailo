import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, throwError } from 'rxjs';
import { LocaleService } from '../i18n/locale.service';
import { AppError, BackendErrorBody, genericErrorKey, toAppError } from './app-error';

/** Normalizes every failed response into an `AppError` whose `message`
 * is already in the user's language: the backend's refusal code when we
 * have words for it, else the generic sentence for its kind. Nothing
 * downstream ever shows the backend's English developer message. */
export const errorInterceptor: HttpInterceptorFn = (req, next) => {
  const locale = inject(LocaleService);
  return next(req).pipe(
    catchError((response: unknown) => {
      if (response instanceof HttpErrorResponse) {
        const body = typeof response.error === 'object' ? (response.error as BackendErrorBody | null) : null;
        return throwError(() => localize(locale, toAppError(response.status, body)));
      }
      return throwError(() => response);
    })
  );
};

function localize(locale: LocaleService, err: AppError): AppError {
  const params = err.params as Record<string, string | number> | undefined;
  let text = locale.t(err.message, params);
  if (text === err.message) {
    // A code this client has no words for yet.
    text = locale.t(genericErrorKey(err.kind));
  }
  return { ...err, message: text };
}
