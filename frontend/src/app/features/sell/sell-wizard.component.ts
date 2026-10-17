import { ChangeDetectionStrategy, Component, ElementRef, computed, effect, inject, signal, untracked, viewChildren } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { Observable, of, switchMap } from 'rxjs';
import { Listing, PriceHint } from '../../domain/models';
import { categoryOf, farmSpeciesGroups, speciesLabelKey, speciesUnitKey } from '../../domain/reference/species';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { ListingPhotosService } from '../../data-access/listings/listing-photos.service';
import { AuthService } from '../../core/auth/auth.service';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { PlacePipe } from '../../core/i18n/place.pipe';
import { apiResource } from '../../core/http/api-resource';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { PhotoPickerComponent, PickedPhoto } from '../../shared/ui/photo-picker/photo-picker.component';
import { QuantityStepperComponent } from '../../shared/ui/quantity-stepper/quantity-stepper.component';
import { SelectFieldComponent, SelectOption } from '../../shared/ui/select-field/select-field.component';
import { StickyActionBarComponent } from '../../shared/ui/sticky-action-bar/sticky-action-bar.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { SellDraft, EMPTY_DRAFT, clearDraft, isEmptyDraft, loadDraft, saveDraft } from './sell-draft';
import { SellerOnboardingComponent } from './seller-onboarding.component';

const AGE_WEEKS = [1, 2, 3, 4, 6, 8, 10, 12, 16, 20, 26, 36, 52];
const MAX_QUANTITY = 100_000;

type Section = 0 | 1 | 2 | 3;

/** "Vendre mes animaux", per mobile design 6: one scrolling page in four
 * numbered sections (photos, which animals, how many and at what price,
 * when and where), a progress bar that fills as each is completed, a
 * live preview of the card buyers will see, and Brouillon / Publier in
 * the thumb zone. Every field is remembered on this device as it's
 * typed, so an interrupted farmer picks up where they left off.
 *
 * The API does the real work in one call (cycle + draft listing, type
 * and unit derived from the species), then photos are attached and, for
 * "Publier", the listing is published: the API decides whether it can
 * be (photos present, cycle open). Anyone who isn't a seller yet
 * presents their farm first. */
