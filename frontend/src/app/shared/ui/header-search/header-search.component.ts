import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router } from '@angular/router';
import { distinctUntilChanged, filter, map } from 'rxjs';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { IconComponent } from '../icon/icon.component';

/** The search field every e-commerce header leads with: type what you
 * want ("poussins Bafoussam"), press Entrée or the button, land on the
 * marketplace with that query. On the marketplace itself it shows the
 * current query, so the header and the page never disagree. */
@Component({
  selector: 'bt-header-search',
  imports: [IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form role="search" class="flex h-11 w-full items-center rounded-full border border-line-strong bg-surface-card pl-4 focus-within:border-forest" (submit)="$event.preventDefault(); submit()">
      <bt-icon name="search" [size]="18" class="shrink-0 text-ink-muted" />
      <input
        type="search"
        name="q"
        enterkeyhint="search"
        [attr.aria-label]="'headerSearch.label' | t"
        [placeholder]="'headerSearch.placeholder' | t"
        [value]="query()"
        (input)="query.set($any($event.target).value)"
        class="h-full min-w-0 flex-1 bg-transparent px-3 text-[16px] text-ink outline-none placeholder:text-ink-muted"
      />
      <button type="submit" class="m-1 flex h-9 shrink-0 items-center rounded-full bg-forest px-4 text-small font-semibold text-on-forest">
        {{ 'headerSearch.submit' | t }}
      </button>
    </form>
  `
})
export class HeaderSearchComponent {
  private readonly router = inject(Router);

  protected readonly query = signal('');

  constructor() {
    // Mirror the marketplace's `q` (and clear it elsewhere).
    this.router.events
      .pipe(
        filter((e) => e instanceof NavigationEnd),
        map(() => {
          const tree = this.router.parseUrl(this.router.url);
          const onMarket = tree.root.children['primary']?.segments[0]?.path === 'marketplace';
          return onMarket ? (tree.queryParamMap.get('q') ?? '') : '';
        }),
        distinctUntilChanged(),
        takeUntilDestroyed()
      )
      .subscribe((q) => this.query.set(q));
  }

  protected submit(): void {
    const q = this.query().trim();
    this.router.navigate(['/marketplace'], { queryParams: { q: q || null } });
  }
}
