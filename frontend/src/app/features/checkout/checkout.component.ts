import { NgTemplateOutlet } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, linkedSignal, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { catchError, debounceTime, map, of, switchMap } from 'rxjs';
import { PaymentMethod } from '../../domain/enums';
import { ListingDetail, ReservationQuote } from '../../domain/models';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { ReservationsApi } from '../../data-access/reservations/reservations.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { LocaleService } from '../../core/i18n/locale.service';
import { apiResource } from '../../core/http/api-resource';
import { PAYMENT_METHOD_LABEL_KEYS, PAYMENT_METHOD_VALUES } from '../../domain/rules/payment-method';
import { ChoiceCardComponent } from '../../shared/ui/choice-card/choice-card.component';
import { QuantityStepperComponent } from '../../shared/ui/quantity-stepper/quantity-stepper.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { StickyActionBarComponent } from '../../shared/ui/sticky-action-bar/sticky-action-bar.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { ThumbSrcDirective } from '../../shared/ui/thumb-src/thumb-src.directive';

const PHONE_STORAGE_KEY = 'betailo.momoPhone';
const QUOTE_DEBOUNCE_MS = 250;

/** Brand marks for the payment cards: fixed brand colours, not theme
 * tokens, so they read the same in light and dark mode. */
const BRAND: Readonly<Record<PaymentMethod, { label: string; class: string }>> = {
  mtn_momo: { label: 'MoMo', class: 'bg-[#ffcc00] text-[#16211c]' },
  orange_money: { label: 'OM', class: 'bg-[#16211c] text-[#ff7900]' },
  card: { label: 'CB', class: 'bg-sky-soft text-sky' }
};

type QuoteState = { quote: ReservationQuote; error: null } | { quote: null; error: AppError } | null;

function readStoredPhone(): string {
  try {
    return localStorage.getItem(PHONE_STORAGE_KEY) ?? '';
  } catch {
    return '';
  }
}

/** Checkout, per the mobile design: quantity, pay-now vs reserve-later,
 * payment method and Mobile Money number, with the total and the pay
 * button in a sticky bar. Every number shown (fee, total, bounds) comes
 * from the API's quote for the current quantity; the client never
 * computes the fee itself. */
