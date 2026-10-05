import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { Farm, Reservation } from '../../domain/models';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { ReservationsApi } from '../../data-access/reservations/reservations.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { apiResource } from '../../core/http/api-resource';

/** `/portal/reservations` — the farmer-side view of `GET
 * /farms/{farmId}/reservations`, with the one farmer-side action the
 * API exposes: `mark-delivered`. */
@Component({
  selector: 'bt-portal-reservations-list',
  imports: [StatusBadgeComponent, ButtonComponent, FcfaCurrencyPipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="p-6 sm:p-8">
      <h1 class="text-heading-2">{{ 'portalReservations.title' | t }}</h1>

      @if (reservations.isLoading()) {
        <p class="mt-6 text-body text-ink-muted">{{ 'portalReservations.loading' | t }}</p>
      } @else if ((reservations.value() ?? []).length === 0) {
        <p class="mt-6 text-body text-ink-muted">{{ 'portalReservations.empty' | t }}</p>
      } @else {
        <div class="mt-6 flex flex-col gap-3">
          @for (r of reservations.value()!; track r.id) {
            <div class="flex flex-wrap items-center gap-4 rounded-lg border border-line bg-surface-card p-4">
              <span class="text-caption text-ink-muted">RSV-{{ r.id.slice(-4).toUpperCase() }}</span>
              <span class="min-w-[100px] flex-1 text-body-strong">{{ r.quantity }} × {{ r.unitPrice | fcfa }}</span>
              <span class="text-amount">{{ r.totalAmount | fcfa }}</span>
              <bt-status-badge kind="reservation" [status]="r.status" />
              @if (r.payment; as payment) {
                <bt-status-badge kind="escrow" [status]="payment.status" />
              }
              @if (r.status === 'confirmed') {
                <bt-button variant="outline" size="sm" [disabled]="acting() === r.id" (click)="markDelivered(r.id)">
                  {{ 'portalReservations.markDeliveredButton' | t }}
                </bt-button>
              }
            </div>
          }
        </div>
      }
    </div>
  `
})
export class PortalReservationsListComponent {
  private readonly farmsApi = inject(FarmsApi);
  private readonly reservationsApi = inject(ReservationsApi);
  private readonly toast = inject(ToastService);

  private readonly farms = apiResource<Farm[]>(() => this.farmsApi.mineUrl());
  private readonly myFarm = computed(() => this.farms.value()?.[0]);

  protected readonly reservations = apiResource<Reservation[]>(() => {
    const farmId = this.myFarm()?.id;
    return farmId ? this.reservationsApi.byFarmUrl(farmId) : undefined;
  });

  protected readonly acting = signal<string | null>(null);

  protected markDelivered(id: string): void {
    this.acting.set(id);
    this.reservationsApi.markDelivered(id).subscribe({
      next: () => {
        this.acting.set(null);
        this.reservations.reload();
      },
      error: (err: AppError) => {
        this.acting.set(null);
        this.toast.show({ tone: 'danger', title: 'portalReservations.markDeliveredFailedTitle', detail: err.message });
      }
    });
  }
}
