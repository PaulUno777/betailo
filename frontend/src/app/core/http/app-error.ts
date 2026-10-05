/** Every backend error response is `{ error: string }`; the status code
 * carries the meaning (verified from backend/interfaces/httpserver —
 * writeError()/writeJSON()). This normalizes both into one shape so
 * features branch on `kind`, never on a raw HTTP status.
 *
 * `message` is either the backend's own text (always English, per
 * ADR-0005 — passed through as-is: there is no reliable way to translate
 * arbitrary backend strings) or, when there is none, one of our own
 * `errors.*` translation keys. `LocaleService.t()` returns an unknown
 * key unchanged, so a consumer can always call `locale.t(err.message)`
 * — a real backend message (no dots) passes straight through, and a
 * fallback key resolves to the localized generic text. */
export type AppErrorKind = 'validation' | 'unauthorized' | 'forbidden' | 'not-found' | 'conflict' | 'server' | 'network';

export interface AppError {
  readonly kind: AppErrorKind;
  readonly message: string;
  readonly status: number;
}

const KIND_BY_STATUS: Readonly<Record<number, AppErrorKind>> = {
  400: 'validation',
  401: 'unauthorized',
  403: 'forbidden',
  404: 'not-found',
  409: 'conflict'
};

export function toAppError(status: number, backendMessage: string | undefined): AppError {
  if (status === 0) {
    return { kind: 'network', message: backendMessage ?? 'errors.network', status };
  }
  const kind = KIND_BY_STATUS[status] ?? 'server';
  return { kind, message: backendMessage ?? 'errors.server', status };
}
