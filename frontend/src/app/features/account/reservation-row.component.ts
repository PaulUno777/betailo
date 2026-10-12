import { ChangeDetectionStrategy, Component, computed, inject, input, output, signal } from '@angular/core';
import { httpResource } from '@angular/common/http';
import { Cycle, Farm, Listing, Reservation } from '../../domain/models';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { CyclesApi } from '../../data-access/cycles/cycles.api';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { ReservationsApi } from '../../data-access/reservations/reservations.api';
import { ReviewsApi } from '../../data-access/reviews/reviews.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { RatingComponent } from '../../shared/ui/rating/rating.component';
import { EscrowStepsComponent, EscrowStep } from '../../shared/ui/escrow-steps/escrow-steps.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { FormsModule } from '@angular/forms';

/** One row in "my reservations". `GET /reservations/mine` doesn't return a
 * joined listing summary (see docs/backend-improvements-needed.md), so
 * each row fetches its own listing — self-contained rather than the
 * parent doing N manual fetches. */
@Component({
  selector: 'bt-reservation-row',
  imports: [
    FormsModule,
    StatusBadgeComponent,
    ButtonComponent,
    DialogComponent,
    TextFieldComponent,
    RatingComponent,
    EscrowStepsComponent,
    FcfaCurrencyPipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-4 rounded-lg border border-line bg-surface-card p-5">
      <div class="flex items-start gap-4">
        @if (listing.value()?.photos?.[0]; as photo) {
          <img [src]="photo" [alt]="listing.value()?.title" class="h-16 w-16 shrink-0 rounded-md object-cover" />
        } @else {
          <div class="h-16 w-16 shrink-0 rounded-md bg-surface-sunken"></div>
        }
        <div class="flex-1">
          <span class="text-caption text-ink-muted">{{ reservationCode() }} · {{ 'reservations.reservedOn' | t: { date: reservedOnLabel() } }}</span>
          <p class="text-body-strong">{{ listing.value()?.title ?? ('reservations.loadingListing' | t) }}</p>
          <p class="text-small text-ink-muted">
            {{ reservation().quantity }} × {{ reservation().unitPrice | fcfa }}
            @if (farmName(); as name) { · {{ name }} }
          </p>
        </div>
        <div class="flex flex-col items-end gap-1.5">
          <bt-status-badge kind="reservation" [status]="reservation().status" />
          <bt-status-badge kind="payment" [status]="reservation().priority" />
        </div>
      </div>

      <div class="border-t border-line pt-4">
        <bt-escrow-steps [steps]="escrowSteps()" />
      </div>

      <div class="flex items-center justify-between border-t border-line pt-3">
        <span class="text-small text-ink-muted">{{ 'reservations.totalLabel' | t }}</span>
        <span class="text-amount">{{ reservation().totalAmount | fcfa }}</span>
      </div>

      @if (reservation().payment; as payment) {
        <div class="flex items-center justify-between">
          <span class="text-small text-ink-muted">{{ 'reservations.escrowLabel' | t }}</span>
          <bt-status-badge kind="escrow" [status]="payment.status" />
        </div>
      }

      <div class="flex flex-wrap gap-3">
        @if (reservation().status === 'pending' && !reservation().priority) {
          <bt-button variant="solid" size="sm" [disabled]="acting()" (click)="pay()">{{ 'reservations.payNow' | t }}</bt-button>
        }
        @if (reservation().status === 'delivered' && reservation().payment?.status !== 'released') {
          <bt-button variant="solid" size="sm" [disabled]="acting()" (click)="confirmDelivery()">{{ 'reservations.confirmDelivery' | t }}</bt-button>
          <bt-button variant="outline" size="sm" (click)="disputeOpen.set(true)">{{ 'reservations.openDispute' | t }}</bt-button>
        }
        @if (reservation().payment?.status === 'released') {
          <bt-button variant="ghost" size="sm" (click)="reviewOpen.set(true)">{{ 'reservations.leaveReview' | t }}</bt-button>
        }
      </div>
    </div>

    <bt-dialog
      [open]="disputeOpen()"
      tone="alert"
      [title]="'reservations.disputeTitle' | t"
      [primaryLabel]="'reservations.disputeSubmit' | t"
      primaryVariant="danger"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="submitDispute()"
      (secondaryClick)="disputeOpen.set(false)"
      (closed)="disputeOpen.set(false)"
    >
      <bt-text-field [label]="'reservations.disputeReasonLabel' | t" multiline [ngModel]="disputeReason()" (ngModelChange)="disputeReason.set($event)" />
    </bt-dialog>

    <bt-dialog
      [open]="reviewOpen()"
      tone="forest"
      [title]="'reservations.reviewTitle' | t"
      [primaryLabel]="'reservations.reviewSubmit' | t"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="submitReview()"
      (secondaryClick)="reviewOpen.set(false)"
      (closed)="reviewOpen.set(false)"
    >
      <div class="flex flex-col gap-4">
        <bt-rating [value]="reviewRating()" interactive [size]="28" (changed)="reviewRating.set($event)" />
        <bt-text-field [label]="'reservations.reviewCommentLabel' | t" multiline [ngModel]="reviewComment()" (ngModelChange)="reviewComment.set($event)" />
      </div>
    </bt-dialog>
  `
})
export class ReservationRowComponent {
  private readonly listingsApi = inject(ListingsApi);
  private readonly cyclesApi = inject(CyclesApi);
  private readonly farmsApi = inject(FarmsApi);
  private readonly reservationsApi = inject(ReservationsApi);
  private readonly reviewsApi = inject(ReviewsApi);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);

  readonly reservation = input.required<Reservation>();
  readonly refresh = output<void>();

  protected readonly listing = httpResource<Listing>(() => this.listingsApi.detailUrl(this.reservation().listingId));

  /** `GET /reservations/mine` has no joined listing/farm summary (see
   * docs/backend-improvements-needed.md), so this row chains its own
   * listing → cycle → farm fetches rather than showing a raw ID. */
  protected readonly cycle = httpResource<Cycle>(() => {
    const cycleId = this.listing.value()?.cycleId;
    return cycleId ? this.cyclesApi.detailUrl(cycleId) : undefined;
  });

  protected readonly farm = httpResource<Farm>(() => {
    const farmId = this.cycle.value()?.farmId;
    return farmId ? this.farmsApi.detailUrl(farmId) : undefined;
  });

  protected readonly farmName = computed(() => this.farm.value()?.name);

  protected reservationCode(): string {
    return `RSV-${this.reservation().id.slice(-4).toUpperCase()}`;
  }

  protected reservedOnLabel(): string {
    const locale = this.locale.locale() === 'fr' ? 'fr-FR' : 'en-US';
    return new Date(this.reservation().createdAt).toLocaleDateString(locale, { day: 'numeric', month: 'short' });
  }

  /** Only steps this session's data can actually back — no per-reservation
   * "hatched" event exists in the API, so the design's 5-step line is
   * trimmed to the 4 states `status`/`payment.status` truthfully carry. */
  protected escrowSteps(): EscrowStep[] {
    const r = this.reservation();
    const paid = r.payment?.status === 'held' || r.payment?.status === 'released';
    const delivered = r.status === 'delivered';
    const released = r.payment?.status === 'released';
    const awaitingConfirmation = delivered && r.payment?.status === 'held';
    return [
      { label: this.locale.t('reservations.stepPaid'), state: paid ? 'done' : 'pending' },
      { label: this.locale.t('reservations.stepDelivered'), state: delivered ? 'done' : paid ? 'current' : 'pending' },
      { label: this.locale.t('reservations.stepConfirm'), state: released ? 'done' : awaitingConfirmation ? 'current' : 'pending' },
      { label: this.locale.t('reservations.stepReleased'), state: released ? 'done' : 'pending' }
    ];
  }

  protected readonly acting = signal(false);
  protected readonly disputeOpen = signal(false);
  protected readonly disputeReason = signal('');
  protected readonly reviewOpen = signal(false);
  protected readonly reviewRating = signal(5);
  protected readonly reviewComment = signal('');

  protected pay(): void {
    this.acting.set(true);
    this.reservationsApi.pay(this.reservation().id).subscribe({
      next: () => {
        this.toast.show({ tone: 'success', title: 'reservations.paymentSuccessTitle', detail: 'reservations.paymentSuccessDetail' });
        this.refresh.emit();
      },
      error: (err: AppError) => {
        this.acting.set(false);
        this.toast.show({ tone: 'danger', title: 'reservations.paymentFailedTitle', detail: err.message });
      }
    });
  }

  protected confirmDelivery(): void {
    this.acting.set(true);
    this.reservationsApi.confirmDelivery(this.reservation().id).subscribe({
      next: () => {
        this.toast.show({ tone: 'success', title: 'reservations.deliveryConfirmedTitle', detail: 'reservations.deliveryConfirmedDetail' });
        this.refresh.emit();
      },
      error: (err: AppError) => {
        this.acting.set(false);
        this.toast.show({ tone: 'danger', title: 'reservations.deliveryFailedTitle', detail: err.message });
      }
    });
  }

  protected submitDispute(): void {
    this.reservationsApi.dispute(this.reservation().id, this.disputeReason()).subscribe({
      next: () => {
        this.disputeOpen.set(false);
        this.disputeReason.set('');
        this.toast.show({ tone: 'info', title: 'reservations.disputeOpenedTitle', detail: 'reservations.disputeOpenedDetail' });
        this.refresh.emit();
      },
      error: (err: AppError) => {
        this.toast.show({ tone: 'danger', title: 'reservations.disputeFailedTitle', detail: err.message });
      }
    });
  }

  protected submitReview(): void {
    this.reviewsApi.create(this.reservation().id, { rating: this.reviewRating(), comment: this.reviewComment() || undefined }).subscribe({
      next: () => {
        this.reviewOpen.set(false);
        this.toast.show({ tone: 'success', title: 'reservations.reviewPublishedTitle', detail: 'reservations.reviewPublishedDetail' });
      },
      error: (err: AppError) => {
        this.toast.show({ tone: 'danger', title: 'reservations.reviewFailedTitle', detail: err.message });
      }
    });
  }
}
