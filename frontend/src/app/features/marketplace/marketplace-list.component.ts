import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { httpResource } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';
import { ListingSearchParams, ListingSearchResult } from '../../domain/models';
import { SPECIES_LABEL_KEYS, SPECIES_VALUES } from '../../domain/reference/species';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ListingCardComponent } from '../../shared/ui/listing-card/listing-card.component';
import { SelectFieldComponent } from '../../shared/ui/select-field/select-field.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';

// Must match the backend's ports.ListingSort* constants verbatim.
const SORT_VALUES = ['recent', 'rating', 'price_asc', 'price_desc'] as const;
const SORT_LABEL_KEYS: Readonly<Record<(typeof SORT_VALUES)[number], string>> = {
  recent: 'marketplace.sortRecent',
  rating: 'marketplace.sortRating',
  price_asc: 'marketplace.sortPriceAsc',
  price_desc: 'marketplace.sortPriceDesc'
};

const PAGE_SIZE = 12;

@Component({
  selector: 'bt-marketplace-list',
  imports: [FormsModule, ListingCardComponent, SelectFieldComponent, TextFieldComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-content-max px-5 py-8 sm:px-18">
      <div class="mb-6 flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
        <div class="flex flex-col gap-1">
          <span class="text-eyebrow text-clay-ink">{{ 'marketplace.eyebrowCount' | t: { count: result.value()?.total ?? 0 } }}</span>
          <h1 class="text-heading-1">{{ 'marketplace.title' | t }}</h1>
        </div>
        <bt-select-field [label]="'marketplace.sortLabel' | t" [options]="sortOptions()" [ngModel]="sort()" (ngModelChange)="sort.set($event)" />
      </div>
      <div class="relative mb-8 flex items-center">
        <bt-icon name="search" [size]="20" class="pointer-events-none absolute left-5 text-ink-muted" />
        <label for="mk-q" class="sr-only">{{ 'marketplace.searchLabel' | t }}</label>
        <input
          id="mk-q"
          type="search"
          [placeholder]="'marketplace.searchPlaceholder' | t"
          [ngModel]="query()"
          (ngModelChange)="query.set($event)"
          class="h-14 w-full rounded-full border border-line bg-surface-card pl-[52px] pr-5 text-body-l text-ink outline-none focus:border-forest"
        />
      </div>
      <div class="grid gap-8 lg:grid-cols-[280px_1fr]">
        <aside class="flex flex-col gap-6 rounded-lg border border-line bg-surface-card p-6">
          <div class="flex items-center justify-between">
            <span class="text-heading-4">{{ 'marketplace.filtersTitle' | t }}</span>
            <button type="button" (click)="resetFilters()" class="text-small font-semibold text-forest hover:text-forest-deep">
              {{ 'marketplace.filtersReset' | t }}
            </button>
          </div>
          <bt-select-field
            [label]="'marketplace.speciesLabel' | t"
            [options]="speciesOptions()"
            [placeholder]="'marketplace.allSpecies' | t"
            [ngModel]="species()"
            (ngModelChange)="species.set($event)"
          />
          <bt-text-field
            [label]="'marketplace.cycleTypeLabel' | t"
            [placeholder]="'marketplace.cycleTypePlaceholder' | t"
            [ngModel]="cycleType()"
            (ngModelChange)="cycleType.set($event)"
          />
          <bt-text-field [label]="'marketplace.regionLabel' | t" [ngModel]="region()" (ngModelChange)="region.set($event)" />
          <div class="grid grid-cols-2 gap-3">
            <bt-text-field
              [label]="'marketplace.minPriceLabel' | t"
              type="number"
              [ngModel]="minPrice()"
              (ngModelChange)="minPrice.set($event)"
            />
            <bt-text-field
              [label]="'marketplace.maxPriceLabel' | t"
              type="number"
              [ngModel]="maxPrice()"
              (ngModelChange)="maxPrice.set($event)"
            />
          </div>
          <label class="flex items-center justify-between gap-3 rounded-md bg-forest-soft p-3.5">
            <span class="text-small font-semibold text-forest-deep">{{ 'marketplace.verifiedOnlyLabel' | t }}</span>
            <span class="relative inline-flex h-[26px] w-11 shrink-0 items-center rounded-full transition" [class]="verifiedOnly() ? 'bg-forest' : 'bg-line-strong'">
              <input
                type="checkbox"
                [checked]="verifiedOnly()"
                (change)="verifiedOnly.set(!verifiedOnly())"
                class="absolute inset-0 h-full w-full cursor-pointer opacity-0"
              />
              <span
                class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-surface-card transition"
                [class]="verifiedOnly() ? 'translate-x-[22px]' : 'translate-x-[3px]'"
              ></span>
            </span>
          </label>
        </aside>
        <section>
          @if (result.isLoading()) {
            <p class="text-body text-ink-muted">{{ 'marketplace.loading' | t }}</p>
          } @else if (result.error()) {
            <p class="text-body text-alert">{{ 'marketplace.error' | t }}</p>
          } @else {
            <div class="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
              @for (item of result.value()?.items ?? []; track item.listing.id) {
                <bt-listing-card [item]="item" />
              }
            </div>
            @if ((result.value()?.items ?? []).length === 0) {
              <p class="text-body text-ink-muted">{{ 'marketplace.empty' | t }}</p>
            }
          }
          <nav aria-label="Pagination" class="mt-8 flex items-center justify-center gap-2">
            <button
              type="button"
              [attr.aria-label]="'marketplace.previous' | t"
              [disabled]="page() <= 1"
              (click)="page.set(page() - 1)"
              class="flex h-control-md w-control-md items-center justify-center rounded-sm border border-line text-ink hover:border-line-strong disabled:opacity-50 disabled:pointer-events-none"
            >
              <bt-icon name="chevron-left" [size]="16" />
            </button>
            @for (p of pageNumbers(); track $index) {
              @if (p === null) {
                <span class="px-1 text-small text-ink-muted">…</span>
              } @else {
                <button
                  type="button"
                  (click)="page.set(p)"
                  [attr.aria-current]="p === page() ? 'page' : null"
                  [class]="
                    'flex h-control-md w-control-md items-center justify-center rounded-sm border text-small font-medium ' +
                    (p === page() ? 'border-ink bg-ink text-on-forest-deep' : 'border-line text-ink hover:border-line-strong')
                  "
                >
                  {{ p }}
                </button>
              }
            }
            <button
              type="button"
              [attr.aria-label]="'marketplace.next' | t"
              [disabled]="page() >= totalPages()"
              (click)="page.set(page() + 1)"
              class="flex h-control-md w-control-md items-center justify-center rounded-sm border border-line text-ink hover:border-line-strong disabled:opacity-50 disabled:pointer-events-none"
            >
              <bt-icon name="chevron-right" [size]="16" />
            </button>
          </nav>
        </section>
      </div>
    </div>
  `
})
export class MarketplaceListComponent {
  private readonly listingsApi = inject(ListingsApi);
  private readonly route = inject(ActivatedRoute);
  private readonly locale = inject(LocaleService);

  private readonly initialQuery = toSignal(
    this.route.queryParamMap.pipe(
      map((params) => ({
        species: params.get('species') ?? '',
        region: params.get('region') ?? ''
      }))
    ),
    { initialValue: { species: '', region: '' } }
  );

  protected readonly speciesOptions = computed(() =>
    SPECIES_VALUES.map((value) => ({ value, label: this.locale.t(SPECIES_LABEL_KEYS[value]) }))
  );
  protected readonly sortOptions = computed(() =>
    SORT_VALUES.map((value) => ({ value, label: this.locale.t(SORT_LABEL_KEYS[value]) }))
  );

  protected readonly query = signal('');
  protected readonly species = signal(this.initialQuery().species);
  protected readonly cycleType = signal('');
  protected readonly region = signal(this.initialQuery().region);
  protected readonly minPrice = signal('');
  protected readonly maxPrice = signal('');
  protected readonly verifiedOnly = signal(false);
  protected readonly sort = signal('recent');
  protected readonly page = signal(1);

  private readonly params = computed<ListingSearchParams>(() => ({
    q: this.query() || undefined,
    species: this.species() || undefined,
    cycleType: this.cycleType() || undefined,
    region: this.region() || undefined,
    minPrice: this.minPrice() ? Number(this.minPrice()) : undefined,
    maxPrice: this.maxPrice() ? Number(this.maxPrice()) : undefined,
    verifiedOnly: this.verifiedOnly() || undefined,
    sort: this.sort(),
    page: this.page(),
    pageSize: PAGE_SIZE
  }));

  protected readonly result = httpResource<ListingSearchResult>(() => this.listingsApi.searchUrl(this.params()));

  protected resetFilters(): void {
    this.query.set('');
    this.species.set('');
    this.cycleType.set('');
    this.region.set('');
    this.minPrice.set('');
    this.maxPrice.set('');
    this.verifiedOnly.set(false);
    this.page.set(1);
  }

  protected readonly totalPages = computed(() => Math.max(1, Math.ceil((this.result.value()?.total ?? 0) / PAGE_SIZE)));

  /** A windowed page-number list (first, last, and up to 2 neighbours of
   * the current page), with `null` marking an ellipsis gap — the design's
   * "1 2 3 … 10" pattern without hardcoding a page count. */
  protected readonly pageNumbers = computed<(number | null)[]>(() => {
    const total = this.totalPages();
    const current = this.page();
    const pages = new Set<number>([1, total, current, current - 1, current + 1]);
    const sorted = [...pages].filter((p) => p >= 1 && p <= total).sort((a, b) => a - b);
    const out: (number | null)[] = [];
    let prev = 0;
    for (const p of sorted) {
      if (prev && p - prev > 1) {
        out.push(null);
      }
      out.push(p);
      prev = p;
    }
    return out;
  });
}
