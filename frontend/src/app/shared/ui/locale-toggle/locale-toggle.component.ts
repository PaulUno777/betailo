import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { Locale, LocaleService } from '../../../core/i18n/locale.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

const LOCALES: readonly Locale[] = ['en', 'fr'];

/** A compact EN/FR switch — used in every shell's header (public, and
 * later portal/admin), so it lives in `shared/ui` rather than one shell. */
@Component({
  selector: 'bt-locale-toggle',
  imports: [TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div [attr.aria-label]="'nav.language' | t" role="group" class="flex items-center gap-0.5 rounded-full bg-surface-sunken p-0.5">
      @for (locale of locales; track locale) {
        <button
          type="button"
          [attr.aria-pressed]="localeService.locale() === locale"
          (click)="localeService.setLocale(locale)"
          [class]="
            'rounded-full px-2 py-1 text-caption uppercase transition ' +
            (localeService.locale() === locale ? 'bg-surface-card text-ink' : 'text-ink-muted')
          "
        >
          {{ locale }}
        </button>
      }
    </div>
  `
})
export class LocaleToggleComponent {
  protected readonly localeService = inject(LocaleService);
  protected readonly locales = LOCALES;
}
