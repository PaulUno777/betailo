/** Every backend error response is `{ error, code, params? }`. `error`
 * is an English developer message (ADR-0005) and is never shown: users
 * read our translation of `code`. This normalizes a failed response into
 * one shape so features branch on `kind`, never on a raw HTTP status.
 *
 * `message` is always a translation key: `apiErrors.<code>` for the
 * refusal the backend named, else the generic `apiErrors.<kind>`.
 * `errorInterceptor` resolves it (with `params`) to the user's language,
 * so by the time a feature sees an AppError, `message` is display text. */
export type AppErrorKind = 'validation' | 'unauthorized' | 'forbidden' | 'not-found' | 'conflict' | 'server' | 'network';

export interface AppError {
  readonly kind: AppErrorKind;
  readonly message: string;
  readonly status: number;
  /** The backend's stable refusal code ("listing.photo_required"). */
  readonly code?: string;
  readonly params?: Readonly<Record<string, string | number>>;
}

const KIND_BY_STATUS: Readonly<Record<number, AppErrorKind>> = {
  400: 'validation',
  401: 'unauthorized',
  403: 'forbidden',
  404: 'not-found',
  409: 'conflict'
};

export interface BackendErrorBody {
  readonly error?: string;
  readonly code?: string;
  readonly params?: Record<string, string | number>;
}

/** The generic message for a kind, used when no code is translated. */
export function genericErrorKey(kind: AppErrorKind): string {
  return `apiErrors.${kind}`;
}

export function toAppError(status: number, body?: BackendErrorBody | null): AppError {
  if (status === 0) {
    return { kind: 'network', message: genericErrorKey('network'), status };
  }
  const kind = KIND_BY_STATUS[status] ?? 'server';
  const code = body?.code || undefined;
  return {
    kind,
    status,
    code,
    params: body?.params,
    message: code && kind !== 'server' ? `apiErrors.${code}` : genericErrorKey(kind)
  };
}
