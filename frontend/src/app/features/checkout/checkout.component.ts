import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { httpResource } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';
import { PaymentMethod } from '../../domain/enums';
import { Cycle, Listing } from '../../domain/models';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { CyclesApi } from '../../data-access/cycles/cycles.api';
import { ReservationsApi } from '../../data-access/reservations/reservations.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { priorityChoiceCopy } from '../../domain/rules/priority-choice';
import { PAYMENT_METHOD_LABEL_KEYS, PAYMENT_METHOD_VALUES } from '../../domain/rules/payment-method';
import { ChoiceCardComponent } from '../../shared/ui/choice-card/choice-card.component';
import { QuantityStepperComponent } from '../../shared/ui/quantity-stepper/quantity-stepper.component';
import { SelectFieldComponent } from '../../shared/ui/select-field/select-field.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';

@Component({
  selector: 'bt-checkout',
  imports: [
    FormsModule,
    ChoiceCardComponent,
    QuantityStepperComponent,
    SelectFieldComponent,
    NoticeComponent,
    ButtonComponent,
    DialogComponent,
    FcfaCurrencyPipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (listing.value(); as item) {
      <div class="mx-auto max-w-content-max px-5 py-8 sm:px-18">
        <h1 class="text-heading-1">{{ 'checkout.title' | t }}</h1>

        <ol class="mt-4 flex items-center gap-2 text-small text-ink-muted">
          <li class="flex items-center gap-2 text-forest">
            <span class="flex h-5 w-5 items-center justify-center rounded-full bg-forest text-caption text-on-forest-deep">1</span>
            {{ 'checkout.stepQuantity' | t }}
          </li>
          <span class="h-px w-4 bg-line-strong"></span>
          <li class="flex items-center gap-2 text-forest">
            <span class="flex h-5 w-5 items-center justify-center rounded-full bg-forest text-caption text-on-forest-deep">2</span>
            {{ 'checkout.stepReservation' | t }}
          </li>
          <span class="h-px w-4 bg-line-strong"></span>
          <li class="flex items-center gap-2">
            <span class="flex h-5 w-5 items-center justify-center rounded-full border border-line-strong text-caption">3</span>
            {{ 'checkout.stepConfirmation' | t }}
          </li>
        </ol>

        <div class="mt-8 grid gap-8 lg:grid-cols-[1fr_400px]">
          <div class="flex flex-col gap-6">
            @if (cycle.value(); as c) {
              <div>
                <span class="text-small text-ink-muted">{{ 'checkout.quantityLabel' | t }}</span>
                <div class="mt-2">
                  <bt-quantity-stepper [(value)]="quantity" [max]="c.availableQuantity" />
                </div>
              </div>
            }

            <div role="radiogroup" [attr.aria-label]="'checkout.modeLabel' | t" class="flex flex-col gap-3">
              <bt-choice-card
                name="payNow"
                value="true"
                [title]="payNowCopy.titleKey | t"
                [description]="'checkout.payNowDesc' | t"
                [checked]="payNow()"
                (selected)="payNow.set(true)"
              />
              <bt-choice-card
                name="payNow"
                value="false"
                [title]="laterCopy.titleKey | t"
                [description]="'checkout.laterDesc' | t"
                tone="amber"
                [checked]="!payNow()"
                (selected)="payNow.set(false)"
              />
            </div>

            <bt-notice [tone]="payNow() ? 'success' : 'warning'" [title]="currentCopy().noticeTitleKey | t">
              {{ currentCopy().noticeBodyKey | t }}
            </bt-notice>

            @if (payNow()) {
              <bt-select-field
                [label]="'checkout.paymentMethodLabel' | t"
                [options]="paymentMethodOptions()"
                [ngModel]="paymentMethod()"
                (ngModelChange)="paymentMethod.set($event)"
              />
            }

            <bt-button size="lg" block [disabled]="submitting()" (click)="submit(item.id)">
              {{ (submitting() ? 'checkout.submitting' : payNow() ? 'checkout.submitPayNow' : 'checkout.submitLater') | t }}
            </bt-button>
          </div>

          <aside class="flex flex-col gap-4">
            <div class="flex flex-col gap-4 rounded-lg border border-line bg-surface-card p-6" style="box-shadow: var(--shadow-float)">
              <div class="flex gap-3.5">
                @if (item.photos?.[0]; as photo) {
                  <img [src]="photo" [alt]="item.title" class="h-[72px] w-[72px] shrink-0 rounded-md object-cover" />
                }
                <div class="flex flex-col gap-1">
                  <p class="text-heading-4">{{ item.title }}</p>
                  <p class="text-small text-ink-muted">{{ item.unitPrice | fcfa }} @if (item.unit; as unit) { / {{ unit }} }</p>
                </div>
              </div>
              <div class="flex items-center justify-between border-t border-line pt-4 text-body-strong">
                <span>{{ 'checkout.subtotalLabel' | t }}</span>
                <span class="text-amount">{{ subtotal() | fcfa }}</span>
              </div>
              <p class="text-small text-ink-muted">{{ 'checkout.feeNote' | t }}</p>
            </div>

            <div class="flex flex-col gap-4 rounded-lg bg-forest-deep p-6 text-on-forest-deep">
              <p class="text-body-strong">{{ 'checkout.protectionTitle' | t }}</p>
              <ol class="flex flex-col">
                <li class="flex gap-3">
                  <span class="flex flex-col items-center">
                    <span class="h-3.5 w-3.5 shrink-0 rounded-full bg-clay"></span>
                    <span class="w-px flex-1 bg-on-forest-deep/20"></span>
                  </span>
                  <span class="pb-4 text-small text-on-forest-deep/80">{{ 'checkout.protectionStep1' | t }}</span>
                </li>
                <li class="flex gap-3">
                  <span class="flex flex-col items-center">
                    <span class="h-3.5 w-3.5 shrink-0 rounded-full border-2 border-on-forest-deep/40"></span>
                    <span class="w-px flex-1 bg-on-forest-deep/20"></span>
                  </span>
                  <span class="pb-4 text-small text-on-forest-deep/80">{{ 'checkout.protectionStep2' | t }}</span>
                </li>
                <li class="flex gap-3">
                  <span class="h-3.5 w-3.5 shrink-0 rounded-full border-2 border-on-forest-deep/40"></span>
                  <span class="text-small text-on-forest-deep/80">{{ 'checkout.protectionStep3' | t }}</span>
                </li>
              </ol>
            </div>
          </aside>
        </div>
      </div>

      <bt-dialog
        [open]="conflictOpen()"
        tone="alert"
        [title]="'checkout.conflictTitle' | t"
        [primaryLabel]="'checkout.conflictReduce' | t: { count: cycle.value()?.availableQuantity ?? 0 }"
        [secondaryLabel]="'checkout.conflictClose' | t"
        (primaryClick)="reduceToAvailable()"
        (secondaryClick)="conflictOpen.set(false)"
        (closed)="conflictOpen.set(false)"
      >
        {{ 'checkout.conflictBody' | t: { count: cycle.value()?.availableQuantity ?? 0 } }}
      </bt-dialog>
    } @else if (listing.isLoading()) {
      <p class="mx-auto max-w-lg px-5 py-16 text-body text-ink-muted">{{ 'checkout.loading' | t }}</p>
    }
  `
})
export class CheckoutComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly listingsApi = inject(ListingsApi);
  private readonly cyclesApi = inject(CyclesApi);
  private readonly reservationsApi = inject(ReservationsApi);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);

  private readonly listingId = toSignal(this.route.paramMap.pipe(map((p) => p.get('listingId') ?? '')), {
    initialValue: ''
  });
  private readonly initialQuantity = toSignal(
    this.route.queryParamMap.pipe(map((p) => Number(p.get('quantity')) || 1)),
    { initialValue: 1 }
  );

  protected readonly listing = httpResource<Listing>(() =>
    this.listingId() ? this.listingsApi.detailUrl(this.listingId()) : undefined
  );

  protected readonly cycle = httpResource<Cycle>(() => {
    const cycleId = this.listing.value()?.cycleId;
    return cycleId ? this.cyclesApi.detailUrl(cycleId) : undefined;
  });

  protected readonly quantity = signal(this.initialQuantity());
  protected readonly payNow = signal(true);
  protected readonly paymentMethod = signal<PaymentMethod>('mtn_momo');
  protected readonly submitting = signal(false);
  protected readonly conflictOpen = signal(false);

  protected readonly paymentMethodOptions = computed(() =>
    PAYMENT_METHOD_VALUES.map((value) => ({ value, label: this.locale.t(PAYMENT_METHOD_LABEL_KEYS[value]) }))
  );
  protected readonly payNowCopy = priorityChoiceCopy(true);
  protected readonly laterCopy = priorityChoiceCopy(false);
  protected readonly currentCopy = computed(() => priorityChoiceCopy(this.payNow()));

  protected readonly subtotal = computed(() => this.quantity() * (this.listing.value()?.unitPrice ?? 0));

  protected submit(listingId: string): void {
    if (this.submitting()) return;
    this.submitting.set(true);
    this.reservationsApi
      .create(listingId, {
        quantity: this.quantity(),
        payNow: this.payNow(),
        paymentMethod: this.payNow() ? this.paymentMethod() : undefined
      })
      .subscribe({
        next: ({ reservation }) => {
          this.toast.show({
            tone: 'success',
            title: reservation.priority ? 'checkout.confirmedTitle' : 'checkout.pendingTitle',
            detail: reservation.priority ? 'checkout.confirmedDetail' : 'checkout.pendingDetail'
          });
          this.router.navigateByUrl('/account/reservations');
        },
        error: (err: AppError) => {
          this.submitting.set(false);
          if (err.kind === 'conflict') {
            this.cycle.reload();
            this.conflictOpen.set(true);
          } else {
            this.toast.show({ tone: 'danger', title: 'checkout.failedTitle', detail: err.message });
          }
        }
      });
  }

  protected reduceToAvailable(): void {
    const available = this.cycle.value()?.availableQuantity ?? 1;
    this.quantity.set(Math.max(1, available));
    this.conflictOpen.set(false);
  }
}
