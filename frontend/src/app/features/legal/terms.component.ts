import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { TERMS, TERMS_UPDATED } from './terms-content';

/** The general terms of use and sale, linked from registration and the
 * footer. Each article has an anchor (`/terms#paiement`). */
@Component({
  selector: 'bt-terms',
  imports: [RouterLink, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <article class="mx-auto flex max-w-3xl flex-col gap-10 px-5 py-10 sm:px-8 sm:py-16">
      <header class="flex flex-col gap-3">
        <span class="text-eyebrow text-clay-ink">{{ 'terms.eyebrow' | t }}</span>
        <h1 class="text-heading-1">{{ 'terms.title' | t }}</h1>
        <p class="text-small text-ink-muted">{{ 'terms.updated' | t: { date: updated() } }}</p>
      </header>

      <nav class="rounded-lg border border-line bg-surface-card p-5" [attr.aria-label]="'terms.contents' | t">
        <span class="text-small font-bold text-ink-muted">{{ 'terms.contents' | t }}</span>
        <ol class="mt-3 grid list-decimal gap-x-8 gap-y-1.5 pl-5 text-body sm:grid-cols-2">
          @for (section of sections(); track section.id) {
            <li>
              <a routerLink="." [fragment]="section.id" class="text-forest underline-offset-2 hover:underline">{{ section.title }}</a>
            </li>
          }
        </ol>
      </nav>

      @for (section of sections(); track section.id; let i = $index) {
        <section [id]="section.id" class="flex scroll-mt-28 flex-col gap-3">
          <h2 class="text-heading-3">{{ i + 1 }}. {{ section.title }}</h2>
          @for (paragraph of section.paragraphs; track $index) {
            <p class="text-body text-ink-muted">{{ paragraph }}</p>
          }
        </section>
      }
    </article>
  `
})
export class TermsComponent {
  private readonly locale = inject(LocaleService);

  protected readonly sections = computed(() => TERMS[this.locale.locale()]);
  protected readonly updated = computed(() =>
    new Intl.DateTimeFormat(this.locale.locale(), { day: 'numeric', month: 'long', year: 'numeric' }).format(new Date(TERMS_UPDATED))
  );
}
