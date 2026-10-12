import { ChangeDetectionStrategy, Component, computed, effect, inject, linkedSignal, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, ParamMap, Router } from '@angular/router';
import { NgTemplateOutlet } from '@angular/common';
import { takeUntilDestroyed, toObservable, toSignal } from '@angular/core/rxjs-interop';
import { debounceTime, distinctUntilChanged, skip } from 'rxjs';
import { ListingSearchParams, ListingSearchResult, MarketplaceListing } from '../../domain/models';
import { SPECIES_LABEL_KEYS, SPECIES_VALUES } from '../../domain/reference/species';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { ListingCardComponent } from '../../shared/ui/listing-card/listing-card.component';
import { SelectFieldComponent } from '../../shared/ui/select-field/select-field.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { BottomSheetComponent } from '../../shared/ui/bottom-sheet/bottom-sheet.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';

// Must match the backend's ports.ListingSort* constants verbatim.
const SORT_VALUES = ['recent', 'rating', 'price_asc', 'price_desc'] as const;
type SortValue = (typeof SORT_VALUES)[number];
const SORT_LABEL_KEYS: Readonly<Record<SortValue, string>> = {
  recent: 'marketplace.sortRecent',
  rating: 'marketplace.sortRating',
  price_asc: 'marketplace.sortPriceAsc',
  price_desc: 'marketplace.sortPriceDesc'
};

const PAGE_SIZE = 12;
const SEARCH_DEBOUNCE_MS = 300;

interface Filters {
  q: string;
  species: string;
  cycleType: string;
  region: string;
  minPrice: string;
  maxPrice: string;
  verifiedOnly: boolean;
  sort: SortValue;
}

const EMPTY_FILTERS: Filters = {
  q: '',
  species: '',
  cycleType: '',
  region: '',
  minPrice: '',
  maxPrice: '',
  verifiedOnly: false,
  sort: 'recent'
};

/** The marketplace, mobile-first per the design: a search pill with a
 * filter button (badged with the number of active filters) opening a
 * bottom sheet, a species chip row, and compact row cards. From `lg` up
 * the same filters sit in a sidebar beside a card grid.
 *
 * Filters live in the URL, so a search is shareable and coming back from
 * a listing restores it. Results load page by page with "Voir plus";
 * any filter change starts again from page 1. */