@Component({
  selector: 'bt-sell-wizard',
  imports: [
    FormsModule,
    ButtonComponent,
    IconComponent,
    NoticeComponent,
    PhotoPickerComponent,
    QuantityStepperComponent,
    SelectFieldComponent,
    StickyActionBarComponent,
    TextFieldComponent,
    FcfaCurrencyPipe,
    TranslatePipe, PlacePipe,
    SellerOnboardingComponent,
    RouterLink
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-2xl px-4 pb-36 pt-4 sm:px-8 lg:pb-16 lg:pt-10">
      <header class="flex items-center gap-3">
        <button
          type="button"
          (click)="leave()"
          [attr.aria-label]="'common.close' | t"
          class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full border border-line bg-surface-card"
        >
          <bt-icon name="x" [size]="22" />
        </button>
        <div class="flex min-w-0 flex-col">
          <h1 class="text-heading-2">{{ 'sell.title' | t }}</h1>
          <span class="text-small text-ink-muted" aria-live="polite">
            {{ 'sell.subtitle' | t }}
            @if (draftSaved()) {
              · {{ 'sell.draftSaved' | t }}
            }
          </span>
        </div>
      </header>

      @if (auth.isStaff()) {
        <bt-notice class="mt-6 block" tone="info">{{ 'sell.adminCannotSell' | t }}</bt-notice>
      } @else if (needsOnboarding()) {
        <div class="mt-6">
          <bt-seller-onboarding (created)="onFarmCreated()" />
        </div>
      } @else if (!farm()) {
        <div class="mt-6 flex flex-col gap-4" aria-busy="true">
          @for (h of [64, 220, 180]; track h) {
            <div class="animate-pulse rounded-lg bg-surface-sunken" [style.height.px]="h"></div>
          }
        </div>
      } @else {
        <div class="sticky top-0 z-10 -mx-4 mt-4 bg-surface px-4 py-3 sm:-mx-8 sm:px-8" role="progressbar" [attr.aria-valuenow]="doneCount()" aria-valuemin="0" aria-valuemax="4" [attr.aria-label]="'sell.progress' | t: { done: doneCount() }">
          <div class="grid grid-cols-4 gap-2">
            @for (s of sections; track s) {
              <span class="h-1.5 rounded-full transition-colors" [class]="segmentClass(s)"></span>
            }
          </div>
        </div>

        <form class="mt-4 flex flex-col gap-10" (submit)="$event.preventDefault(); save(true)">
          <!-- 1. Photos -->
          <section #section class="flex scroll-mt-20 flex-col gap-4">
            <h2 class="flex items-center gap-3 text-heading-3">
              <span [class]="badgeClass(0)">@if (complete()[0]) {<bt-icon name="check" [size]="14" />} @else {1}</span>
              {{ 'sell.photosTitle' | t }}
            </h2>
            <bt-photo-picker [(photos)]="photos" [max]="12" />
          </section>

          <!-- 2. Animals -->
          <section #section class="flex scroll-mt-20 flex-col gap-4">
            <h2 class="flex items-center gap-3 text-heading-3">
              <span [class]="badgeClass(1)">@if (complete()[1]) {<bt-icon name="check" [size]="14" />} @else {2}</span>
              {{ 'sell.animalsTitle' | t }}
            </h2>
            <!-- First the category, then what is sold in it. -->
            <div class="no-scrollbar -mx-1 flex gap-2 overflow-x-auto px-1" role="tablist" [attr.aria-label]="'sell.categoryLabel' | t">
              @for (g of groups(); track g.key) {
                <button
                  type="button"
                  role="tab"
                  [attr.aria-selected]="currentGroup()?.key === g.key"
                  (click)="pickedCategory.set(g.key)"
                  class="flex h-10 shrink-0 items-center gap-2 rounded-full border px-4 text-small font-semibold transition"
                  [class]="currentGroup()?.key === g.key ? 'border-forest bg-forest text-on-forest' : 'border-line-strong bg-surface-card text-ink'"
                >
                  <span class="h-2.5 w-2.5 rounded-full" [class]="g.dot"></span>
                  {{ g.labelKey | t }}
                </button>
              }
            </div>
            <div class="grid grid-cols-2 gap-2 sm:grid-cols-3" role="radiogroup" [attr.aria-label]="'sell.animalsTitle' | t">
              @for (s of currentGroup()?.species ?? []; track s) {
                <button
                  type="button"
                  role="radio"
                  [attr.aria-checked]="draft().species === s"
                  (click)="patch({ species: s })"
                  class="flex min-h-24 flex-col items-center justify-center gap-2 rounded-lg border-2 p-2 text-center text-small font-semibold transition active:scale-[0.98]"
                  [class]="draft().species === s ? 'border-forest bg-forest-soft text-on-forest-soft' : 'border-line bg-surface-card text-ink'"
                >
                  <span class="h-3 w-3 rounded-full" [class]="currentGroup()?.dot"></span>
                  {{ speciesLabel(s) | t }}
                </button>
              }
            </div>
            <a routerLink="/portal/farm" class="self-start text-small text-forest underline underline-offset-2">{{ 'sell.addSpecies' | t }}</a>
            <bt-text-field
              [label]="'sell.breedLabel' | t"
              [placeholder]="'sell.breedPlaceholder' | t"
              name="breed"
              [ngModel]="draft().breed"
              (ngModelChange)="patch({ breed: $event })"
            />
            <bt-select-field
              [label]="'sell.ageLabel' | t"
              [options]="ageOptions()"
              name="age"
              [ngModel]="draft().ageWeeks === null ? '' : '' + draft().ageWeeks"
              (ngModelChange)="patch({ ageWeeks: $event ? +$event : null })"
            />
          </section>

          <!-- 3. Quantity and price -->
          <section #section class="flex scroll-mt-20 flex-col gap-4">
            <h2 class="flex items-center gap-3 text-heading-3">
              <span [class]="badgeClass(2)">@if (complete()[2]) {<bt-icon name="check" [size]="14" />} @else {3}</span>
              {{ 'sell.priceTitle' | t }}
            </h2>
            <div class="flex flex-col gap-2">
              <span class="text-small font-bold text-ink">{{ 'sell.quantityLabel' | t }}</span>
              <bt-quantity-stepper [value]="draft().quantity" (valueChange)="patch({ quantity: $event })" [min]="1" [max]="maxQuantity" />
            </div>
            <bt-text-field
              [label]="(unitKey() ? 'sell.pricePerUnitLabel' : 'sell.priceLabel') | t: { unit: (unitKey() | t) }"
              inputmode="numeric"
              suffix="FCFA"
              name="price"
              [ngModel]="priceText()"
              (ngModelChange)="setPrice($event)"
            />
            @if (hint(); as h) {
              <div class="flex gap-3 rounded-lg bg-forest-soft p-3.5 text-small text-on-forest-soft" aria-live="polite">
                <bt-icon name="chart" [size]="18" class="mt-0.5 shrink-0 text-forest" />
                <span>
                  {{ (h.regional ? 'sell.hintNational' : 'sell.hintRegional') | t: { region: (farm()!.location | place), low: (h.low | fcfa), high: (h.high | fcfa) } }}
                </span>
              </div>
            }
          </section>

          <!-- 4. When and where -->
          <section #section class="flex scroll-mt-20 flex-col gap-4">
            <h2 class="flex items-center gap-3 text-heading-3">
              <span [class]="badgeClass(3)">@if (complete()[3]) {<bt-icon name="check" [size]="14" />} @else {4}</span>
              {{ 'sell.whenTitle' | t }}
            </h2>
            <div class="grid grid-cols-2 gap-1 rounded-full bg-surface-sunken p-1" role="radiogroup">
              @for (w of whenOptions; track w) {
                <button
                  type="button"
                  role="radio"
                  [attr.aria-checked]="draft().when === w"
                  (click)="patch({ when: w })"
                  class="h-12 rounded-full text-body-strong transition"
                  [class]="draft().when === w ? 'bg-surface-card text-ink shadow-sm' : 'text-ink-muted'"
                >
                  {{ 'sell.when.' + w | t }}
                </button>
              }
            </div>
            @if (draft().when === 'date') {
              <label class="flex flex-col gap-2">
                <span class="text-small font-bold text-ink">{{ 'sell.dateLabel' | t }}</span>
                <input
                  type="date"
                  [min]="today"
                  [value]="draft().date"
                  (change)="patch({ date: $any($event.target).value })"
                  class="h-14 rounded-md border border-line-strong bg-surface-card px-4 text-[16px] text-ink outline-none focus:border-forest"
                />
              </label>
            }
            <div class="flex items-center gap-3 rounded-lg border border-line bg-surface-card p-4">
              <bt-icon name="pin" [size]="22" class="shrink-0 text-forest" />
              <div class="flex min-w-0 flex-1 flex-col">
                <span class="truncate text-body-strong">{{ farm()!.location | place }}</span>
                <span class="text-small text-ink-muted">{{ 'sell.farmLocation' | t }}</span>
              </div>
              <a href="/portal/farm" (click)="$event.preventDefault(); router.navigateByUrl('/portal/farm')" class="flex h-11 items-center px-2 text-body-strong text-forest">
                {{ 'sell.change' | t }}
              </a>
            </div>
            <label class="flex min-h-12 cursor-pointer items-center gap-3">
              <input
                type="checkbox"
                [checked]="draft().delivery"
                (change)="patch({ delivery: !draft().delivery })"
                class="h-6 w-6 shrink-0 cursor-pointer rounded accent-forest"
              />
              <span class="text-body">{{ 'sell.deliveryLabel' | t }}</span>
            </label>
            <bt-text-field
              [label]="'sell.descriptionLabel' | t"
              [hint]="'sell.descriptionHint' | t"
              multiline
              name="description"
              [ngModel]="draft().description"
              (ngModelChange)="patch({ description: $event })"
            />
          </section>

          <!-- Buyer preview -->
          <section class="flex flex-col gap-3">
            <h2 class="text-eyebrow text-ink-muted">{{ 'sell.previewTitle' | t }}</h2>
            <div class="flex gap-4 rounded-xl border border-line bg-surface-card p-3">
              <div class="h-24 w-24 shrink-0 overflow-hidden rounded-lg bg-surface-sunken">
                @if (photos()[0]; as cover) {
                  <img [src]="cover.preview" alt="" class="h-full w-full object-cover" />
                } @else {
                  <div class="flex h-full w-full items-center justify-center text-ink-muted"><bt-icon name="camera" [size]="24" /></div>
                }
              </div>
              <div class="flex min-w-0 flex-col justify-center gap-1">
                <span class="line-clamp-2 text-body-strong">{{ title() || ('sell.previewEmptyTitle' | t) }}</span>
                <span class="truncate text-small text-ink-muted">{{ farm()!.name }} · {{ farm()!.location | place }}</span>
                <span class="text-body">
                  <strong class="text-heading-4">{{ draft().unitPrice ?? 0 | fcfa }}</strong>
                  <span class="text-small text-ink-muted"> · {{ 'sell.previewAvailable' | t: { count: draft().quantity } }}</span>
                </span>
              </div>
            </div>
          </section>

          <div class="hidden gap-3 lg:flex">
            <bt-button variant="outline" size="lg" [disabled]="saving()" (click)="save(false)">{{ 'sell.saveDraft' | t }}</bt-button>
            <bt-button class="flex-1" type="submit" size="lg" block [disabled]="saving()">{{ publishLabel() | t }}</bt-button>
          </div>
        </form>

        <bt-sticky-action-bar>
          <bt-button variant="outline" size="lg" [disabled]="saving()" (click)="save(false)">{{ 'sell.saveDraft' | t }}</bt-button>
          <bt-button class="flex-1" size="lg" block [disabled]="saving()" (click)="save(true)">{{ publishLabel() | t }}</bt-button>
        </bt-sticky-action-bar>
      }
    </div>
  `
})
export class SellWizardComponent {
  protected readonly auth = inject(AuthService);
  protected readonly router = inject(Router);
  private readonly seller = inject(SellerSpaceService);
  private readonly listingsApi = inject(ListingsApi);
  private readonly listingPhotos = inject(ListingPhotosService);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);

  protected readonly sections: readonly Section[] = [0, 1, 2, 3];
  /** Only what the farm declared: the API refuses anything else. */
  protected readonly groups = computed(() => farmSpeciesGroups(this.farm()?.species));
  protected readonly speciesLabel = speciesLabelKey;
  /** The category the farmer opened; until then, the one of the species
   * already picked, or the first. */
  protected readonly pickedCategory = signal('');
  protected readonly currentGroup = computed(() => {
    const key = this.pickedCategory() || categoryOf(this.draft().species) || this.groups()[0]?.key;
    return this.groups().find((g) => g.key === key);
  });
  protected readonly whenOptions = ['now', 'date'] as const;
  protected readonly maxQuantity = MAX_QUANTITY;
  protected readonly today = new Date().toISOString().slice(0, 10);

  private readonly sectionEls = viewChildren<ElementRef<HTMLElement>>('section');

  /** The listing goes to the farm the seller is working on. */
  protected readonly farm = computed(() => this.seller.currentFarm());
  /** No farm yet: open one first (and submit it for validation). Once
   * started it stays up until its last step is done, even though the
   * farm exists from its first step on. */
  private readonly onboarding = signal(false);
  protected readonly needsOnboarding = computed(() => this.onboarding() || (this.seller.loaded() && !this.seller.hasFarm()));

  private readonly userId = this.auth.user()?.id ?? 'anon';
  protected readonly draft = signal<SellDraft>(loadDraft(this.userId) ?? EMPTY_DRAFT);
  protected readonly draftSaved = signal(false);
  protected readonly photos = signal<PickedPhoto[]>([]);
  protected readonly saving = signal(false);
  protected readonly publishing = signal(false);

  protected readonly unitKey = computed(() => speciesUnitKey(this.draft().species));

  private readonly hintRes = apiResource<PriceHint | null>(() => {
    const species = this.draft().species;
    const farm = this.farm();
    return species && farm ? this.listingsApi.priceHintUrl(species, farm.location) : undefined;
  });
  protected readonly hint = computed(() => this.hintRes.value() ?? null);

  protected readonly ageOptions = computed<SelectOption[]>(() => [
    { value: '', label: this.locale.t('sell.ageUnknown') },
    ...AGE_WEEKS.map((w) => ({ value: String(w), label: this.locale.t('sell.ageWeeks', { count: w }) }))
  ]);

  protected readonly priceText = computed(() => {
    const p = this.draft().unitPrice;
    return p === null ? '' : new Intl.NumberFormat('fr-FR').format(p);
  });

  /** The title buyers see, built from what was picked: "Jeunes poulets
   * Brahma — disponibles". */
  protected readonly title = computed(() => {
    const d = this.draft();
    if (!d.species) return '';
    const name = [this.locale.t(speciesLabelKey(d.species)), d.breed.trim()].filter(Boolean).join(' ');
    if (d.when === 'date' && d.date) {
      const date = new Intl.DateTimeFormat(this.locale.locale(), { day: 'numeric', month: 'long' }).format(new Date(d.date + 'T00:00:00'));
      return this.locale.t('sell.titleFrom', { name, date });
    }
    return this.locale.t('sell.titleNow', { name });
  });

  protected readonly complete = computed<readonly boolean[]>(() => {
    const d = this.draft();
    return [
      this.photos().length > 0,
      !!d.species,
      d.quantity > 0 && (d.unitPrice ?? 0) > 0,
      d.when === 'now' || (d.when === 'date' && !!d.date)
    ];
  });
  protected readonly doneCount = computed(() => this.complete().filter(Boolean).length);
  private readonly current = computed(() => this.complete().findIndex((c) => !c));

  protected readonly publishLabel = computed(() => (this.publishing() ? 'sell.publishing' : 'sell.publish'));

  constructor() {
    effect(() => {
      if (this.seller.loaded() && !this.seller.hasFarm()) this.onboarding.set(true);
    });
    // Remember the form on this device as it's filled in.
    let first = true;
    effect(() => {
      const d = this.draft();
      if (first) {
        first = false;
        return;
      }
      untracked(() => {
        saveDraft(this.userId, d);
        this.draftSaved.set(!isEmptyDraft(d));
      });
    });
  }

  protected patch(change: Partial<SellDraft>): void {
    this.draft.update((d) => ({ ...d, ...change }));
  }

  protected setPrice(text: string): void {
    const digits = text.replace(/\D/g, '').slice(0, 9);
    this.patch({ unitPrice: digits ? parseInt(digits, 10) : null });
  }

  protected segmentClass(s: Section): string {
    if (this.complete()[s]) return 'bg-forest';
    return s === this.current() ? 'bg-clay' : 'bg-line';
  }

  protected badgeClass(s: Section): string {
    const base = 'flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-small font-bold ';
    return base + (this.complete()[s] ? 'bg-forest text-on-forest' : 'bg-ink text-surface');
  }

  protected onFarmCreated(): void {
    this.onboarding.set(false);
    this.seller.refresh();
    this.toast.show({ tone: 'success', title: 'sellerOnboarding.done' });
  }

  protected leave(): void {
    this.router.navigateByUrl(this.seller.hasFarm() ? '/portal' : '/');
  }

  /** Brouillon needs what the API needs to create the listing (species,
   * quantity, price); Publier needs every section. Whatever is missing
   * is scrolled to rather than a button sitting greyed out unexplained. */
  protected save(publish: boolean): void {
    const farm = this.farm();
    if (!farm || this.saving()) return;
    const required: Section[] = publish ? [0, 1, 2, 3] : [1, 2];
    const missing = required.find((s) => !this.complete()[s]);
    if (missing !== undefined) {
      this.sectionEls()[missing]?.nativeElement.scrollIntoView({ behavior: 'smooth', block: 'start' });
      this.toast.show({ tone: 'warning', title: `sell.missing.${missing}` });
      return;
    }

    const d = this.draft();
    this.saving.set(true);
    this.publishing.set(publish);
    let createdId: string | null = null;
    this.listingsApi
      .quickCreate(farm.id, {
        species: d.species,
        breed: d.breed.trim() || undefined,
        ageWeeks: d.ageWeeks ?? undefined,
        quantity: d.quantity,
        unitPrice: d.unitPrice!,
        title: this.title(),
        description: d.description.trim() || undefined,
        availableFrom: d.when === 'date' && d.date ? new Date(d.date + 'T00:00:00').toISOString() : undefined,
        deliveryAvailable: d.delivery
      })
      .pipe(
        switchMap(({ listing }) => {
          createdId = listing.id;
          return this.listingPhotos.save(listing.id, this.photos()).pipe(switchMap(() => this.afterPhotos(listing, publish)));
        })
      )
      .subscribe({
        next: (listing) => {
          clearDraft(this.userId);
          this.toast.show({ tone: 'success', title: publish ? 'sell.publishedToast' : 'sell.draftToast' });
          this.router.navigateByUrl(publish ? `/marketplace/${listing.id}` : '/portal/listings');
        },
        error: (err: AppError) => {
          this.saving.set(false);
          this.publishing.set(false);
          if (createdId && err.code === 'farm.not_approved') {
            // Saved as a draft; it goes live once staff approved the
            // farm. Not a failure, so it isn't shown as one.
            clearDraft(this.userId);
            this.toast.show({ tone: 'info', title: 'sell.pendingToast', detail: 'sell.pendingToastDetail' });
            this.router.navigateByUrl('/portal/listings');
          } else if (createdId) {
            // The listing exists as a draft: finish it from its page
            // instead of creating a duplicate on retry.
            clearDraft(this.userId);
            this.toast.show({ tone: 'warning', title: 'sell.partialToast', detail: err.message });
            this.router.navigateByUrl(`/portal/listings/${createdId}`);
          } else {
            this.toast.show({ tone: 'danger', title: 'sell.failedToast', detail: err.message });
          }
        }
      });
  }

  private afterPhotos(listing: Listing, publish: boolean): Observable<Listing> {
    return publish ? this.listingsApi.publish(listing.id) : of(listing);
  }
}
