import { ReferenceService } from '../../core/reference/reference.service';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { ForumPost, ListingSearchResult } from '../../domain/models';
import { speciesGroups, speciesLabelKey } from '../../domain/reference/species';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { ForumApi } from '../../data-access/forum/forum.api';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { PlacePipe } from '../../core/i18n/place.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { ListingCardComponent } from '../../shared/ui/listing-card/listing-card.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { IconName } from '../../shared/ui/icon/icon-paths';
import { AnimalIconComponent } from '../../shared/ui/animal-icon/animal-icon.component';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { categoryTone } from '../../domain/rules/category-tone';
import { apiResource } from '../../core/http/api-resource';
import { AuthService } from '../../core/auth/auth.service';
import { SellerSpaceService } from '../../core/auth/seller-space.service';

@Component({
  selector: 'bt-home',
  imports: [
    FormsModule,
    RouterLink,
    ButtonComponent,
    ListingCardComponent,
    IconComponent,
    AnimalIconComponent,
    BadgeComponent,
    TranslatePipe, PlacePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <!-- Phones: the app home from the mobile design. -->
    <div class="relative flex flex-col gap-7 overflow-hidden px-4 pb-4 pt-6 sm:hidden">
      <div aria-hidden="true" class="animal-pattern text-forest opacity-10"></div>
      <header class="relative flex flex-col gap-1">
        <h1 class="text-heading-1">{{ 'home.mobileTitle' | t }}</h1>
      </header>

      <form (submit)="$event.preventDefault(); searchQuery()" role="search">
        <label for="home-q" class="sr-only">{{ 'marketplace.searchLabel' | t }}</label>
        <div class="relative flex items-center">
          <bt-icon name="search" [size]="20" class="pointer-events-none absolute left-4 text-ink-muted" />
          <input
            id="home-q"
            type="search"
            enterkeyhint="search"
            autocomplete="off"
            [placeholder]="'home.mobileSearchPlaceholder' | t"
            [ngModel]="query()"
            (ngModelChange)="query.set($event)"
            name="q"
            class="h-control-xl w-full rounded-full border border-line bg-surface-card pl-12 pr-4 text-body text-ink outline-none focus:border-forest"
            style="box-shadow: var(--shadow-float)"
          />
        </div>
      </form>

      <nav class="no-scrollbar -mx-4 flex gap-2 overflow-x-auto px-4" [attr.aria-label]="'home.searchSpeciesLabel' | t">
        @for (g of groups; track g.key) {
          <a
            [routerLink]="['/marketplace']"
            [queryParams]="{ category: g.key }"
            class="flex h-control-md shrink-0 items-center gap-2 rounded-full border border-line bg-surface-card px-4 text-body-strong"
          >
            <bt-animal-icon [name]="g.icon" [size]="22" class="text-forest" />
            {{ g.labelKey | t }}
          </a>
        }
      </nav>

      @if (topRated.value()?.items?.[0]; as top) {
        <a [routerLink]="['/marketplace', top.listing.id]" class="relative block aspect-[16/11] overflow-hidden rounded-xl bg-forest-deep">
          @if (top.listing.photos?.[0]; as photo) {
            <img [src]="photo" [alt]="top.listing.title" class="absolute inset-0 h-full w-full object-cover" />
          }
          <div class="absolute inset-0 bg-gradient-to-t from-forest-deep via-forest-deep/40 to-transparent"></div>
          @if (top.farmVerified) {
            <span class="absolute left-4 top-4 rounded-full bg-clay px-3 py-1 text-caption text-on-clay">{{ 'home.verifiedBreeder' | t }}</span>
          }
          <div class="absolute inset-x-4 bottom-4 flex flex-col gap-1 text-on-forest-deep">
            <span class="text-heading-3 leading-tight">{{ top.listing.title }}</span>
            <span class="flex items-center gap-1 text-small text-on-forest-deep/85">
              {{ top.farmName }} · {{ top.farmLocation | place }}
              @if (top.reviewCount) {
                · <bt-icon name="star" [size]="13" /> {{ top.reviewAverage.toFixed(1) }}
              }
            </span>
          </div>
        </a>
      }

      <section class="flex flex-col gap-3">
        <div class="flex items-baseline justify-between">
          <h2 class="text-heading-3">{{ 'home.latestTitle' | t }}</h2>
          <a routerLink="/marketplace" class="text-body-strong text-forest underline">{{ 'home.seeAll' | t }}</a>
        </div>
        @if (featured.value()?.items?.length) {
          <div class="no-scrollbar -mx-4 flex snap-x snap-mandatory gap-3 overflow-x-auto scroll-px-4 px-4 pb-1">
            @for (item of featured.value()!.items; track item.listing.id) {
              <div class="w-[72%] shrink-0 snap-start">
                <bt-listing-card [item]="item" />
              </div>
            }
          </div>
        } @else if (featured.isLoading()) {
          <div class="flex gap-3 overflow-hidden">
            <div class="h-72 w-[72%] shrink-0 animate-pulse rounded-lg bg-surface-sunken"></div>
            <div class="h-72 w-[72%] shrink-0 animate-pulse rounded-lg bg-surface-sunken"></div>
          </div>
        } @else {
          <p class="rounded-lg border border-dashed border-line-strong p-6 text-body text-ink-muted">{{ 'home.featuredEmpty' | t }}</p>
        }
      </section>
    </div>

    <div class="relative hidden overflow-hidden sm:block">
      <div aria-hidden="true" class="animal-pattern text-forest opacity-10"></div>
      <section class="relative mx-auto max-w-content-max px-5 py-16 sm:px-18 sm:py-22">
        <div class="grid gap-10 lg:grid-cols-2 lg:items-center">
          <div class="flex flex-col gap-6">
            <span class="inline-flex w-fit items-center gap-1.5 rounded-full bg-forest-soft px-3 py-1.5 text-caption text-forest">
              <bt-icon name="shield" [size]="14" />
              {{ 'home.heroTrustChip' | t }}
            </span>
            <h1 class="text-display-l sm:text-display-xl">
              {{ 'home.heroTitleStart' | t }} <span class="text-forest">{{ 'home.heroTitleAccent' | t }}</span>
            </h1>
            <p class="max-w-lg text-body-l text-ink-muted">{{ 'home.heroSubtitle' | t }}</p>
            <form
              (submit)="$event.preventDefault(); search()"
              class="flex flex-col gap-3 rounded-lg bg-surface-card p-3 sm:flex-row sm:items-center sm:gap-0 sm:rounded-full sm:py-1.5 sm:pl-6 sm:pr-1.5"
              style="box-shadow: var(--shadow-float)"
            >
              <div class="flex flex-1 flex-col gap-0.5 border-b border-line px-2 pb-2 sm:border-b-0 sm:border-r sm:px-4 sm:py-0">
                <label for="home-species" class="text-small font-bold text-ink-muted">{{ 'home.searchSpeciesLabel' | t }}</label>
                <select
                  id="home-species"
                  class="h-6 w-full appearance-none bg-transparent text-body-strong text-ink outline-none"
                  [ngModel]="species()"
                  (ngModelChange)="species.set($event)"
                  name="species"
                >
                  <option value="">{{ 'home.searchAllSpecies' | t }}</option>
                  <!-- A whole category, or one kind within it. -->
                  @for (g of groups; track g.key) {
                    <optgroup [label]="g.labelKey | t">
                      <option [value]="'category:' + g.key">{{ 'home.searchWholeCategory' | t: { category: (g.labelKey | t) } }}</option>
                      @for (s of g.species; track s) {
                        <option [value]="s">{{ speciesLabel(s) | t }}</option>
                      }
                    </optgroup>
                  }
                </select>
              </div>
              <div class="flex flex-1 flex-col gap-0.5 border-b border-line px-2 pb-2 sm:border-b-0 sm:border-r sm:px-4 sm:py-0">
                <label for="home-city" class="text-small font-bold text-ink-muted">{{ 'home.searchCityLabel' | t }}</label>
                <input
                  id="home-city"
                  type="text"
                  autocomplete="address-level2"
                  [placeholder]="'home.searchCityPlaceholder' | t"
                  class="h-6 w-full bg-transparent text-body-strong text-ink outline-none placeholder:font-normal placeholder:text-ink-muted"
                  [ngModel]="city()"
                  (ngModelChange)="city.set($event)"
                  name="city"
                />
              </div>
              <div class="flex flex-1 flex-col gap-0.5 px-2 sm:px-4">
                <label for="home-month" class="text-small font-bold text-ink-muted">{{ 'home.searchAvailableLabel' | t }}</label>
                <select
                  id="home-month"
                  class="h-6 w-full appearance-none bg-transparent text-body-strong text-ink outline-none"
                  [ngModel]="month()"
                  (ngModelChange)="month.set($event)"
                  name="month"
                >
                  <option value="">{{ 'marketplace.anyDate' | t }}</option>
                  @for (opt of monthOptions(); track opt.value) {
                    <option [value]="opt.value">{{ opt.label }}</option>
                  }
                </select>
              </div>
              <bt-button type="submit" size="lg">
                <bt-icon name="search" [size]="18" />
                {{ 'home.searchButton' | t }}
              </bt-button>
            </form>
          </div>

          <div class="relative hidden lg:-mr-12 lg:block xl:-mr-20">
            <!-- A soft glow so the cut-out animals sit on the page, not in a box. -->
            <div aria-hidden="true" class="absolute inset-x-[8%] inset-y-[4%] rounded-full bg-forest-soft opacity-80 blur-3xl"></div>
            <div aria-hidden="true" class="absolute bottom-[6%] right-[4%] h-2/5 w-2/5 rounded-full bg-clay-soft opacity-70 blur-3xl"></div>
            <img
              src="/images/hero-farm.webp"
              width="1100"
              height="733"
              fetchpriority="high"
              [alt]="'home.heroImageAlt' | t"
              class="relative h-auto w-full origin-bottom-right scale-115"
            />
            <div
              class="absolute -bottom-14 left-2 w-72 rounded-lg bg-forest-deep p-5 text-on-forest-deep"
              style="box-shadow: var(--shadow-dialog)"
            >
              <div class="flex items-center justify-between">
                <span class="text-body-strong">{{ 'home.heroCycleTitle' | t }}</span>
                <span class="rounded-full bg-clay/20 px-2 py-0.5 text-caption text-clay">{{ 'home.heroCycleDay' | t: { count: 34 } }}</span>
              </div>
              <div class="mt-4 flex flex-col gap-3">
                <div class="flex items-center gap-3">
                  <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-clay">
                    <bt-icon name="check" [size]="12" class="text-on-clay" />
                  </span>
                  <span class="text-small">{{ 'home.heroCycleStep1' | t }}</span>
                </div>
                <div class="flex items-center gap-3">
                  <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-clay">
                    <bt-icon name="check" [size]="12" class="text-on-clay" />
                  </span>
                  <span class="text-small">{{ 'home.heroCycleStep2' | t }}</span>
                </div>
                <div class="flex items-center gap-3">
                  <span class="h-5 w-5 shrink-0 rounded-full border-2 border-on-forest-deep/40"></span>
                  <span class="text-small text-on-forest-deep/70">{{ 'home.heroCycleStep3' | t }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>

    <section id="how" class="mx-auto max-w-content-max px-5 py-16 sm:px-18">
      <div class="flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
        <div class="flex flex-col gap-2.5">
          <span class="text-eyebrow text-clay-ink">{{ 'home.howItWorksEyebrow' | t }}</span>
          <h2 class="max-w-xl text-heading-1">{{ 'home.howItWorksHeadline' | t }}</h2>
        </div>
        <bt-button variant="outline" href="/marketplace" class="w-fit">{{ 'home.howItWorksCta' | t }}</bt-button>
      </div>
      <div class="mt-8 grid gap-6 sm:grid-cols-3">
        <div class="flex flex-col gap-2 border-t-[3px] border-forest pt-6">
          <span class="text-number-xl text-forest">01</span>
          <h3 class="text-heading-4">{{ 'home.step1Title' | t }}</h3>
          <p class="text-body text-ink-muted">{{ 'home.step1Body' | t }}</p>
        </div>
        <div class="flex flex-col gap-2 border-t-[3px] border-forest pt-6">
          <span class="text-number-xl text-forest">02</span>
          <h3 class="text-heading-4">{{ 'home.step2Title' | t }}</h3>
          <p class="text-body text-ink-muted">{{ 'home.step2Body' | t }}</p>
        </div>
        <div class="flex flex-col gap-2 border-t-[3px] border-clay pt-6">
          <span class="text-number-xl text-clay-ink">03</span>
          <h3 class="text-heading-4">{{ 'home.step3Title' | t }}</h3>
          <p class="text-body text-ink-muted">{{ 'home.step3Body' | t }}</p>
        </div>
      </div>
    </section>

    <section class="mx-auto hidden max-w-content-max px-5 py-8 sm:block sm:px-18">
      <span class="text-eyebrow text-clay-ink">{{ 'home.featuredEyebrow' | t }}</span>
      <h2 class="mt-2.5 text-heading-2">{{ 'home.featuredTitle' | t }}</h2>
      @if (featured.value()?.items?.length) {
        <div class="mt-6 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          @for (item of featured.value()!.items.slice(0, 4); track item.listing.id) {
            <bt-listing-card [item]="item" />
          }
        </div>
      } @else if (!featured.isLoading()) {
        <div class="mt-6 flex flex-col items-start gap-3 rounded-lg border border-dashed border-line-strong p-8">
          <p class="text-body text-ink-muted">{{ 'home.featuredEmpty' | t }}</p>
          <bt-button variant="outline" size="sm" [href]="farmerCta().link" [queryParams]="farmerCta().query">
            {{ 'home.featuredEmptyCta' | t }}
          </bt-button>
        </div>
      }
    </section>

    @defer (on viewport) {
      <section class="relative mx-5 overflow-hidden rounded-xl bg-forest-deep py-16 text-on-forest-deep sm:mx-18">
        <div aria-hidden="true" class="animal-pattern text-on-forest-deep opacity-[0.08]"></div>
        <div class="relative mx-auto grid max-w-content-max gap-10 px-5 sm:grid-cols-12 sm:px-13">
          <div class="flex flex-col gap-4 sm:col-span-5">
            <span class="text-eyebrow text-on-forest-deep/70">{{ 'home.trustEyebrow' | t }}</span>
            <h2 class="text-heading-2">{{ 'home.trustHeadline' | t }}</h2>
            <p class="text-body text-on-forest-deep/80">{{ 'home.trustBody' | t }}</p>
          </div>
          <div class="grid gap-4 sm:col-span-7 sm:grid-cols-2">
            <div class="flex flex-col gap-2 rounded-lg border border-on-forest-deep/15 bg-on-forest-deep/5 p-5">
              <bt-icon name="shield" [size]="24" class="text-clay" />
              <span class="text-body-strong">{{ 'home.trustVerified' | t }}</span>
              <p class="text-small text-on-forest-deep/70">{{ 'home.trustVerifiedDesc' | t }}</p>
            </div>
            <div class="flex flex-col gap-2 rounded-lg border border-on-forest-deep/15 bg-on-forest-deep/5 p-5">
              <bt-icon name="star" [size]="24" class="text-clay" />
              <span class="text-body-strong">{{ 'home.trustReviews' | t }}</span>
              <p class="text-small text-on-forest-deep/70">{{ 'home.trustReviewsDesc' | t }}</p>
            </div>
            <div class="flex flex-col gap-2 rounded-lg border border-on-forest-deep/15 bg-on-forest-deep/5 p-5">
              <bt-icon name="lock" [size]="24" class="text-clay" />
              <span class="text-body-strong">{{ 'home.trustEscrow' | t }}</span>
              <p class="text-small text-on-forest-deep/70">{{ 'home.trustEscrowDesc' | t }}</p>
            </div>
            <div class="flex flex-col gap-2 rounded-lg border border-on-forest-deep/15 bg-on-forest-deep/5 p-5">
              <bt-icon name="flag" [size]="24" class="text-clay" />
              <span class="text-body-strong">{{ 'home.trustReport' | t }}</span>
              <p class="text-small text-on-forest-deep/70">{{ 'home.trustReportDesc' | t }}</p>
            </div>
          </div>
        </div>
      </section>

      @if (forumTeaser.value()?.length) {
        <section class="mx-auto grid max-w-content-max gap-10 px-5 py-16 sm:grid-cols-12 sm:px-18">
          <div class="flex flex-col gap-4 sm:col-span-4">
            <span class="text-eyebrow text-clay-ink">{{ 'home.forumEyebrow' | t }}</span>
            <h2 class="text-heading-2">{{ 'home.forumTitle' | t }}</h2>
            <p class="text-body text-ink-muted">{{ 'home.forumBody' | t }}</p>
            <bt-button variant="solid" href="/forum" class="w-fit">{{ 'home.forumCta' | t }}</bt-button>
          </div>
          <div class="flex flex-col border-t border-line sm:col-span-8">
            @for (post of forumTeaser.value()!.slice(0, 3); track post.id) {
              <a
                [routerLink]="['/forum', post.id]"
                class="flex items-center gap-4 border-b border-line py-5 text-ink no-underline hover:text-forest"
              >
                <bt-badge [tone]="categoryTone(post.category)">{{ reference.forumLabelKey(post.category) | t }}</bt-badge>
                <span class="flex-1 text-body-strong">{{ post.title }}</span>
                <span class="flex items-center gap-1.5 text-small text-ink-muted">
                  <bt-icon name="message" [size]="16" />
                  {{ 'home.forumReplyCount' | t: { count: post.replies?.length ?? 0 } }}
                </span>
                <bt-icon name="chevron-right" [size]="18" class="text-ink-muted" />
              </a>
            }
          </div>
        </section>
      }
    } @placeholder {
      <div class="h-40"></div>
    }

    <section
      id="eleveurs"
      class="relative mx-5 my-10 overflow-hidden rounded-2xl bg-clay px-6 py-12 text-on-clay sm:mx-18 sm:my-16 sm:px-14 sm:py-16"
    >
      <div aria-hidden="true" class="animal-pattern text-on-clay opacity-10"></div>
      <div class="relative flex flex-col gap-8 lg:flex-row lg:items-center lg:justify-between lg:gap-12">
        <div class="flex max-w-2xl flex-col gap-3.5">
          <span class="text-eyebrow text-on-clay/70">{{ 'home.farmerCtaEyebrow' | t }}</span>
          <h2 class="text-heading-1">{{ 'home.farmerCtaTitle' | t }}</h2>
          <p class="text-body-l text-on-clay/90">{{ 'home.farmerCtaBody' | t }}</p>
          <ol class="mt-2 flex flex-col gap-2.5 sm:flex-row sm:flex-wrap sm:gap-x-6">
            @for (step of farmerSteps; track step; let i = $index) {
              <li class="flex items-center gap-2.5 text-body-strong" [class.opacity-60]="i < farmerStep()">
                @if (i < farmerStep()) {
                  <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-ink text-surface">
                    <bt-icon name="check" [size]="14" />
                  </span>
                  <span class="line-through decoration-2">{{ step | t }}</span>
                  <span class="sr-only">({{ 'home.farmerCtaStepDone' | t }})</span>
                } @else {
                  <span
                    class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full border-2 border-ink text-small font-bold"
                    [class.bg-ink]="i === farmerStep()"
                    [class.text-surface]="i === farmerStep()"
                  >
                    {{ i + 1 }}
                  </span>
                  {{ step | t }}
                }
              </li>
            }
          </ol>
        </div>
        <a
          [routerLink]="farmerCta().link"
          [queryParams]="farmerCta().query"
          class="inline-flex h-control-xl w-fit shrink-0 items-center justify-center gap-2 rounded-full bg-ink px-7 text-body-strong text-surface no-underline"
        >
          <bt-icon [name]="farmerCta().icon" [size]="18" />
          {{ farmerCta().labelKey | t }}
        </a>
      </div>
    </section>
  `
})
export class HomeComponent {
  private readonly listingsApi = inject(ListingsApi);
  private readonly forumApi = inject(ForumApi);
  private readonly router = inject(Router);
  private readonly locale = inject(LocaleService);
  private readonly auth = inject(AuthService);
  private readonly seller = inject(SellerSpaceService);

  protected readonly groups = speciesGroups();
  protected readonly speciesLabel = speciesLabelKey;
  protected readonly categoryTone = categoryTone;
  protected readonly reference = inject(ReferenceService);
  protected readonly species = signal('');
  protected readonly query = signal('');
  protected readonly city = signal('');
  protected readonly month = signal('');

  /** This month and the next five: "Disponible dès" from the design. */
  protected readonly monthOptions = computed(() => {
    const lang = this.locale.locale();
    const now = new Date();
    return Array.from({ length: 6 }, (_, i) => {
      const d = new Date(now.getFullYear(), now.getMonth() + i, 1);
      const value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
      const label = new Intl.DateTimeFormat(lang, { month: 'long', year: 'numeric' }).format(d);
      return { value, label: label.charAt(0).toUpperCase() + label.slice(1) };
    });
  });

  /** Selling takes an account, then a farm, then listings: the farmer
   * section shows where this visitor stands and its button the next
   * step. `/sell` itself opens the farm form or the listing wizard. */
  protected readonly farmerSteps = ['home.farmerCtaStepAccount', 'home.farmerCtaStepFarm', 'home.farmerCtaStepPost'];
  protected readonly farmerStep = computed(() => (!this.auth.isAuthenticated() ? 0 : this.seller.hasFarm() ? 2 : 1));
  protected readonly farmerCta = computed<{ labelKey: string; icon: IconName; link: string; query?: Record<string, string> }>(() => {
    switch (this.farmerStep()) {
      case 0:
        return { labelKey: 'home.farmerCtaRegister', icon: 'user', link: '/register', query: { redirectTo: '/sell' } };
      case 1:
        return { labelKey: 'home.farmerCtaFarm', icon: 'farm', link: '/sell' };
      default:
        return { labelKey: 'home.farmerCtaPost', icon: 'plus', link: '/sell' };
    }
  });

  protected readonly featured = apiResource<ListingSearchResult>(() =>
    this.listingsApi.searchUrl({ pageSize: 8, sort: 'recent' })
  );

  protected readonly topRated = apiResource<ListingSearchResult>(() => this.listingsApi.searchUrl({ pageSize: 1, sort: 'rating' }));

  protected readonly forumTeaser = apiResource<ForumPost[]>(() => this.forumApi.postsUrl({ sort: 'recent' }));

  protected searchQuery(): void {
    this.router.navigate(['/marketplace'], { queryParams: { q: this.query().trim() || null } });
  }

  /** The town goes in the word search, which matches farms' towns and
   * regions, so "Bafoussam" or "Ouest" both work. */
  protected search(): void {
    this.router.navigate(['/marketplace'], {
      queryParams: {
        category: this.species().startsWith('category:') ? this.species().slice('category:'.length) : null,
        species: this.species() && !this.species().startsWith('category:') ? this.species() : null,
        q: this.city().trim() || null,
        available: this.month() || null
      }
    });
  }
}