@Component({
  selector: 'bt-marketplace-list',
  imports: [
    FormsModule,
    NgTemplateOutlet,
    ListingCardComponent,
    SelectFieldComponent,
    TextFieldComponent,
    IconComponent,
    BottomSheetComponent,
    ButtonComponent,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-content-max px-4 pb-8 pt-5 sm:px-8 lg:px-18 lg:py-10">
      <header class="flex items-end justify-between gap-4">
        <div class="flex flex-col gap-1">
          <span class="hidden text-eyebrow text-clay-ink lg:block">{{ 'marketplace.eyebrowCount' | t: { count: total() } }}</span>
          <h1 class="text-heading-1">
            <span class="lg:hidden">{{ 'marketplace.titleShort' | t }}</span>
            <span class="hidden lg:inline">{{ 'marketplace.title' | t }}</span>
          </h1>
        </div>
        <div class="hidden w-56 lg:block">
          <bt-select-field [label]="'marketplace.sortLabel' | t" [options]="sortOptions()" [ngModel]="filters().sort" (ngModelChange)="patch({ sort: $event })" />
        </div>
      </header>

      <!-- From lg up the site header's search is the search box (it
           mirrors the query), so the page doesn't show a second one. -->
      <div class="mt-4 flex items-center gap-3 lg:hidden">
        <div class="relative flex flex-1 items-center">
          <bt-icon name="search" [size]="20" class="pointer-events-none absolute left-4 text-ink-muted" />
          <label for="mk-q" class="sr-only">{{ 'marketplace.searchLabel' | t }}</label>
          <input
            id="mk-q"
            type="search"
            enterkeyhint="search"
            autocomplete="off"
            [placeholder]="'marketplace.searchPlaceholder' | t"
            [ngModel]="queryInput()"
            (ngModelChange)="queryInput.set($event)"
            class="h-control-xl w-full rounded-full border border-line bg-surface-card pl-12 pr-4 text-body text-ink outline-none focus:border-forest"
          />
        </div>
        <button
          type="button"
          (click)="filtersOpen.set(true)"
          [attr.aria-label]="'marketplace.filtersTitle' | t"
          class="relative flex h-control-xl w-14 shrink-0 items-center justify-center rounded-full bg-ink text-surface lg:hidden"
        >
          <bt-icon name="filter" [size]="22" />
          @if (activeFilterCount()) {
            <span class="absolute -right-0.5 -top-0.5 flex h-5 min-w-5 items-center justify-center rounded-full bg-clay px-1 text-caption text-on-clay">
              {{ activeFilterCount() }}
            </span>
          }
        </button>
      </div>

      <div class="no-scrollbar -mx-4 mt-4 flex gap-2 overflow-x-auto px-4 lg:hidden" role="group" [attr.aria-label]="'marketplace.speciesLabel' | t">
        <button type="button" [class]="chipClass(!filters().species)" (click)="patch({ species: '' })">{{ 'marketplace.allChip' | t }}</button>
        @for (opt of speciesOptions(); track opt.value) {
          <button type="button" [class]="chipClass(filters().species === opt.value)" (click)="patch({ species: opt.value })">
            {{ opt.label }}
          </button>
        }
      </div>

      <div class="mt-5 grid gap-8 lg:mt-8 lg:grid-cols-[280px_1fr]">
        <aside class="hidden self-start rounded-lg border border-line bg-surface-card p-6 lg:block">
          <div class="mb-5 flex items-center justify-between">
            <span class="text-heading-4">{{ 'marketplace.filtersTitle' | t }}</span>
            <button type="button" (click)="resetFilters()" class="text-small font-semibold text-forest hover:text-forest-deep">
              {{ 'marketplace.filtersReset' | t }}
            </button>
          </div>
          <ng-container *ngTemplateOutlet="filterFields; context: { $implicit: true }" />
        </aside>

        <section aria-live="polite">
          <p class="mb-3 text-small text-ink-muted lg:hidden">
            {{ 'marketplace.resultsSorted' | t: { count: total(), sort: (sortLabelKey() | t) } }}
          </p>

          @if (items().length === 0 && result.isLoading()) {
            <div class="grid gap-3 lg:grid-cols-2 xl:grid-cols-3 lg:gap-5" aria-busy="true">
              @for (i of skeletons; track i) {
                <div class="h-[122px] animate-pulse rounded-lg bg-surface-sunken lg:h-[380px]"></div>
              }
            </div>
          } @else if (result.error() && items().length === 0) {
            <div class="flex flex-col items-center gap-4 py-16 text-center">
              <p class="text-body text-ink">{{ 'marketplace.error' | t }}</p>
              <bt-button variant="outline" (click)="result.reload()">{{ 'common.retry' | t }}</bt-button>
            </div>
          } @else if (items().length === 0) {
            <div class="flex flex-col items-center gap-3 py-16 text-center">
              <bt-icon name="search" [size]="36" class="text-ink-muted" />
              <p class="text-body text-ink">{{ 'marketplace.empty' | t }}</p>
              @if (activeFilterCount() || filters().q) {
                <bt-button variant="outline" (click)="resetFilters()">{{ 'marketplace.filtersReset' | t }}</bt-button>
              }
            </div>
          } @else {
            <div class="flex flex-col gap-3 lg:hidden">
              @for (item of items(); track item.listing.id) {
                <bt-listing-card [item]="item" layout="row" />
              }
            </div>
            <div class="hidden gap-5 lg:grid lg:grid-cols-2 xl:grid-cols-3">
              @for (item of items(); track item.listing.id) {
                <bt-listing-card [item]="item" />
              }
            </div>
            @if (hasMore()) {
              <div class="mt-6 flex justify-center">
                <bt-button variant="outline" [disabled]="result.isLoading()" (click)="page.set(page() + 1)">
                  {{ (result.isLoading() ? 'common.loading' : 'marketplace.loadMore') | t }}
                </bt-button>
              </div>
            }
          }
        </section>
      </div>
    </div>

    <bt-bottom-sheet [open]="filtersOpen()" [title]="'marketplace.filtersTitle' | t" (closed)="filtersOpen.set(false)">
      <div class="flex flex-col gap-5">
        <bt-select-field [label]="'marketplace.sortLabel' | t" [options]="sortOptions()" [ngModel]="filters().sort" (ngModelChange)="patch({ sort: $event })" />
        <ng-container *ngTemplateOutlet="filterFields; context: { $implicit: false }" />
      </div>
      <div sheetFooter class="flex gap-3 border-t border-line px-5 pt-3" style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))">
        <bt-button variant="outline" (click)="resetFilters()">{{ 'marketplace.filtersReset' | t }}</bt-button>
        <bt-button class="flex-1" variant="solid" block (click)="filtersOpen.set(false)">
          {{ 'marketplace.showResults' | t: { count: total() } }}
        </bt-button>
      </div>
    </bt-bottom-sheet>

    <ng-template #filterFields let-withSpecies>
      <div class="flex flex-col gap-5">
        @if (withSpecies) {
          <bt-select-field
            [label]="'marketplace.speciesLabel' | t"
            [options]="speciesOptions()"
            [placeholder]="'marketplace.allSpecies' | t"
            [ngModel]="filters().species"
            (ngModelChange)="patch({ species: $event })"
          />
        }
        <bt-text-field
          [label]="'marketplace.cycleTypeLabel' | t"
          [placeholder]="'marketplace.cycleTypePlaceholder' | t"
          [ngModel]="filters().cycleType"
          (ngModelChange)="patch({ cycleType: $event })"
        />
        <bt-text-field [label]="'marketplace.regionLabel' | t" autocomplete="address-level2" [ngModel]="filters().region" (ngModelChange)="patch({ region: $event })" />
        <div class="grid grid-cols-2 gap-3">
          <bt-text-field
            [label]="'marketplace.minPriceLabel' | t"
            type="number"
            inputmode="numeric"
            [ngModel]="filters().minPrice"
            (ngModelChange)="patch({ minPrice: $event })"
          />
          <bt-text-field
            [label]="'marketplace.maxPriceLabel' | t"
            type="number"
            inputmode="numeric"
            [ngModel]="filters().maxPrice"
            (ngModelChange)="patch({ maxPrice: $event })"
          />
        </div>
        <label class="flex min-h-control-lg items-center justify-between gap-3 rounded-md bg-forest-soft p-3.5">
          <span class="text-small font-semibold text-forest-deep">{{ 'marketplace.verifiedOnlyLabel' | t }}</span>
          <span class="relative inline-flex h-[26px] w-11 shrink-0 items-center rounded-full transition" [class]="filters().verifiedOnly ? 'bg-forest' : 'bg-line-strong'">
            <input
              type="checkbox"
              role="switch"
              [checked]="filters().verifiedOnly"
              (change)="patch({ verifiedOnly: !filters().verifiedOnly })"
              class="absolute inset-0 h-full w-full cursor-pointer opacity-0"
            />
            <span
              class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-surface-card transition"
              [class]="filters().verifiedOnly ? 'translate-x-[22px]' : 'translate-x-[3px]'"
            ></span>
          </span>
        </label>
      </div>
    </ng-template>
  `
})
export class MarketplaceListComponent {
  private readonly listingsApi = inject(ListingsApi);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly locale = inject(LocaleService);

  protected readonly skeletons = [1, 2, 3, 4, 5, 6];

  protected readonly speciesOptions = computed(() =>
    SPECIES_VALUES.map((value) => ({ value, label: this.locale.t(SPECIES_LABEL_KEYS[value]) }))
  );
  protected readonly sortOptions = computed(() =>
    SORT_VALUES.map((value) => ({ value, label: this.locale.t(SORT_LABEL_KEYS[value]) }))
  );

  protected readonly filters = signal<Filters>(this.filtersFromUrl());
  protected readonly filtersOpen = signal(false);

  /** The search box updates on every keystroke; the request waits for a
   * pause in typing. */
  protected readonly queryInput = signal(this.filters().q);
  private readonly debouncedQuery = toSignal(
    toObservable(this.queryInput).pipe(debounceTime(SEARCH_DEBOUNCE_MS), distinctUntilChanged()),
    { initialValue: this.filters().q }
  );

  /** Back to page 1 whenever any filter changes. */
  protected readonly page = linkedSignal({ source: this.filters, computation: () => 1 });

  private readonly params = computed<ListingSearchParams>(() => {
    const f = this.filters();
    return {
      q: f.q || undefined,
      species: f.species || undefined,
      cycleType: f.cycleType || undefined,
      region: f.region || undefined,
      minPrice: f.minPrice ? Number(f.minPrice) : undefined,
      maxPrice: f.maxPrice ? Number(f.maxPrice) : undefined,
      verifiedOnly: f.verifiedOnly || undefined,
      sort: f.sort,
      page: this.page(),
      pageSize: PAGE_SIZE
    };
  });

  protected readonly result = apiResource<ListingSearchResult>(() => this.listingsApi.searchUrl(this.params()));

  /** Accumulated results across "Voir plus" pages. While a page loads,
   * the previous results stay on screen instead of flashing empty. */
  protected readonly items = linkedSignal<{ res: ListingSearchResult | undefined; page: number }, MarketplaceListing[]>({
    source: () => ({ res: this.result.value(), page: this.page() }),
    computation: ({ res, page }, previous) => {
      const kept = previous?.value ?? [];
      if (!res) return kept;
      if (page === 1) return res.items;
      const seen = new Set(kept.map((i) => i.listing.id));
      return [...kept, ...res.items.filter((i) => !seen.has(i.listing.id))];
    }
  });

  protected readonly total = linkedSignal<number | undefined, number>({
    source: () => this.result.value()?.total,
    computation: (total, previous) => total ?? previous?.value ?? 0
  });
  protected readonly hasMore = computed(() => this.items().length < this.total());

  protected readonly activeFilterCount = computed(() => {
    const f = this.filters();
    return [f.species, f.cycleType, f.region, f.minPrice || f.maxPrice, f.verifiedOnly].filter(Boolean).length;
  });

  protected readonly sortLabelKey = computed(() => SORT_LABEL_KEYS[this.filters().sort]);

  constructor() {
    // A search or species chip in the header while already on this page
    // changes the URL without re-creating the component: follow it. The
    // mirror below writes back the same params, which compare equal.
    this.route.queryParamMap.pipe(skip(1), takeUntilDestroyed()).subscribe((p) => {
      const next = this.filtersFromUrl(p);
      if (JSON.stringify(next) !== JSON.stringify(this.filters())) {
        this.filters.set(next);
        this.queryInput.set(next.q);
      }
    });
    effect(() => {
      const q = this.debouncedQuery();
      if (q !== this.filters().q) {
        this.patch({ q });
      }
    });
    // Mirror the filters into the URL (replacing, not pushing, history).
    effect(() => {
      const f = this.filters();
      this.router.navigate([], {
        relativeTo: this.route,
        replaceUrl: true,
        queryParams: {
          q: f.q || null,
          species: f.species || null,
          cycleType: f.cycleType || null,
          region: f.region || null,
          minPrice: f.minPrice || null,
          maxPrice: f.maxPrice || null,
          verified: f.verifiedOnly ? '1' : null,
          sort: f.sort === 'recent' ? null : f.sort
        }
      });
    });
  }

  protected patch(change: Partial<Filters>): void {
    this.filters.update((f) => ({ ...f, ...change }));
  }

  protected resetFilters(): void {
    this.queryInput.set('');
    this.filters.set({ ...EMPTY_FILTERS, sort: this.filters().sort });
  }

  protected chipClass(active: boolean): string {
    return (
      'h-control-md shrink-0 rounded-full border px-4 text-body-strong transition ' +
      (active ? 'border-forest bg-forest text-on-forest' : 'border-line bg-surface-card text-ink')
    );
  }

  private filtersFromUrl(p: ParamMap = this.route.snapshot.queryParamMap): Filters {
    const sort = p.get('sort');
    return {
      q: p.get('q') ?? '',
      species: p.get('species') ?? '',
      cycleType: p.get('cycleType') ?? '',
      region: p.get('region') ?? '',
      minPrice: p.get('minPrice') ?? '',
      maxPrice: p.get('maxPrice') ?? '',
      verifiedOnly: p.get('verified') === '1',
      sort: (SORT_VALUES as readonly string[]).includes(sort ?? '') ? (sort as SortValue) : 'recent'
    };
  }
}
