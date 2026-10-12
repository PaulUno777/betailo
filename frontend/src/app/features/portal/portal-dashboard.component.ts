import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { httpResource } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { Farm, SellerDashboard } from '../../domain/models';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { DashboardApi } from '../../data-access/dashboard/dashboard.api';
import { AppError } from '../../core/http/app-error';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { CapacityBarComponent } from '../../shared/ui/capacity-bar/capacity-bar.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';

/** `/portal` — per `Portal.dc.html`: KPI cards, active-cycle table and a
 * weekly reservations chart, all from `GET /farms/{farmId}/dashboard`.
 * A brand-new farmer account has no farm yet (there's no `Farm` board in
 * the design — the whole design assumes one exists), so this page is
 * also where the one-time "create your farm" step lives, built from the
 * plain `Farm` fields the API already accepts rather than left as a
 * dead end. */
@Component({
  selector: 'bt-portal-dashboard',
  imports: [
    FormsModule,
    RouterLink,
    StatusBadgeComponent,
    CapacityBarComponent,
    TextFieldComponent,
    ButtonComponent,
    DialogComponent,
    FcfaCurrencyPipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (farms.isLoading()) {
      <p class="p-8 text-body text-ink-muted">{{ 'portal.loading' | t }}</p>
    } @else if (!myFarm()) {
      <div class="mx-auto max-w-lg p-8">
        <div class="rounded-lg border border-line bg-surface-card p-6">
          <h1 class="text-heading-3">{{ 'portal.noFarmTitle' | t }}</h1>
          <p class="mt-2 text-body text-ink-muted">{{ 'portal.noFarmBody' | t }}</p>
          <div class="mt-5 flex flex-col gap-4">
            <bt-text-field [label]="'portal.farmNameLabel' | t" [ngModel]="farmName()" (ngModelChange)="farmName.set($event)" />
            <bt-text-field [label]="'portal.farmLocationLabel' | t" [ngModel]="farmLocation()" (ngModelChange)="farmLocation.set($event)" />
            <bt-text-field
              [label]="'portal.farmSpeciesLabel' | t"
              [ngModel]="farmSpecies()"
              (ngModelChange)="farmSpecies.set($event)"
            />
            <bt-button [disabled]="creatingFarm() || !farmName() || !farmLocation()" (click)="createFarm()">
              {{ (creatingFarm() ? 'portal.farmSaving' : 'portal.noFarmButton') | t }}
            </bt-button>
          </div>
        </div>
      </div>
    } @else {
      <div class="p-6 sm:p-8">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <h1 class="text-heading-2">{{ 'portal.greeting' | t: { name: myFarm()!.name } }}</h1>
          <a routerLink="/portal/cycles" class="inline-flex">
            <bt-button variant="solid">{{ 'portal.newCycleButton' | t }}</bt-button>
          </a>
        </div>

        @if (dashboard.value(); as d) {
          <div class="mt-6 grid gap-4 sm:grid-cols-4">
            <div class="rounded-lg border border-line bg-surface-card p-5">
              <span class="text-eyebrow text-ink-muted">{{ 'portal.kpiReservations' | t }}</span>
              <p class="mt-2 text-number-xl">{{ d.reservationsThisPeriod }}</p>
            </div>
            <div class="rounded-lg border border-line bg-sky-soft p-5">
              <span class="text-eyebrow text-sky">{{ 'portal.kpiEscrowHeld' | t }}</span>
              <p class="mt-2 text-number-xl">{{ d.escrowHeldTotal | fcfa }}</p>
            </div>
            <div class="rounded-lg border border-line bg-forest-soft p-5">
              <span class="text-eyebrow text-forest">{{ 'portal.kpiReleased' | t }}</span>
              <p class="mt-2 text-number-xl">{{ d.releasedThisPeriod | fcfa }}</p>
            </div>
            <div class="rounded-lg bg-forest-deep p-5 text-on-forest-deep">
              <span class="text-eyebrow text-on-forest-deep/70">{{ 'portal.kpiTrustScore' | t }}</span>
              <p class="mt-2 text-number-xl">{{ d.trustScore }}<span class="text-body text-on-forest-deep/70">/100</span></p>
            </div>
          </div>

          <div class="mt-6 grid gap-5 lg:grid-cols-[1fr_360px]">
            <section class="rounded-lg border border-line bg-surface-card p-6">
              <h2 class="text-heading-4">{{ 'portal.activeCyclesTitle' | t }}</h2>
              @if (d.activeCycles.length === 0) {
                <p class="mt-3 text-body text-ink-muted">{{ 'portal.noActiveCycles' | t }}</p>
              } @else {
                <div class="mt-4 flex flex-col gap-3">
                  @for (c of d.activeCycles; track c.cycle.id) {
                    <div class="grid grid-cols-[1fr_auto_1.4fr_auto] items-center gap-4 rounded-md border border-line p-3">
                      <div class="flex flex-col">
                        <span class="text-body-strong">{{ c.cycle.species }} · {{ c.cycle.type }}</span>
                        <span class="text-caption text-ink-muted">{{ c.cycle.id.slice(-6) }}</span>
                      </div>
                      <bt-status-badge kind="cycle" [status]="c.cycle.status" />
                      <bt-capacity-bar [capacity]="c.cycle.capacity" [availableQuantity]="c.cycle.availableQuantity" />
                      <span class="text-small text-ink-muted">{{ c.nextMilestone }}</span>
                    </div>
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

            <section class="rounded-lg border border-line bg-surface-card p-6">
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

    <bt-dialog
      [open]="farmError() !== null"
      tone="alert"
      [title]="'portal.farmSaveFailedTitle' | t"
      [primaryLabel]="'common.close' | t"
      [secondaryLabel]="'common.close' | t"
      (primaryClick)="farmError.set(null)"
      (secondaryClick)="farmError.set(null)"
      (closed)="farmError.set(null)"
    >
      {{ farmError() }}
    </bt-dialog>
  `
})
export class PortalDashboardComponent {
  private readonly farmsApi = inject(FarmsApi);
  private readonly dashboardApi = inject(DashboardApi);

  protected readonly farms = httpResource<Farm[]>(() => this.farmsApi.mineUrl());
  protected readonly myFarm = computed(() => this.farms.value()?.[0]);

  protected readonly dashboard = httpResource<SellerDashboard>(() => {
    const farmId = this.myFarm()?.id;
    return farmId ? this.dashboardApi.sellerDashboardUrl(farmId) : undefined;
  });

  protected readonly farmName = signal('');
  protected readonly farmLocation = signal('');
  protected readonly farmSpecies = signal('');
  protected readonly creatingFarm = signal(false);
  protected readonly farmError = signal<string | null>(null);

  protected barHeightPct(value: number, all: number[]): number {
    const max = Math.max(1, ...all);
    return Math.max(6, Math.round((value / max) * 100));
  }

  protected createFarm(): void {
    if (this.creatingFarm() || !this.farmName() || !this.farmLocation()) return;
    this.creatingFarm.set(true);
    const species = this.farmSpecies()
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);
    this.farmsApi.create({ name: this.farmName(), location: this.farmLocation(), species }).subscribe({
      next: () => {
        this.creatingFarm.set(false);
        this.farms.reload();
      },
      error: (err: AppError) => {
        this.creatingFarm.set(false);
        this.farmError.set(err.message);
      }
    });
  }
}
