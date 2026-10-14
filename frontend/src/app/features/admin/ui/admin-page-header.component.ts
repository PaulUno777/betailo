import { ChangeDetectionStrategy, Component, input } from '@angular/core';

/** How every back-office page opens, per the admin artboards: a mono
 * eyebrow naming the section, the page title, and on the right whatever
 * the page projects (a period switch, its one primary action). */
@Component({
  selector: 'bt-admin-page-header',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div class="flex min-w-0 flex-col gap-0.5">
        <span class="text-eyebrow text-ink-muted">{{ eyebrow() }}</span>
        <h1 class="font-display text-[28px] leading-tight font-extrabold tracking-[-0.02em] sm:text-[32px]">{{ title() }}</h1>
        @if (subtitle(); as text) {
          <p class="mt-1 max-w-2xl text-small text-ink-muted">{{ text }}</p>
        }
      </div>
      <div class="flex flex-wrap items-center gap-2"><ng-content /></div>
    </header>
  `
})
export class AdminPageHeaderComponent {
  readonly eyebrow = input.required<string>();
  readonly title = input.required<string>();
  readonly subtitle = input<string>();
}
