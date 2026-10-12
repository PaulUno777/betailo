export type CapacityTone = 'forest' | 'clay' | 'complete';

export interface CapacityState {
  readonly reservedRatio: number;
  readonly tone: CapacityTone;
  readonly labelKey: string;
  readonly labelParams?: Record<string, string | number>;
}

/** How full a cycle reads, per `CapacityBar`'s rule: forest fill, clay
 * once 90% reserved, muted "Complet" at 100%. Capacity and availability
 * are always the API's numbers — this only derives a colour and a
 * translation key (never literal text — see `status-registry.ts`) from
 * them. */
export function capacityState(capacity: number, availableQuantity: number): CapacityState {
  const reserved = Math.max(0, capacity - availableQuantity);
  const reservedRatio = capacity > 0 ? Math.min(1, reserved / capacity) : 0;

  if (availableQuantity <= 0) {
    return { reservedRatio: 1, tone: 'complete', labelKey: 'capacity.complete' };
  }
  const tone: CapacityTone = reservedRatio >= 0.9 ? 'clay' : 'forest';
  return { reservedRatio, tone, labelKey: 'capacity.remaining', labelParams: { count: availableQuantity } };
}
