import { ChangeDetectionStrategy, Component, Injector, afterNextRender, computed, effect, inject, linkedSignal, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, NavigationStart, ParamMap, Router } from '@angular/router';
import { Location, NgTemplateOutlet, ViewportScroller } from '@angular/common';
import { takeUntilDestroyed, toObservable, toSignal } from '@angular/core/rxjs-interop';
import { debounceTime, distinctUntilChanged, filter, skip } from 'rxjs';
import { ListingSearchParams, ListingSearchResult, MarketplaceListing, SearchFacets } from '../../domain/models';
import { categoryLabelKey, isSpecies, speciesGroups, speciesIcon, speciesLabelKey, speciesOf } from '../../domain/reference/species';
import { cycleTypeLabelKey } from '../../domain/reference/cycle-types';
import { countryName, placeLabelKey } from '../../domain/reference/countries';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { ListingCardComponent } from '../../shared/ui/listing-card/listing-card.component';
import { SelectFieldComponent, SelectOption } from '../../shared/ui/select-field/select-field.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { AnimalIconComponent } from '../../shared/ui/animal-icon/animal-icon.component';
import { BottomSheetComponent } from '../../shared/ui/bottom-sheet/bottom-sheet.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { PriceRangeComponent, PriceRangeValue } from '../../shared/ui/price-range/price-range.component';
import { mediaQuery } from '../../shared/layout/media-query';

// Must match the backend's ports.ListingSort* constants verbatim.
const SORT_VALUES = ['availability', 'recent', 'rating', 'price_asc', 'price_desc'] as const;
type SortValue = (typeof SORT_VALUES)[number];
const SORT_LABEL_KEYS: Readonly<Record<SortValue, string>> = {
  availability: 'marketplace.sortAvailability',
  recent: 'marketplace.sortRecent',
  rating: 'marketplace.sortRating',
  price_asc: 'marketplace.sortPriceAsc',
  price_desc: 'marketplace.sortPriceDesc'
};

/** The statuses a buyer can filter on: closed cycles aren't for sale. */
const OPEN_STATUSES = ['planned', 'in_progress'] as const;

const PAGE_SIZE = 12;
/** Coming back with `?page=N` refetches pages 1..N in one request, up to
 * this many results (the API allows 100). */
const MAX_RESTORE_SIZE = 96;
const SEARCH_DEBOUNCE_MS = 300;
const AVAILABILITY_MONTHS = 6;

interface Filters {
  q: string;
  /** Whole categories (Volaille…); `species` narrows within them. */
  categories: string[];
  species: string[];
  cycleTypes: string[];
  statuses: string[];
  /** `CM|ouest`: a region is only unique within its country. */
  place: string;
  /** A town key within `place`'s region. */
  city: string;
  /** `2026-10`: lots open at some point that month. */
  available: string;
  minPrice: number | null;
  maxPrice: number | null;
  verifiedOnly: boolean;
  sort: SortValue;
}

const EMPTY_FILTERS: Filters = {
  q: '',
  categories: [],
  species: [],
  cycleTypes: [],
  statuses: [],
  place: '',
  city: '',
  available: '',
  minPrice: null,
  maxPrice: null,
  verifiedOnly: false,
  sort: 'availability'
};

/** One removable chip above the results, per active filter. */
interface ActiveChip {
  readonly id: string;
  readonly label: string;
  readonly remove: Partial<Filters>;
}

const EMPTY_FACETS: SearchFacets = { species: [], cycleTypes: [], statuses: [], regions: [], cities: [], priceMin: 0, priceMax: 0 };

/** Where a category stands in the filter: whole, some of its kinds, none. */
type CategoryState = 'all' | 'some' | 'none';

/** The marketplace, per the design: a search, the active filters as
 * removable chips, and the filters themselves — species (with how many
 * lots each), cycle type, region, month of availability, price range,
 * status and verified farmers. From `lg` up they sit in a sidebar beside
 * a card grid; on phones and tablets in a sheet behind the filter
 * button, with a species chip row for the most common choice.
 *
 * Every count comes from the API's facets for the current search, each
 * computed without its own filter, so a buyer sees where a choice leads
 * before making it. Filters live in the URL, so a search is shareable
 * and coming back from a listing restores it. Results load page by page;
 * any filter change starts again from page 1. */
