import { DestroyRef, Signal, inject, signal } from '@angular/core';

/** Whether a CSS media query matches, kept current as the window
 * resizes. For layouts that must render differently, not just look
 * different: rendering one list instead of two hidden by CSS. Call it in
 * an injection context; false where there's no `window`.
 *
 *   protected readonly isWide = mediaQuery('(min-width: 640px)'); */
export function mediaQuery(query: string): Signal<boolean> {
  if (typeof window === 'undefined' || !window.matchMedia) {
    return signal(false).asReadonly();
  }
  const list = window.matchMedia(query);
  const matches = signal(list.matches);
  const onChange = (e: MediaQueryListEvent) => matches.set(e.matches);
  list.addEventListener('change', onChange);
  inject(DestroyRef).onDestroy(() => list.removeEventListener('change', onChange));
  return matches.asReadonly();
}
