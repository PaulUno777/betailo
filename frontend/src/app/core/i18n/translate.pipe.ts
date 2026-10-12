import { ChangeDetectorRef, Pipe, PipeTransform, effect, inject } from '@angular/core';
import { LocaleService } from './locale.service';

/** `{{ 'home.heroTitle' | t }}`, or with interpolation: `{{ 'capacity.remaining' | t: { count: 5 } }}`.
 *
 * Deliberately `pure: false`: a *pure* pipe is only re-invoked when its
 * own arguments change identity — reading a signal inside `transform`
 * does not make Angular re-run it when that signal changes later, since
 * pure-pipe memoization only ever compares the pipe's own input
 * (verified live: locale switching silently failed to re-render with a
 * pure pipe). Instead, an `effect()` in the constructor watches the
 * locale signal and calls `markForCheck()` on this pipe's own host view,
 * and the impure pipe then genuinely re-evaluates on that check. */
@Pipe({ name: 't', pure: false })
export class TranslatePipe implements PipeTransform {
  private readonly locale = inject(LocaleService);

  constructor() {
    const changeDetectorRef = inject(ChangeDetectorRef);
    effect(() => {
      this.locale.locale();
      changeDetectorRef.markForCheck();
    });
  }

  transform(key: string, params?: Record<string, string | number>): string {
    return this.locale.t(key, params);
  }
}
