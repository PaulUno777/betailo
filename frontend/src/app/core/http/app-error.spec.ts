import { toAppError } from './app-error';

describe('toAppError', () => {
  it('maps known statuses to their AppErrorKind', () => {
    expect(toAppError(400).kind).toBe('validation');
    expect(toAppError(401).kind).toBe('unauthorized');
    expect(toAppError(403).kind).toBe('forbidden');
    expect(toAppError(404).kind).toBe('not-found');
    expect(toAppError(409).kind).toBe('conflict');
  });

  it('falls back to server for any other status', () => {
    expect(toAppError(500).kind).toBe('server');
    expect(toAppError(502, null).kind).toBe('server');
  });

  it('treats status 0 as a network error', () => {
    expect(toAppError(0)).toEqual({ kind: 'network', message: 'apiErrors.network', status: 0 });
  });

  it('points at the translation of the backend code, with its params', () => {
    const err = toAppError(400, { error: 'reservation: quantity below the minimum: invalid', code: 'reservation.below_min', params: { min: 10 } });
    expect(err.message).toBe('apiErrors.reservation.below_min');
    expect(err.params).toEqual({ min: 10 });
  });

  it('never carries the English developer message', () => {
    expect(toAppError(409, { error: 'conflict' }).message).toBe('apiErrors.conflict');
    expect(toAppError(500, { error: 'internal error', code: 'internal' }).message).toBe('apiErrors.server');
  });
});