@Component({
  selector: 'bt-checkout',
  imports: [ThumbSrcDirective, 
    NgTemplateOutlet,
    FormsModule,
    ChoiceCardComponent,
    QuantityStepperComponent,
    NoticeComponent,
    ButtonComponent,
    IconComponent,
    StickyActionBarComponent,
    FcfaCurrencyPipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (detail.value(); as d) {
      <div class="mx-auto max-w-content-max px-4 pb-32 pt-4 sm:px-8 lg:px-18 lg:pb-16 lg:pt-10">
        <header class="flex items-center gap-3">
          <button
            type="button"
            (click)="back(d.listing.id)"
            [attr.aria-label]="'common.back' | t"
            class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-line bg-surface-card"
          >
            <bt-icon name="chevron-left" [size]="22" />
          </button>
          <h1 class="text-heading-2">{{ 'checkout.title' | t }}</h1>
        </header>

        <div class="mt-5 grid gap-8 lg:grid-cols-[minmax(0,1fr)_400px]">
          <form class="flex flex-col gap-6" (submit)="$event.preventDefault(); submit(d)">
            <div class="flex items-center gap-3 rounded-lg border border-line bg-surface-card p-3">
              <div class="h-16 w-16 shrink-0 overflow-hidden rounded-md bg-surface-sunken">
                @if (d.listing.photos?.[0]; as photo) {
                  <img [btThumbSrc]="d.listing.thumbs?.[0]" [photo]="photo" alt="" decoding="async" class="h-full w-full object-cover" />
                }
              </div>
              <div class="flex min-w-0 flex-col">
                <span class="line-clamp-2 text-body-strong">{{ d.listing.title }}</span>
                <span class="text-small text-ink-muted">
                  {{ d.farm.name }}
                  @if (d.reviewSummary.count) {
                    · ★ {{ d.reviewSummary.average.toFixed(1) }}
                  }
                </span>
              </div>
            </div>

            <section class="flex flex-col gap-2">
              <span class="text-body-strong">{{ 'checkout.quantityLabel' | t }}</span>
              <bt-quantity-stepper [(value)]="quantity" [min]="bounds().min" [max]="bounds().max" />
              <span class="text-small text-ink-muted">
                {{ 'checkout.placesLeft' | t: { count: d.cycle.availableQuantity } }}
                @if (bounds().min > 1) {
                  · {{ 'checkout.minimum' | t: { count: bounds().min } }}
                }
              </span>
              @if (quoteError(); as err) {
                <span class="text-small text-alert" role="alert">{{ err }}</span>
              }
            </section>

            <section class="flex flex-col gap-3" role="radiogroup" [attr.aria-label]="'checkout.modeLabel' | t">
              <span class="text-body-strong">{{ 'checkout.modeLabel' | t }}</span>
              <bt-choice-card
                name="mode"
                value="now"
                [title]="'priorityChoice.payNowTitle' | t"
                [description]="'checkout.payNowDesc' | t"
                [badge]="'checkout.priorityBadge' | t"
                tone="forest"
                [checked]="payNow()"
                (selected)="payNow.set(true)"
              />
              <bt-choice-card
                name="mode"
                value="later"
                [title]="'priorityChoice.laterTitle' | t"
                [description]="'checkout.laterDesc' | t"
                [checked]="!payNow()"
                (selected)="payNow.set(false)"
              />
            </section>

            @if (payNow()) {
              <section class="flex flex-col gap-3" role="radiogroup" [attr.aria-label]="'checkout.paymentMethodLabel' | t">
                <span class="text-body-strong">{{ 'checkout.paymentMethodLabel' | t }}</span>
                @for (m of methods; track m) {
                  <bt-choice-card name="method" [value]="m" [title]="methodLabelKeys[m] | t" [checked]="method() === m" (selected)="method.set(m)">
                    <span leading class="flex h-10 w-12 shrink-0 items-center justify-center rounded-md text-caption font-bold" [class]="brand[m].class">
                      {{ brand[m].label }}
                    </span>
                  </bt-choice-card>
                }
              </section>

              @if (method() !== 'card') {
                <section class="flex flex-col gap-2">
                  <label for="momo-phone" class="text-body-strong">{{ 'checkout.momoNumberLabel' | t }}</label>
                  <div class="flex gap-2">
                    <span class="flex h-control-lg items-center rounded-sm bg-surface-sunken px-4 text-body-strong">+237</span>
                    <input
                      id="momo-phone"
                      type="tel"
                      inputmode="tel"
                      autocomplete="tel-national"
                      placeholder="6 90 12 34 56"
                      [ngModel]="phone()"
                      (ngModelChange)="phone.set($event)"
                      name="phone"
                      [attr.aria-invalid]="phoneInvalid() || null"
                      aria-describedby="momo-phone-help"
                      class="h-control-lg min-w-0 flex-1 rounded-sm border border-line-strong bg-surface-card px-4 text-body text-ink outline-none focus:border-forest"
                      [class.border-alert]="phoneInvalid()"
                    />
                  </div>
                  @if (phoneInvalid()) {
                    <span id="momo-phone-help" role="alert" class="text-small text-alert">{{ 'checkout.momoNumberInvalid' | t }}</span>
                  } @else {
                    <span id="momo-phone-help" class="text-small text-ink-muted">{{ 'checkout.momoNumberHint' | t }}</span>
                  }
                </section>
              }
            }

            <bt-notice tone="success" icon="lock">{{ 'checkout.escrowNotice' | t }}</bt-notice>
            <!-- Last reminder before paying: the escrow covers the
                 animals only, transport is settled with the farmer. -->
            <bt-notice tone="info" icon="pin" [title]="'checkout.transportTitle' | t">{{ 'checkout.transportBody' | t }}</bt-notice>

            <!-- Desktop submit; phones use the sticky bar. -->
            <div class="hidden lg:block">
              <bt-button type="submit" size="lg" block [disabled]="!canSubmit()">{{ submitLabel() }}</bt-button>
            </div>
          </form>

          <aside class="hidden lg:block">
            <div class="sticky top-24 flex flex-col gap-3 rounded-lg border border-line bg-surface-card p-6" style="box-shadow: var(--shadow-float)">
              <ng-container [ngTemplateOutlet]="summary" />
            </div>
          </aside>
        </div>
      </div>

      <bt-sticky-action-bar>
        <div class="flex w-full flex-col gap-2">
          <div class="flex items-baseline justify-between gap-3 text-small">
            <span class="text-ink-muted">{{ 'checkout.totalWithFee' | t: { rate: feePercent() } }}</span>
            <span class="text-amount text-body-strong">{{ (quote()?.total ?? 0) | fcfa }}</span>
          </div>
          <bt-button size="lg" block [disabled]="!canSubmit()" (click)="submit(d)">{{ submitLabel() }}</bt-button>
        </div>
      </bt-sticky-action-bar>

      <ng-template #summary>
        <div class="flex justify-between text-body">
          <span class="text-ink-muted">{{ 'checkout.subtotalLabel' | t }}</span>
          <span class="text-amount">{{ (quote()?.subtotal ?? 0) | fcfa }}</span>
        </div>
        <div class="flex justify-between text-body">
          <span class="text-ink-muted">{{ 'checkout.feeLabel' | t: { rate: feePercent() } }}</span>
          <span class="text-amount">{{ (quote()?.serviceFee ?? 0) | fcfa }}</span>
        </div>
        <div class="flex justify-between border-t border-line pt-3 text-body-strong">
          <span>{{ 'checkout.totalLabel' | t }}</span>
          <span class="text-amount">{{ (quote()?.total ?? 0) | fcfa }}</span>
        </div>
      </ng-template>
    } @else if (detail.isLoading()) {
      <div class="mx-auto flex max-w-lg flex-col gap-4 px-4 py-10" aria-busy="true">
        @for (h of [64, 120, 160]; track h) {
          <div class="animate-pulse rounded-lg bg-surface-sunken" [style.height.px]="h"></div>
        }
      </div>
    } @else if (detail.error()) {
      <div class="mx-auto flex max-w-md flex-col items-center gap-4 px-5 py-20 text-center">
        <p class="text-body">{{ 'checkout.unavailable' | t }}</p>
        <bt-button variant="outline" href="/marketplace">{{ 'listingDetail.backToMarket' | t }}</bt-button>
      </div>
    }
  `
})
export class CheckoutComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly listingsApi = inject(ListingsApi);
  private readonly reservationsApi = inject(ReservationsApi);
  private readonly toast = inject(ToastService);

  protected readonly methods = PAYMENT_METHOD_VALUES;
  protected readonly methodLabelKeys = PAYMENT_METHOD_LABEL_KEYS;
  protected readonly brand = BRAND;

  private readonly listingId = toSignal(this.route.paramMap.pipe(map((p) => p.get('listingId') ?? '')), { initialValue: '' });

  protected readonly detail = apiResource<ListingDetail>(() =>
    this.listingId() ? this.listingsApi.fullDetailUrl(this.listingId()) : undefined
  );

  protected readonly quantity = signal(1);
  /** Starts on the waiting list when the lot is full (`?waitlist=1`). */
  protected readonly payNow = signal(this.route.snapshot.queryParamMap.get('waitlist') !== '1');
  protected readonly method = signal<PaymentMethod>('mtn_momo');
  protected readonly phone = signal(readStoredPhone());
  protected readonly submitting = signal(false);

  /** The quote follows the quantity (debounced), so the stepper stays
   * instant while the totals catch up a beat later. */
  private readonly quoteState = toSignal<QuoteState>(
    toObservable(computed(() => ({ id: this.listingId(), q: this.quantity(), payNow: this.payNow() }))).pipe(
      debounceTime(QUOTE_DEBOUNCE_MS),
      switchMap(({ id, q, payNow }) =>
        id
          ? this.reservationsApi.quote(id, q, payNow).pipe(
              map((quote): QuoteState => ({ quote, error: null })),
              catchError((error: AppError) => of<QuoteState>({ quote: null, error }))
            )
          : of(null)
      )
    ),
    { initialValue: null }
  );

  /** The latest good quote, kept while a new one loads or after an
   * error so the bounds and totals don't flash to zero. */
  protected readonly quote = linkedSignal<QuoteState, ReservationQuote | null>({
    source: this.quoteState,
    computation: (s, previous) => s?.quote ?? previous?.value ?? null
  });
  protected readonly quoteError = computed(() => this.quoteState()?.error?.message ?? null);

  protected readonly bounds = computed(() => {
    const q = this.quote();
    const available = this.detail.value()?.cycle.availableQuantity ?? 1;
    return { min: q?.min ?? 1, max: Math.max(1, q?.max ?? available) };
  });

  protected readonly feePercent = computed(() => Math.round((this.quote()?.feeRate ?? 0.02) * 100));

  /** A light format check for instant feedback; the API is the judge. */
  protected readonly phoneInvalid = computed(() => {
    const digits = this.phone().replace(/\D/g, '').replace(/^(00)?237/, '');
    return this.phone().trim() !== '' && !/^6\d{8}$/.test(digits);
  });

  private readonly needsPhone = computed(() => this.payNow() && this.method() !== 'card');

  protected readonly canSubmit = computed(
    () =>
      !this.submitting() &&
      !!this.quote() &&
      !this.quoteError() &&
      (!this.needsPhone() || (this.phone().trim() !== '' && !this.phoneInvalid()))
  );

  protected readonly submitLabel = computed(() => {
    const total = this.quote()?.total ?? 0;
    if (this.submitting()) return this.t('checkout.submitting');
    return this.payNow()
      ? this.t('checkout.payAmount', { amount: this.fcfa.transform(total) })
      : this.t('checkout.submitLater');
  });

  private readonly locale = inject(LocaleService);
  private readonly fcfa = new FcfaCurrencyPipe();

  private t(key: string, params?: Record<string, string | number>): string {
    return this.locale.t(key, params);
  }

  protected back(listingId: string): void {
    if (this.router.lastSuccessfulNavigation()?.previousNavigation) {
      history.back();
    } else {
      this.router.navigate(['/marketplace', listingId]);
    }
  }

  protected submit(d: ListingDetail): void {
    if (!this.canSubmit()) return;
    this.submitting.set(true);
    const payNow = this.payNow();
    if (this.needsPhone()) {
      try {
        localStorage.setItem(PHONE_STORAGE_KEY, this.phone());
      } catch {
        // Remembering the number is a convenience only.
      }
    }
    this.reservationsApi
      .create(d.listing.id, {
        quantity: this.quantity(),
        payNow,
        paymentMethod: payNow ? this.method() : undefined,
        payerPhone: this.needsPhone() ? this.phone() : undefined
      })
      .subscribe({
        next: ({ reservation, paymentCaptured }) => {
          if (payNow && !paymentCaptured) {
            this.toast.show({ tone: 'warning', title: 'checkout.paymentFailedTitle', detail: 'checkout.paymentFailedDetail' });
          } else {
            this.toast.show({
              tone: 'success',
              title: reservation.priority ? 'checkout.confirmedTitle' : 'checkout.pendingTitle',
              detail: reservation.priority ? 'checkout.confirmedDetail' : 'checkout.pendingDetail'
            });
          }
          this.router.navigate(['/account/reservations'], { queryParams: { highlight: reservation.id } });
        },
        error: (err: AppError) => {
          this.submitting.set(false);
          this.toast.show({ tone: 'danger', title: 'checkout.failedTitle', detail: err.message });
        }
      });
  }
}
