import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { Farm, ReservationList } from '../../domain/models';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { ReservationsApi } from '../../data-access/reservations/reservations.api';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { apiResource } from '../../core/http/api-resource';

/** `/portal/payments` — no dedicated payments endpoint exists; this
 * groups the same `GET /farms/{farmId}/reservations` feed `Reservations`
 * uses by `payment.status`, rather than inventing a new endpoint for
 * data the API already returns in a different shape. */
@Component({
  selector: 'bt-portal-payments',
  imports: [StatusBadgeComponent, FcfaCurrencyPipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="p-6 sm:p-8">
      <h1 class="text-heading-2">{{ 'portalPayments.title' | t }}</h1>

      @if (reservations.isLoading()) {
        <p class="mt-6 text-body text-ink-muted">{{ 'portalPayments.loading' | t }}</p>
      } @else {
        <div class="mt-6 grid gap-4 sm:grid-cols-3">
          <div class="rounded-lg border border-line bg-sky-soft p-5">
            <span class="text-eyebrow text-sky">{{ 'portalPayments.heldTotal' | t }}</span>
            <p class="mt-2 text-number-xl">{{ heldTotal() | fcfa }}</p>
          </div>
          <div class="rounded-lg border border-line bg-forest-soft p-5">
            <span class="text-eyebrow text-forest">{{ 'portalPayments.releasedTotal' | t }}</span>
            <p class="mt-2 text-number-xl">{{ releasedTotal() | fcfa }}</p>
          </div>
          <div class="rounded-lg border border-line bg-alert-soft p-5">
            <span class="text-eyebrow text-alert">{{ 'portalPayments.refundedTotal' | t }}</span>
            <p class="mt-2 text-number-xl">{{ refundedTotal() | fcfa }}</p>
          </div>
        </div>

        @if (withPayment().length === 0) {
          <p class="mt-6 text-body text-ink-muted">{{ 'portalPayments.empty' | t }}</p>
        } @else {
          <div class="mt-6 flex flex-col gap-2">
            @for (r of withPayment(); track r.id) {
              <div class="flex items-center justify-between gap-4 rounded-md border border-line p-3">
                <span class="text-mono-id text-ink-muted">{{ r.reference }}</span>
                <span class="text-body-strong">{{ r.totalAmount | fcfa }}</span>
                <bt-status-badge kind="escrow" [status]="r.payment!.status" />
              </div>
            }
          </div>
        }
      }
    </div>
  `
})
export class PortalPaymentsComponent {
  private readonly farmsApi = inject(FarmsApi);
  private readonly reservationsApi = inject(ReservationsApi);

  private readonly farms = apiResource<Farm[]>(() => this.farmsApi.mineUrl());
  private readonly myFarm = computed(() => this.farms.value()?.[0]);

  protected readonly reservations = apiResource<ReservationList>(() => {
    const farmId = this.myFarm()?.id;
    return farmId ? this.reservationsApi.byFarmUrl(farmId) : undefined;
  });

  protected readonly withPayment = computed(() => (this.reservations.value()?.items ?? []).filter((r) => r.payment));

  protected readonly heldTotal = computed(() => this.sumByStatus('held'));
  protected readonly releasedTotal = computed(() => this.sumByStatus('released'));
  protected readonly refundedTotal = computed(() => this.sumByStatus('refunded'));

  private sumByStatus(status: string): number {
    return this.withPayment()
      .filter((r) => r.payment?.status === status)
      .reduce((sum, r) => sum + (r.payment?.amount ?? 0), 0);
  }
}
