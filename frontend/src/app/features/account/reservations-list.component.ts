import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { Reservation } from '../../domain/models';
import { ReservationsApi } from '../../data-access/reservations/reservations.api';
import { ReservationRowComponent } from './reservation-row.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';

type ReservationTab = 'active' | 'completed' | 'cancelled';

const TAB_STATUSES: Readonly<Record<ReservationTab, Reservation['status'][]>> = {
  active: ['pending', 'confirmed', 'delivered'],
  completed: ['delivered'],
  cancelled: ['cancelled']
};

const TABS: readonly { id: ReservationTab; labelKey: string }[] = [
  { id: 'active', labelKey: 'reservations.tabActive' },
  { id: 'completed', labelKey: 'reservations.tabCompleted' },
  { id: 'cancelled', labelKey: 'reservations.tabCancelled' }
];

@Component({
  selector: 'bt-reservations-list',
  imports: [ReservationRowComponent, FcfaCurrencyPipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-content-max px-5 py-8 sm:px-18">
      <h1 class="text-heading-2">{{ 'reservations.title' | t }}</h1>

      @if (escrowHeldTotal() > 0) {
        <div class="mt-4 flex items-center justify-between rounded-lg bg-forest-deep p-5 text-on-forest-deep">
          <span class="text-body-strong">{{ 'reservations.escrowTotalLabel' | t }}</span>
          <span class="text-amount">{{ escrowHeldTotal() | fcfa }}</span>
        </div>
      }

      <div class="mt-6 flex gap-6 border-b border-line text-body-strong">
        @for (tab of tabs; track tab.id) {
          <button
            type="button"
            (click)="activeTab.set(tab.id)"
            [class]="'pb-3 ' + (activeTab() === tab.id ? 'border-b-2 border-forest text-forest' : 'text-ink-muted')"
          >
            {{ tab.labelKey | t }} · {{ tabCount(tab.id) }}
          </button>
        }
      </div>

      <div class="mt-6 flex flex-col gap-4">
        @if (reservations.isLoading()) {
          <p class="text-body text-ink-muted">{{ 'reservations.loading' | t }}</p>
        } @else if (filtered().length === 0) {
          <p class="text-body text-ink-muted">{{ 'reservations.empty' | t }}</p>
        } @else {
          @for (reservation of filtered(); track reservation.id) {
            <bt-reservation-row [reservation]="reservation" (refresh)="reservations.reload()" />
          }
        }
      </div>
    </div>
  `
})
export class ReservationsListComponent {
  private readonly reservationsApi = inject(ReservationsApi);

  protected readonly tabs = TABS;
  protected readonly activeTab = signal<ReservationTab>('active');

  protected readonly reservations = apiResource<Reservation[]>(() => this.reservationsApi.mineUrl());

  protected readonly filtered = computed(() => this.byTab(this.activeTab()));

  protected tabCount(tab: ReservationTab): number {
    return this.byTab(tab).length;
  }

  private byTab(tab: ReservationTab): Reservation[] {
    const statuses = TAB_STATUSES[tab];
    const all = this.reservations.value() ?? [];
    if (tab === 'completed') {
      return all.filter((r) => r.status === 'delivered' && r.payment?.status === 'released');
    }
    if (tab === 'active') {
      return all.filter((r) => statuses.includes(r.status) && !(r.status === 'delivered' && r.payment?.status === 'released'));
    }
    return all.filter((r) => statuses.includes(r.status));
  }

  protected readonly escrowHeldTotal = computed(() =>
    (this.reservations.value() ?? [])
      .filter((r) => r.payment?.status === 'held')
      .reduce((sum, r) => sum + (r.payment?.amount ?? 0), 0)
  );
}