@Component({
  selector: 'bt-marketplace-list',
  imports: [
    FormsModule,
    NgTemplateOutlet,
    ListingCardComponent,
    SelectFieldComponent,
    IconComponent,
    AnimalIconComponent,
    BottomSheetComponent,
    ButtonComponent,
    PriceRangeComponent,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-content-max px-4 pb-8 pt-5 sm:px-8 lg:px-18 lg:py-10">
      <header class="flex items-end justify-between gap-4">
        <div class="flex flex-col gap-1">
          <span class="text-eyebrow text-clay-ink">{{ 'marketplace.eyebrowCount' | t: { count: total() } }}</span>
          <h1 class="text-heading-1">
            <span class="lg:hidden">{{ 'marketplace.titleShort' | t }}</span>
            <span class="hidden lg:inline">{{ 'marketplace.title' | t }}</span>
          </h1>
        </div>
        <div class="hidden w-60 lg:block">
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
            class="h-control-xl w-full rounded-full border border-line bg-surface-card pl-12 pr-4 text-[16px] text-ink outline-none focus:border-forest"
          />
        </div>
        <button
          type="button"
          (click)="filtersOpen.set(true)"
          [attr.aria-label]="('marketplace.filtersTitle' | t) + (activeChips().length ? ' (' + activeChips().length + ')' : '')"
          class="relative flex h-control-xl w-14 shrink-0 items-center justify-center rounded-full bg-ink text-surface"
        >
          <bt-icon name="filter" [size]="22" />
          @if (activeChips().length) {
            <span aria-hidden="true" class="absolute -right-0.5 -top-0.5 flex h-5 min-w-5 items-center justify-center rounded-full bg-clay px-1 text-caption text-on-clay">
              {{ activeChips().length }}
            </span>
          }
        </button>
      </div>

      <div class="no-scrollbar -mx-4 mt-4 flex gap-2 overflow-x-auto px-4 lg:hidden" role="group" [attr.aria-label]="'marketplace.speciesLabel' | t">
        <button type="button" [attr.aria-pressed]="!anyAnimalFilter()" [class]="chipClass(!anyAnimalFilter())" (click)="patch({ categories: [], species: [] })">{{ 'marketplace.allChip' | t }}</button>
        @for (cat of categoryOptions(); track cat.value) {
          <button type="button" [attr.aria-pressed]="cat.state !== 'none'" [class]="chipClass(cat.state !== 'none')" (click)="toggleCategory(cat.value)">
            <bt-animal-icon [name]="cat.icon" [size]="20" />
            {{ cat.label }}
          </button>
        }
      </div>

      @if (activeChips().length) {
        <div class="mt-4 flex flex-wrap items-center gap-2 lg:mt-6" [attr.aria-label]="'marketplace.activeFilters' | t" role="group">
          @for (chip of activeChips(); track chip.id) {
            <button
              type="button"
              (click)="patch(chip.remove)"
              [attr.aria-label]="'marketplace.removeFilter' | t: { label: chip.label }"
              class="inline-flex h-11 items-center gap-1.5 rounded-full bg-ink pl-3.5 pr-2.5 text-small font-semibold text-surface lg:h-9"
            >
              {{ chip.label }}
              <bt-icon name="x" [size]="14" />
            </button>
          }
          <button type="button" (click)="resetFilters()" class="h-11 px-2 text-small font-semibold text-forest underline-offset-2 hover:underline lg:h-9">{{ 'marketplace.clearAll' | t }}</button>
        </div>
      }

      <!-- minmax(0, 1fr): a long unbreakable line in a card can't widen
           the column past the screen. -->
      <div class="mt-5 grid grid-cols-[minmax(0,1fr)] gap-8 lg:mt-6 lg:grid-cols-[280px_minmax(0,1fr)]">
        <aside class="hidden self-start rounded-lg border border-line bg-surface-card p-6 lg:block">
          <div class="mb-5 flex items-center justify-between">
            <span class="text-heading-4">{{ 'marketplace.filtersTitle' | t }}</span>
            <button type="button" (click)="resetFilters()" class="text-small font-semibold text-forest underline-offset-2 hover:underline">
              {{ 'marketplace.filtersReset' | t }}
            </button>
          </div>
          <ng-container *ngTemplateOutlet="filterFields" />
        </aside>

        <section [attr.aria-busy]="result.isLoading()">
          <p class="mb-3 text-small text-ink-muted lg:sr-only" role="status">
            {{ 'marketplace.resultsSorted' | t: { count: total(), sort: (sortLabelKey() | t) } }}
          </p>

          @if (items().length === 0 && result.isLoading()) {
            <div class="grid gap-3 sm:grid-cols-2 sm:gap-5 xl:grid-cols-3">
              @for (i of skeletons; track i) {
                <div class="h-[122px] animate-pulse rounded-lg bg-surface-sunken sm:h-[380px]"></div>
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
              @if (activeChips().length || filters().q) {
                <bt-button variant="outline" (click)="resetFilters()">{{ 'marketplace.filtersReset' | t }}</bt-button>
              }
            </div>
          } @else {
            <!-- One list, laid out for the screen: rows on phones, a grid
                 from sm up. Rendering both and hiding one with CSS would
                 build every card twice. -->
            <div [class]="isWide() ? 'grid grid-cols-2 gap-5 xl:grid-cols-3' : 'flex flex-col gap-3'">
              @for (item of items(); track item.listing.id) {
                <bt-listing-card [item]="item" [layout]="isWide() ? 'tile' : 'row'" />
              }
              @if (loadingMore()) {
                @for (i of moreSkeletons; track i) {
                  <div aria-hidden="true" class="animate-pulse rounded-lg bg-surface-sunken" [class]="isWide() ? 'h-[380px]' : 'h-[148px]'"></div>
                }
              }
            </div>
            <!-- More loads only on request: buyers on metered data pay
                 for what they ask to see, and the footer stays reachable. -->
            <div class="mt-6 flex flex-col items-center gap-3">
              <p class="text-small text-ink-muted" role="status">{{ 'marketplace.shownCount' | t: { shown: items().length, total: total() } }}</p>
              @if (hasMore()) {
                @if (result.error()) {
                  <p class="text-small text-alert">{{ 'marketplace.loadMoreFailed' | t }}</p>
                }
                <bt-button variant="outline" [disabled]="result.isLoading()" (click)="loadMore()">
                  {{ (result.isLoading() ? 'common.loading' : result.error() ? 'common.retry' : 'marketplace.loadMore') | t }}
                </bt-button>
              }
            </div>
          }
        </section>
      </div>
    </div>

    <bt-bottom-sheet [open]="filtersOpen()" [title]="'marketplace.filtersTitle' | t" (closed)="filtersOpen.set(false)">
      <div class="flex flex-col gap-6">
        <bt-select-field [label]="'marketplace.sortLabel' | t" [options]="sortOptions()" [ngModel]="filters().sort" (ngModelChange)="patch({ sort: $event })" />
        <ng-container *ngTemplateOutlet="filterFields" />
      </div>
      <div sheetFooter class="flex gap-3 border-t border-line px-5 pt-3" style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))">
        <bt-button variant="outline" (click)="resetFilters()">{{ 'marketplace.filtersReset' | t }}</bt-button>
        <bt-button class="flex-1" variant="solid" block (click)="filtersOpen.set(false)">
          {{ 'marketplace.showResults' | t: { count: total() } }}
        </bt-button>
      </div>
    </bt-bottom-sheet>

    <ng-template #filterFields>
      <div class="flex flex-col gap-6">
        <fieldset class="flex flex-col gap-0.5">
          <legend class="mb-2 text-eyebrow text-ink-muted">{{ 'marketplace.speciesLabel' | t }}</legend>
          <!-- Each category checks as a whole (all its kinds), or partly
               when some kinds are picked; the chevron shows its kinds. -->
          @for (cat of categoryOptions(); track cat.value) {
            <div class="flex min-h-11 items-center rounded-md hover:bg-surface-sunken" [class.opacity-50]="!cat.count && cat.state === 'none'">
              <label class="flex flex-1 cursor-pointer items-center gap-3 px-1 py-1">
                <input
                  type="checkbox"
                  [checked]="cat.state === 'all'"
                  [indeterminate]="cat.state === 'some'"
                  (change)="toggleCategory(cat.value, cat.expanded)"
                  class="h-5 w-5 shrink-0 rounded border-line-strong accent-forest"
                />
                <bt-animal-icon [name]="cat.icon" [size]="26" class="text-forest" />
                <span class="flex-1 font-semibold">{{ cat.label }}</span>
                <span class="font-mono text-small text-ink-muted">{{ cat.count }}</span>
              </label>
              <button
                type="button"
                (click)="toggleExpanded(cat.value, cat.expanded)"
                [attr.aria-expanded]="cat.expanded"
                [attr.aria-controls]="'kinds-' + cat.value"
                [attr.aria-label]="(cat.expanded ? 'marketplace.hideKinds' : 'marketplace.showKinds') | t: { category: cat.label }"
                class="flex h-10 w-10 shrink-0 items-center justify-center rounded-md text-ink-muted hover:text-ink"
              >
                <bt-icon name="chevron-down" [size]="18" class="transition-transform" [class.rotate-180]="cat.expanded" />
              </button>
            </div>
            @if (cat.expanded) {
              <div [id]="'kinds-' + cat.value" role="group" [attr.aria-label]="cat.label" class="mb-1 ml-3.5 flex flex-col border-l-2 border-line pl-3">
                @for (opt of cat.species; track opt.value) {
                  <label
                    class="flex min-h-10 cursor-pointer items-center gap-2.5 rounded-md px-1 text-small hover:bg-surface-sunken"
                    [class.opacity-50]="!opt.count && !opt.checked"
                  >
                    <input
                      type="checkbox"
                      [checked]="opt.checked"
                      (change)="toggleSpecies(cat.value, opt.value)"
                      class="h-[18px] w-[18px] shrink-0 rounded border-line-strong accent-forest"
                    />
                    <bt-animal-icon [name]="opt.icon" [size]="20" class="text-ink-muted" />
                    <span class="flex-1">{{ opt.label }}</span>
                    <span class="font-mono text-caption text-ink-muted">{{ opt.count }}</span>
                  </label>
                }
              </div>
            }
          }
          @for (opt of otherSpecies(); track opt.value) {
            <label class="flex min-h-11 cursor-pointer items-center gap-3 rounded-md px-1 text-body hover:bg-surface-sunken">
              <input type="checkbox" [checked]="filters().species.includes(opt.value)" (change)="toggle('species', opt.value)" class="h-5 w-5 shrink-0 rounded border-line-strong accent-forest" />
              <span class="flex-1">{{ opt.label }}</span>
              <span class="font-mono text-small text-ink-muted">{{ opt.count }}</span>
            </label>
          }
        </fieldset>

        @if (typeOptions().length) {
          <fieldset>
            <legend class="mb-3 text-eyebrow text-ink-muted">{{ 'marketplace.cycleTypeLabel' | t }}</legend>
            <div class="flex flex-wrap gap-2">
              @for (opt of typeOptions(); track opt.value) {
                <button type="button" [attr.aria-pressed]="filters().cycleTypes.includes(opt.value)" (click)="toggle('cycleTypes', opt.value)" [class]="smallChipClass(filters().cycleTypes.includes(opt.value))">
                  {{ opt.label }} <span class="font-mono opacity-70">{{ opt.count }}</span>
                </button>
              }
            </div>
          </fieldset>
        }

        <bt-select-field
          [label]="'marketplace.regionLabel' | t"
          [options]="regionOptions()"
          [placeholder]="'marketplace.allRegions' | t"
          [ngModel]="filters().place"
          (ngModelChange)="patch({ place: $event, city: '' })"
        />

        @if (filters().place && cityOptions().length) {
          <bt-select-field
            [label]="'marketplace.townLabel' | t"
            [options]="cityOptions()"
            [placeholder]="'marketplace.allTowns' | t"
            [ngModel]="filters().city"
            (ngModelChange)="patch({ city: $event })"
          />
        }

        <bt-select-field
          [label]="'marketplace.availableLabel' | t"
          [options]="monthOptions()"
          [placeholder]="'marketplace.anyDate' | t"
          [ngModel]="filters().available"
          (ngModelChange)="patch({ available: $event })"
        />

        @if (priceBounds(); as bounds) {
          <div class="flex flex-col gap-3">
            <span class="text-eyebrow text-ink-muted">{{ 'marketplace.priceLabel' | t }}</span>
            <bt-price-range [bounds]="bounds" [value]="{ min: filters().minPrice, max: filters().maxPrice }" (valueChange)="setPrice($event)" />
          </div>
        }

        <fieldset class="flex flex-col gap-1">
          <legend class="mb-2 text-eyebrow text-ink-muted">{{ 'marketplace.statusLabel' | t }}</legend>
          @for (opt of statusOptions(); track opt.value) {
            <label class="flex min-h-11 cursor-pointer items-center gap-3 rounded-md px-1 text-body hover:bg-surface-sunken">
              <input
                type="checkbox"
                [checked]="filters().statuses.includes(opt.value)"
                (change)="toggle('statuses', opt.value)"
                class="h-5 w-5 shrink-0 rounded border-line-strong accent-forest"
              />
              <span class="flex-1">{{ opt.label }}</span>
              <span class="font-mono text-small text-ink-muted">{{ opt.count }}</span>
            </label>
          }
        </fieldset>

        <label class="flex min-h-control-lg cursor-pointer items-center justify-between gap-3 rounded-md bg-forest-soft p-3.5">
          <span class="text-small font-semibold text-on-forest-soft">{{ 'marketplace.verifiedOnlyLabel' | t }}</span>
          <span class="relative inline-flex h-[26px] w-11 shrink-0 items-center rounded-full transition has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-focus" [class]="filters().verifiedOnly ? 'bg-forest' : 'bg-line-strong'">
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
  private readonly scroller = inject(ViewportScroller);
  private readonly location = inject(Location);
  private readonly injector = inject(Injector);

  /** Where to scroll once the results are on screen, when coming back
   * (browser back) to a search left for a listing. The router's own
   * restoration runs before the results arrive, on a page too short to
   * reach it. */
  private readonly pendingScroll = signal<number | null>(
    this.router.currentNavigation()?.trigger === 'popstate' ? (savedScroll.get(currentUrl()) ?? null) : null
  );

  protected readonly skeletons = [1, 2, 3, 4, 5, 6];
  protected readonly moreSkeletons = [1, 2, 3];
  protected readonly isWide = mediaQuery('(min-width: 640px)');

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
  protected readonly page = linkedSignal<Filters, number>({
    source: this.filters,
    computation: (_, previous) => (previous ? 1 : this.pageFromUrl())
  });

  /** Coming back from a listing with `?page=3`: the first request fetches
   * pages 1 to 3 at once so the list is as long as the reader left it and
   * the router can restore the scroll position. Later pages load as
   * usual; a filter change drops it. */
  private readonly restorePages = linkedSignal<Filters, number>({
    source: this.filters,
    computation: (_, previous) => (previous ? 1 : this.pageFromUrl())
  });

  private readonly params = computed<ListingSearchParams>(() => {
    const f = this.filters();
    const [country, region] = f.place ? f.place.split('|') : [undefined, undefined];
    return {
      q: f.q || undefined,
      category: f.categories.join(',') || undefined,
      species: f.species.join(',') || undefined,
      cycleType: f.cycleTypes.join(',') || undefined,
      status: f.statuses.join(',') || undefined,
      country: country || undefined,
      region: region || undefined,
      city: f.city || undefined,
      available: f.available || undefined,
      minPrice: f.minPrice ?? undefined,
      maxPrice: f.maxPrice ?? undefined,
      verifiedOnly: f.verifiedOnly || undefined,
      sort: f.sort,
      ...this.pageParams()
    };
  });

  private readonly pageParams = computed(() => {
    const page = this.page();
    const restore = this.restorePages();
    if (page === restore && restore > 1) {
      return { page: 1, pageSize: Math.min(restore * PAGE_SIZE, MAX_RESTORE_SIZE) };
    }
    return { page, pageSize: PAGE_SIZE };
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
  /** A next page is on its way: skeletons follow the cards. */
  protected readonly loadingMore = computed(() => this.result.isLoading() && this.items().length > 0);

  /** The last facets, kept while the next search loads. */
  private readonly facets = linkedSignal<SearchFacets | undefined, SearchFacets>({
    source: () => this.result.value()?.facets,
    computation: (facets, previous) => facets ?? previous?.value ?? EMPTY_FACETS
  });

  /** Categories the buyer opened or closed with the chevron; the others
   * are open exactly when some (not all) of their kinds are picked. */
  private readonly expandedChoice = signal<ReadonlyMap<string, boolean>>(new Map());

  /** Every category open to sale, with the drawing the API gives it, its
   * count the sum of its kinds' (0 included so the list never jumps),
   * where it stands (whole, some kinds, none) and its kinds. */
  protected readonly categoryOptions = computed(() => {
    const counts = new Map(this.facets().species.map((f) => [f.value, f.count]));
    const f = this.filters();
    const expanded = this.expandedChoice();
    return speciesGroups().map((g) => {
      const whole = f.categories.includes(g.key);
      const species = g.species.map((value) => ({
        value,
        label: this.locale.t(speciesLabelKey(value)),
        icon: speciesIcon(value) || g.icon,
        count: counts.get(value) ?? 0,
        checked: whole || f.species.includes(value)
      }));
      const state: CategoryState = whole ? 'all' : species.some((s) => s.checked) ? 'some' : 'none';
      return {
        value: g.key,
        label: this.locale.t(g.labelKey),
        icon: g.icon,
        count: species.reduce((n, s) => n + s.count, 0),
        state,
        expanded: expanded.get(g.key) ?? state === 'some',
        species
      };
    });
  });

  /** Species the listings hold that the catalog doesn't offer. */
  protected readonly otherSpecies = computed(() =>
    this.facets()
      .species.filter((f) => !isSpecies(f.value))
      .map((f) => ({ value: f.value, label: f.value, count: f.count }))
  );

  protected readonly anyAnimalFilter = computed(() => this.filters().categories.length > 0 || this.filters().species.length > 0);

  protected readonly typeOptions = computed(() =>
    this.facets().cycleTypes.map((f) => ({ value: f.value, label: this.locale.t(cycleTypeLabelKey(f.value)), count: f.count }))
  );

  protected readonly statusOptions = computed(() => {
    const counts = new Map(this.facets().statuses.map((f) => [f.value, f.count]));
    return OPEN_STATUSES.map((value) => ({ value, label: this.locale.t(STATUS_LABEL_KEYS[value]), count: counts.get(value) ?? 0 }));
  });

  /** Regions that have lots, with their count, in the reader's
   * language; prefixed with the country once the platform spans several. */
  protected readonly regionOptions = computed<SelectOption[]>(() => {
    const regions = this.facets().regions;
    const multi = new Set(regions.map((r) => r.country)).size > 1;
    const lang = this.locale.locale();
    const options = regions.map((r) => ({
      value: `${r.country}|${r.region}`,
      label: `${multi ? countryName(r.country, lang) + ' · ' : ''}${this.locale.t(placeLabelKey(r.region))} (${r.count})`
    }));
    // Keep the active region listed even when the other filters leave it empty.
    const place = this.filters().place;
    if (place && !options.some((o) => o.value === place)) {
      options.unshift({ value: place, label: this.placeLabel(place) });
    }
    return options;
  });

  /** The chosen region's towns that have lots, with their count. */
  protected readonly cityOptions = computed<SelectOption[]>(() => {
    const [country, region] = this.filters().place.split('|');
    const options = (this.facets().cities ?? [])
      .filter((c) => c.country === country && c.region === region)
      .map((c) => ({ value: c.city, label: `${this.locale.t(placeLabelKey(c.city))} (${c.count})` }));
    // Keep the active town listed even when the other filters leave it empty.
    const city = this.filters().city;
    if (city && !options.some((o) => o.value === city)) {
      options.unshift({ value: city, label: this.locale.t(placeLabelKey(city)) });
    }
    return options;
  });

  /** This month and the next few, for "available in". */
  protected readonly monthOptions = computed<SelectOption[]>(() => {
    const lang = this.locale.locale();
    const now = new Date();
    return Array.from({ length: AVAILABILITY_MONTHS }, (_, i) => {
      const d = new Date(now.getFullYear(), now.getMonth() + i, 1);
      return { value: monthValue(d), label: this.monthLabel(monthValue(d), lang) };
    });
  });

  protected readonly priceBounds = computed(() => {
    const f = this.facets();
    return f.priceMax > f.priceMin ? { min: f.priceMin, max: f.priceMax } : null;
  });

  protected readonly activeChips = computed<ActiveChip[]>(() => {
    const f = this.filters();
    const lang = this.locale.locale();
    const chips: ActiveChip[] = [];
    for (const v of f.categories) {
      // Removing a category drops the kinds picked inside it too.
      const inside = speciesOf(v);
      chips.push({
        id: 'c-' + v,
        label: this.locale.t(categoryLabelKey(v)),
        remove: { categories: f.categories.filter((x) => x !== v), species: f.species.filter((x) => !inside.includes(x)) }
      });
    }
    for (const v of f.species) {
      chips.push({ id: 's-' + v, label: this.locale.t(speciesLabelKey(v)), remove: { species: f.species.filter((x) => x !== v) } });
    }
    for (const v of f.cycleTypes) {
      chips.push({ id: 't-' + v, label: this.locale.t(cycleTypeLabelKey(v)), remove: { cycleTypes: f.cycleTypes.filter((x) => x !== v) } });
    }
    if (f.place) chips.push({ id: 'place', label: this.placeLabel(f.place), remove: { place: '', city: '' } });
    if (f.city) chips.push({ id: 'city', label: this.locale.t(placeLabelKey(f.city)), remove: { city: '' } });
    if (f.available) chips.push({ id: 'available', label: this.monthLabel(f.available, lang), remove: { available: '' } });
    if (f.minPrice !== null || f.maxPrice !== null) {
      const fmt = (n: number) => new Intl.NumberFormat(lang === 'fr' ? 'fr-FR' : 'en-US').format(n);
      const label =
        f.minPrice !== null && f.maxPrice !== null
          ? `${fmt(f.minPrice)} – ${fmt(f.maxPrice)} FCFA`
          : f.minPrice !== null
            ? this.locale.t('marketplace.priceFrom', { price: fmt(f.minPrice) })
            : this.locale.t('marketplace.priceUpTo', { price: fmt(f.maxPrice!) });
      chips.push({ id: 'price', label, remove: { minPrice: null, maxPrice: null } });
    }
    for (const v of f.statuses) {
      chips.push({ id: 'st-' + v, label: this.locale.t(STATUS_LABEL_KEYS[v] ?? v), remove: { statuses: f.statuses.filter((x) => x !== v) } });
    }
    if (f.verifiedOnly) chips.push({ id: 'verified', label: this.locale.t('marketplace.verifiedChip'), remove: { verifiedOnly: false } });
    return chips;
  });

  protected readonly sortLabelKey = computed(() => SORT_LABEL_KEYS[this.filters().sort]);

  constructor() {
    // Leaving: remember where the reader was in this search.
    this.router.events
      .pipe(
        filter((e) => e instanceof NavigationStart),
        takeUntilDestroyed()
      )
      .subscribe(() => savedScroll.set(currentUrl(), this.scroller.getScrollPosition()[1]));
    effect(() => {
      const y = this.pendingScroll();
      if (y !== null && this.items().length && !this.result.isLoading()) {
        this.pendingScroll.set(null);
        afterNextRender(() => this.scroller.scrollToPosition([0, y]), { injector: this.injector });
      }
    });
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
    let mirrored: Filters | undefined;
    effect(() => {
      const f = this.filters();
      const [country, region] = f.place ? f.place.split('|') : [null, null];
      const tree = this.router.createUrlTree([], {
        relativeTo: this.route,
        queryParams: {
          q: f.q || null,
          category: f.categories.join(',') || null,
          species: f.species.join(',') || null,
          cycleType: f.cycleTypes.join(',') || null,
          status: f.statuses.join(',') || null,
          country: country || null,
          region: region || null,
          city: f.city || null,
          available: f.available || null,
          minPrice: f.minPrice ?? null,
          maxPrice: f.maxPrice ?? null,
          verified: f.verifiedOnly ? '1' : null,
          sort: f.sort === 'availability' ? null : f.sort,
          page: this.page() > 1 ? this.page() : null
        }
      });
      const url = this.router.serializeUrl(tree);
      const filtersChanged = f !== mirrored;
      mirrored = f;
      // Same URL: the router would skip the navigation but still scroll
      // to the top.
      if (url === currentUrl()) return;
      if (filtersChanged) {
        this.router.navigateByUrl(tree, { replaceUrl: true });
      } else {
        // Only the page moved on: a router navigation would scroll the
        // reader back to the top mid-list.
        this.location.replaceState(url, '', this.location.getState());
      }
    });
  }

  /** The next page, or the same one again after a failed load. */
  protected loadMore(): void {
    if (this.result.error()) {
      this.result.reload();
    } else {
      this.page.update((p) => p + 1);
    }
  }

  protected patch(change: Partial<Filters>): void {
    this.filters.update((f) => ({ ...f, ...change }));
  }

  /** Picks a whole category, or drops it with any of its kinds. Picking
   * it replaces kinds picked inside it: the whole includes them. From
   * the sidebar, its kinds stay shown or hidden as they were. */
  protected toggleCategory(key: string, expanded?: boolean): void {
    if (expanded !== undefined) this.pinExpanded(key, expanded);
    this.filters.update((f) => {
      const inside = speciesOf(key);
      const species = f.species.filter((s) => !inside.includes(s));
      return f.categories.includes(key)
        ? { ...f, categories: f.categories.filter((c) => c !== key), species }
        : { ...f, categories: [...f.categories, key], species };
    });
  }

  /** Picks or drops one kind. Dropping one from a whole category keeps
   * the others; picking the last missing one makes it whole again. */
  protected toggleSpecies(category: string, value: string): void {
    this.pinExpanded(category, true);
    const kinds = speciesGroups().find((g) => g.key === category)?.species ?? [value];
    this.filters.update((f) => {
      const others = f.species.filter((s) => !kinds.includes(s));
      if (f.categories.includes(category)) {
        return { ...f, categories: f.categories.filter((c) => c !== category), species: [...others, ...kinds.filter((k) => k !== value)] };
      }
      const picked = f.species.includes(value) ? f.species.filter((s) => s !== value) : [...f.species, value];
      if (kinds.every((k) => picked.includes(k))) {
        return { ...f, categories: [...f.categories, category], species: others };
      }
      return { ...f, species: picked };
    });
  }

  protected toggleExpanded(category: string, expanded: boolean): void {
    this.pinExpanded(category, !expanded);
  }

  private pinExpanded(category: string, expanded: boolean): void {
    this.expandedChoice.update((m) => new Map(m).set(category, expanded));
  }

  protected toggle(key: 'species' | 'cycleTypes' | 'statuses', value: string): void {
    this.filters.update((f) => {
      const list = f[key];
      return { ...f, [key]: list.includes(value) ? list.filter((v) => v !== value) : [...list, value] };
    });
  }

  protected setPrice(range: PriceRangeValue): void {
    this.patch({ minPrice: range.min, maxPrice: range.max });
  }

  protected resetFilters(): void {
    this.queryInput.set('');
    this.filters.set({ ...EMPTY_FILTERS, sort: this.filters().sort });
  }

  protected chipClass(active: boolean): string {
    return (
      'inline-flex h-control-md shrink-0 items-center gap-2 rounded-full border px-4 text-body-strong transition ' +
      (active ? 'border-forest bg-forest text-on-forest' : 'border-line bg-surface-card text-ink')
    );
  }

  protected smallChipClass(active: boolean): string {
    return (
      'inline-flex h-11 items-center gap-1.5 rounded-full border px-3.5 text-small font-semibold transition lg:h-9 ' +
      (active ? 'border-forest bg-forest text-on-forest' : 'border-line-strong bg-surface-card text-ink hover:border-forest')
    );
  }

  private placeLabel(place: string): string {
    const [country, region] = place.split('|');
    return region ? this.locale.t(placeLabelKey(region)) : countryName(country, this.locale.locale());
  }

  private monthLabel(value: string, lang: string): string {
    const [y, m] = value.split('-').map(Number);
    if (!y || !m) return value;
    return this.locale.t('marketplace.availableIn', {
      month: new Intl.DateTimeFormat(lang, { month: 'long', year: 'numeric' }).format(new Date(y, m - 1, 1))
    });
  }

  private pageFromUrl(): number {
    const n = Number(this.route.snapshot.queryParamMap.get('page'));
    return Number.isInteger(n) && n > 1 ? Math.min(n, Math.floor(MAX_RESTORE_SIZE / PAGE_SIZE)) : 1;
  }

  private filtersFromUrl(p: ParamMap = this.route.snapshot.queryParamMap): Filters {
    const sort = p.get('sort');
    const list = (key: string) => (p.get(key) ?? '').split(',').map((v) => v.trim()).filter(Boolean);
    const num = (key: string) => {
      const v = Number(p.get(key));
      return p.get(key) !== null && Number.isFinite(v) && v > 0 ? v : null;
    };
    const country = p.get('country')?.toUpperCase() ?? '';
    const region = p.get('region') ?? '';
    const city = p.get('city') ?? '';
    const available = p.get('available') ?? '';
    return {
      q: p.get('q') ?? '',
      categories: list('category'),
      species: list('species'),
      cycleTypes: list('cycleType'),
      statuses: list('status').filter((s) => (OPEN_STATUSES as readonly string[]).includes(s)),
      place: region ? `${country || 'CM'}|${region}` : '',
      city: region ? city : '',
      available: /^\d{4}-\d{2}$/.test(available) ? available : '',
      minPrice: num('minPrice'),
      maxPrice: num('maxPrice'),
      verifiedOnly: p.get('verified') === '1',
      sort: (SORT_VALUES as readonly string[]).includes(sort ?? '') ? (sort as SortValue) : 'availability'
    };
  }
}

/** Scroll offsets of searches the reader left, by URL, for coming back
 * with the browser's back button. */
const savedScroll = new Map<string, number>();

function currentUrl(): string {
  return location.pathname + location.search;
}

const STATUS_LABEL_KEYS: Readonly<Record<string, string>> = {
  planned: 'status.cyclePlanned',
  in_progress: 'status.cycleInProgress'
};

function monthValue(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
}
