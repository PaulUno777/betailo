import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { DashboardTodo, Farm, SellerDashboard } from '../../domain/models';
import { speciesLabelKey } from '../../domain/reference/species';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { DashboardApi } from '../../data-access/dashboard/dashboard.api';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { CapacityBarComponent } from '../../shared/ui/capacity-bar/capacity-bar.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { IconName } from '../../shared/ui/icon/icon-paths';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { SellerOnboardingComponent } from '../sell/seller-onboarding.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { apiResource } from '../../core/http/api-resource';

const TODO_ICONS: Readonly<Record<DashboardTodo['kind'], IconName>> = {
  mark_delivered: 'check',
  health_log: 'leaf',
  awaiting_payment: 'wallet'
};

/** `/portal` — per `Portal.dc.html`: KPI cards, active-cycle table and a
 * weekly reservations chart, all from `GET /farms/{farmId}/dashboard`.
 * "À faire aujourd'hui" comes first: the server decides what needs the
 * farmer now, and each item opens the page where it gets done. A farmer
 * account without a farm gets the same onboarding step as the sell
 * wizard instead of a dead end. */
@Component({
  selector: 'bt-portal-dashboard',
  imports: [
    RouterLink,
    StatusBadgeComponent,
    CapacityBarComponent,
    IconComponent,
    ButtonComponent,
    SellerOnboardingComponent,
    FcfaCurrencyPipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (farms.isLoading()) {
      <p class="p-8 text-body text-ink-muted">{{ 'portal.loading' | t }}</p>
    } @else if (!myFarm()) {
      <div class="mx-auto max-w-lg p-4 sm:p-8">
        <bt-seller-onboarding (created)="farms.reload()" />
      </div>
    } @else {
      <div class="p-4 sm:p-8">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <h1 class="text-heading-2">{{ 'portal.greeting' | t: { name: myFarm()!.name } }}</h1>
          <bt-button variant="solid" href="/sell">
            <bt-icon name="plus" [size]="18" />
            {{ 'portal.sellButton' | t }}
          </bt-button>
        </div>

        @if (dashboard.value(); as d) {
          @if (d.todos.length) {
            <section class="mt-6 rounded-xl border border-clay/30 bg-clay-soft p-4 sm:p-5">
              <h2 class="text-heading-4">{{ 'portal.todoTitle' | t }}</h2>
              <ul class="mt-3 flex flex-col gap-2">
                @for (todo of d.todos; track $index) {
                  <li>
                    <a
                      [routerLink]="todoRoute(todo)"
                      class="flex min-h-14 items-center gap-3 rounded-lg bg-surface-card px-4 py-2 text-body transition active:scale-[0.99]"
                    >
                      <bt-icon [name]="todoIcon(todo)" [size]="22" class="shrink-0 text-clay" />
                      <span class="flex-1">{{ 'portal.todo.' + todo.kind | t: { count: todo.count ?? 1, ref: todo.cycleRef ?? '' } }}</span>
                      <bt-icon name="chevron-right" [size]="18" class="text-ink-muted" />
                    </a>
                  </li>
                }
              </ul>
            </section>
          }

          <div class="mt-6 grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-4">
            <div class="order-1 rounded-lg border border-line bg-surface-card p-4 sm:p-5 lg:order-none">
              <span class="text-eyebrow text-ink-muted">{{ 'portal.kpiReservations' | t }}</span>
              <p class="mt-2 text-heading-2 sm:text-number-xl">{{ d.reservationsThisPeriod }}</p>
            </div>
            <div class="order-3 col-span-2 rounded-lg border border-line bg-sky-soft p-4 sm:p-5 lg:order-none lg:col-span-1">
              <span class="text-eyebrow text-sky">{{ 'portal.kpiEscrowHeld' | t }}</span>
              <p class="mt-2 text-heading-2 sm:text-number-xl">{{ d.escrowHeldTotal | fcfa }}</p>
            </div>
            <div class="order-4 col-span-2 rounded-lg border border-line bg-forest-soft p-4 sm:p-5 lg:order-none lg:col-span-1">
              <span class="text-eyebrow text-forest">{{ 'portal.kpiReleased' | t }}</span>
              <p class="mt-2 text-heading-2 sm:text-number-xl">{{ d.releasedThisPeriod | fcfa }}</p>
            </div>
            <div class="order-2 rounded-lg bg-forest-deep p-4 text-on-forest-deep sm:p-5 lg:order-none">
              <span class="text-eyebrow text-on-forest-deep/70">{{ 'portal.kpiTrustScore' | t }}</span>
              <p class="mt-2 text-heading-2 sm:text-number-xl">{{ d.trustScore }}<span class="text-body text-on-forest-deep/70">/100</span></p>
            </div>
          </div>

          <div class="mt-6 grid gap-5 lg:grid-cols-[1fr_360px]">
            <section class="rounded-lg border border-line bg-surface-card p-4 sm:p-6">
              <h2 class="text-heading-4">{{ 'portal.activeCyclesTitle' | t }}</h2>
              @if (d.activeCycles.length === 0) {
                <p class="mt-3 text-body text-ink-muted">{{ 'portal.noActiveCycles' | t }}</p>
              } @else {
                <div class="mt-4 flex flex-col gap-3">
                  @for (c of d.activeCycles; track c.cycle.id) {
                    <a
                      [routerLink]="['/portal/cycles', c.cycle.id]"
                      class="grid grid-cols-[1fr_auto] items-center gap-x-4 gap-y-2 rounded-md border border-line p-3 transition hover:bg-surface-sunken sm:grid-cols-[1fr_auto_1.4fr_auto]"
                    >
                      <div class="flex flex-col">
                        <span class="text-body-strong">{{ speciesKey(c.cycle.species) | t }} · {{ c.cycle.type }}</span>
                        <span class="text-caption text-ink-muted">{{ c.cycle.reference || c.cycle.id.slice(-6) }}</span>
                      </div>
                      <bt-status-badge kind="cycle" [status]="c.cycle.status" />
                      <bt-capacity-bar class="col-span-2 sm:col-span-1" [capacity]="c.cycle.capacity" [availableQuantity]="c.cycle.availableQuantity" />
                      <span class="col-span-2 text-small text-ink-muted sm:col-span-1">{{ c.nextMilestone }}</span>
                    </a>
                  }
                </div>
              }

              <div class="mt-6 border-t border-line pt-5">
                <p class="text-body-strong">{{ 'portal.weeklyChartTitle' | t }}</p>
                <div class="mt-4 grid grid-cols-8 items-end gap-2" style="height: 100px">
                  @for (v of d.weeklyReservationCounts; track $index) {
                    <div
                      class="rounded-t-sm bg-forest"
                      [style.height.%]="barHeightPct(v, d.weeklyReservationCounts)"
                      [attr.title]="v"
                    ></div>
                  }
                </div>
              </div>
            </section>

            <section class="rounded-lg border border-line bg-surface-card p-4 sm:p-6">
              <h2 class="text-heading-4">{{ 'portal.latestReservationsTitle' | t }}</h2>
              @if (d.latestReservations.length === 0) {
                <p class="mt-3 text-body text-ink-muted">{{ 'portal.noReservations' | t }}</p>
              } @else {
                <div class="mt-4 flex flex-col gap-3">
                  @for (r of d.latestReservations; track r.id) {
                    <div class="flex items-center justify-between gap-3">
                      <span class="text-small text-ink-muted">RSV-{{ r.id.slice(-4).toUpperCase() }}</span>
                      <span class="text-small">{{ r.quantity }} × {{ r.unitPrice | fcfa }}</span>
                      <bt-status-badge kind="reservation" [status]="r.status" />
                    </div>
                  }
                </div>
              }
            </section>
          </div>
        }
      </div>
    }

  `
})
export class PortalDashboardComponent {
  private readonly farmsApi = inject(FarmsApi);
  private readonly dashboardApi = inject(DashboardApi);

  protected readonly farms = apiResource<Farm[]>(() => this.farmsApi.mineUrl());
  protected readonly myFarm = computed(() => this.farms.value()?.[0]);

  protected readonly dashboard = apiResource<SellerDashboard>(() => {
    const farmId = this.myFarm()?.id;
    return farmId ? this.dashboardApi.sellerDashboardUrl(farmId) : undefined;
  });

  protected barHeightPct(value: number, all: number[]): number {
    const max = Math.max(1, ...all);
    return Math.max(6, Math.round((value / max) * 100));
  }

  protected speciesKey(value: string): string {
    return speciesLabelKey(value);
  }

  /** Each "À faire" item opens the page where it gets done. */
  protected todoRoute(todo: DashboardTodo): string | readonly string[] {
    return todo.kind === 'health_log' && todo.cycleId ? ['/portal/cycles', todo.cycleId] : '/portal/reservations';
  }

  protected todoIcon(todo: DashboardTodo): IconName {
    return TODO_ICONS[todo.kind];
  }
}
