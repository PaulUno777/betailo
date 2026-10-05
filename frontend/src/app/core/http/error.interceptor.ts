import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { catchError, throwError } from 'rxjs';
import { toAppError } from './app-error';

interface BackendErrorBody {
  readonly error?: string;
}

/** Normalizes every failed response into an `AppError` before it reaches
 * a feature, so nothing downstream branches on a raw `HttpErrorResponse`
 * or the `{error: string}` envelope shape directly. */
export const errorInterceptor: HttpInterceptorFn = (req, next) =>
  next(req).pipe(
    catchError((response: unknown) => {
      if (response instanceof HttpErrorResponse) {
        const body = response.error as BackendErrorBody | null;
        return throwError(() => toAppError(response.status, body?.error));
      }
      return throwError(() => response);
    })
  );
