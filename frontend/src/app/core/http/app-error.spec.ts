import { toAppError } from './app-error';

describe('toAppError', () => {
  it('maps known statuses to their AppErrorKind', () => {
    expect(toAppError(400, 'bad input').kind).toBe('validation');
    expect(toAppError(401, 'invalid credentials').kind).toBe('unauthorized');
    expect(toAppError(403, 'forbidden').kind).toBe('forbidden');
    expect(toAppError(404, 'not found').kind).toBe('not-found');
    expect(toAppError(409, 'conflict').kind).toBe('conflict');
  });

  it('falls back to server for any other status', () => {
    expect(toAppError(500, 'internal error').kind).toBe('server');
    expect(toAppError(502, undefined).kind).toBe('server');
  });

  it('treats status 0 as a network error regardless of body', () => {
    const error = toAppError(0, undefined);
    expect(error.kind).toBe('network');
  });

  it('falls back to a translation key when the backend sends no message', () => {
    expect(toAppError(500, undefined).message).toBe('errors.server');
    expect(toAppError(0, undefined).message).toBe('errors.network');
  });

  it('passes the backend message through untranslated for a known status', () => {
    expect(toAppError(409, 'insufficient capacity').message).toBe('insufficient capacity');
  });
});
