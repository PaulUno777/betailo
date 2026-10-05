import { capacityState } from './capacity';

describe('capacityState', () => {
  it('reads forest below 90% reserved', () => {
    const state = capacityState(100, 50);
    expect(state.tone).toBe('forest');
    expect(state.labelKey).toBe('capacity.remaining');
    expect(state.labelParams).toEqual({ count: 50 });
  });

  it('switches to clay at 90% reserved or more', () => {
    const state = capacityState(100, 8);
    expect(state.tone).toBe('clay');
  });

  it('reads complete once nothing remains', () => {
    const state = capacityState(100, 0);
    expect(state.tone).toBe('complete');
    expect(state.labelKey).toBe('capacity.complete');
  });

  it('never reports a negative or over-100% ratio for an over-booked cycle', () => {
    const state = capacityState(10, -2);
    expect(state.reservedRatio).toBeLessThanOrEqual(1);
    expect(state.tone).toBe('complete');
  });
});
