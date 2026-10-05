import { httpResource } from '@angular/common/http';
import { ResourceStatus, Signal, computed } from '@angular/core';

/** The read side of an `httpResource` that never throws from a template.
 *
 * Angular's `Resource.value()` *throws* while the resource is in its
 * error state, so a single failed request (offline, a 5xx, a dropped
 * mobile connection) read straight from a template crashes change
 * detection and blanks the whole page. Here `value()` is simply
 * `undefined` until a value exists, and failures surface through
 * `error()` for the component to render, never as an exception. */
export interface ApiResource<T> {
  readonly value: Signal<T | undefined>;
  readonly isLoading: Signal<boolean>;
  readonly error: Signal<unknown>;
  readonly status: Signal<ResourceStatus>;
  hasValue(): boolean;
  reload(): boolean;
}

/** `httpResource`, wrapped so reading it can't throw — see `ApiResource`.
 * Like `httpResource`, a request function returning `undefined` means
 * "nothing to fetch yet" (e.g. waiting on another resource's id). */
export function apiResource<T>(url: () => string | undefined): ApiResource<T> {
  const res = httpResource<T>(url);
  return {
    value: computed(() => (res.hasValue() ? res.value() : undefined)),
    isLoading: res.isLoading,
    error: res.error,
    status: res.status,
    hasValue: () => res.hasValue(),
    reload: () => res.reload()
  };
}
