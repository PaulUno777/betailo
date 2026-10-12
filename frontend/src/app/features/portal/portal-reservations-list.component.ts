import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { Farm, ReservationAction, ReservationGroup, ReservationList, ReservationView } from '../../domain/models';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { ReservationsApi } from '../../data-access/reservations/reservations.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { ReservationCardComponent } from '../account/reservation-card.component';

const TABS: readonly { id: ReservationGroup; labelKey: string }[] = [
  { id: 'active', labelKey: 'reservations.tabActive' },
  { id: 'done', labelKey: 'reservations.tabCompleted' },
  { id: 'cancelled', labelKey: 'reservations.tabCancelled' }
];

/** The farmer's side of reservations: the same server-built cards as
 * the buyer's list, from the seller's perspective. The only action a
 * farmer has is the one the API offers, marking a paid reservation
 * delivered, which starts the buyer's confirmation window. */
@Component({
  selector: 'bt-portal-reservations-list',
  imports: [ReservationCardComponent, ButtonComponent, IconComponent, FcfaCurrencyPipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-3xl p-4 sm:p-8">
      <h1 class="text-heading-2">{{ 'portalReservations.title' | t }}</h1>

      <div class="mt-4 grid grid-cols-3 gap-1 rounded-full bg-surface-sunken p-1" role="tablist">
        @for (tab of tabs; track tab.id) {
          <button
            type="button"
            role="tab"
            [attr.aria-selected]="group() === tab.id"
            (click)="group.set(tab.id)"
            class="h-11 rounded-full text-small font-semibold transition"
            [class]="group() === tab.id ? 'bg-surface-card text-ink shadow-sm' : 'text-ink-muted'"
          >
            {{ tab.labelKey | t }} · {{ list.value()?.counts?.[tab.id] ?? 0 }}
          </button>
        }
      </div>

      @if ((list.value()?.escrowTotal ?? 0) > 0) {
        <div class="mt-4 flex items-center gap-3 rounded-xl bg-sky-soft px-5 py-4">
          <bt-icon name="lock" [size]="22" class="text-sky" />
          <div class="flex flex-col">
            <span class="text-heading-3">{{ list.value()!.escrowTotal | fcfa }}</span>
            <span class="text-small text-ink-muted">{{ 'portalReservations.escrowBanner' | t }}</span>
          </div>
        </div>
      }

      <div class="mt-5 flex flex-col gap-3">
        @if (list.value(); as l) {
          @for (v of l.items; track v.id) {
            <bt-reservation-card [view]="v" perspective="farmer" (action)="onAction($event, v)" />
          } @empty {
            <p class="rounded-xl border border-dashed border-line-strong px-6 py-12 text-center text-body text-ink-muted">
              {{ 'portalReservations.empty' | t }}
            </p>
          }
        } @else if (list.isLoading() || farms.isLoading()) {
          @for (i of [1, 2]; track i) {
            <div class="h-40 animate-pulse rounded-xl bg-surface-sunken"></div>
          }
        } @else if (list.error()) {
          <div class="flex flex-col items-center gap-3 py-10">
            <bt-button variant="outline" (click)="list.reload()">{{ 'common.retry' | t }}</bt-button>
          </div>
        }
      </div>
    </div>
  `
})
export class PortalReservationsListComponent {
  private readonly farmsApi = inject(FarmsApi);
  private readonly reservationsApi = inject(ReservationsApi);
  private readonly toast = inject(ToastService);

  protected readonly tabs = TABS;
  protected readonly group = signal<ReservationGroup>('active');

  protected readonly farms = apiResource<Farm[]>(() => this.farmsApi.mineUrl());
  private readonly myFarm = computed(() => this.farms.value()?.[0]);

  protected readonly list = apiResource<ReservationList>(() => {
    const farmId = this.myFarm()?.id;
    return farmId ? this.reservationsApi.byFarmUrl(farmId, this.group()) : undefined;
  });

  protected onAction(action: ReservationAction, v: ReservationView): void {
    if (action !== 'mark_delivered') return;
    this.reservationsApi.markDelivered(v.id).subscribe({
      next: () => {
        this.list.reload();
        this.toast.show({ tone: 'success', title: 'portalReservations.deliveredToast' });
      },
      error: (err: AppError) => {
        this.toast.show({ tone: 'danger', title: 'portalReservations.markDeliveredFailedTitle', detail: err.message });
      }
    });
  }
}
