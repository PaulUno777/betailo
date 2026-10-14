import { ChangeDetectorRef, Pipe, PipeTransform, effect, inject } from '@angular/core';
import { LocaleService } from '../../core/i18n/locale.service';

export type LocalDateFormat = 'dayMonth' | 'dayMonthYear' | 'dayMonthTime' | 'monthYear' | 'year';

const OPTIONS: Readonly<Record<LocalDateFormat, Intl.DateTimeFormatOptions>> = {
  dayMonth: { day: 'numeric', month: 'short' }, // "23 sept."
  dayMonthYear: { day: 'numeric', month: 'short', year: 'numeric' }, // "23 sept. 2026"
  dayMonthTime: { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }, // "10 oct. à 16:42"
  monthYear: { month: 'long', year: 'numeric' }, // "juillet 2026"
  year: { year: 'numeric' } // "2024"
};

/** `{{ iso | localDate: 'dayMonth' }}`: an RFC3339 string from the API,
 * formatted for the current UI locale. Impure for the same reason as the
 * `t` pipe: it must re-render when the locale signal changes. */
@Pipe({ name: 'localDate', pure: false })
export class LocalDatePipe implements PipeTransform {
  private readonly locale = inject(LocaleService);

  constructor() {
    const changeDetectorRef = inject(ChangeDetectorRef);
    effect(() => {
      this.locale.locale();
      changeDetectorRef.markForCheck();
    });
  }

  transform(value: string | null | undefined, format: LocalDateFormat = 'dayMonth'): string {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '';
    return date.toLocaleDateString(this.locale.locale() === 'fr' ? 'fr-FR' : 'en-GB', OPTIONS[format]);
  }
}
