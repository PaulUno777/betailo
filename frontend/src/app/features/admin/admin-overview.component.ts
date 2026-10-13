import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { AdminOverview } from '../../domain/models';
import { AdminApi } from '../../data-access/admin/admin.api';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { apiResource } from '../../core/http/api-resource';

/** `/admin` — per `Admin.dc.html`: platform-wide KPIs and a 30-day
 * reservations chart, from `GET /admin/overview`. */
@Component({
  selector: 'bt-admin-overview',
  imports: [ButtonComponent, FcfaCurrencyPipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="p-6 sm:p-8">
      <h1 class="text-heading-2">{{ 'adminOverview.title' | t }}</h1>

      @if (overview.isLoading()) {
        <p class="mt-6 text-body text-ink-muted">{{ 'adminOverview.loading' | t }}</p>
      } @else if (overview.value(); as o) {
        <div class="mt-6 grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-5">
          <div class="col-span-2 lg:col-span-1 rounded-lg border border-line bg-surface-card p-4 sm:p-5">
            <span class="text-eyebrow text-ink-muted">{{ 'adminOverview.volumeReserved' | t }}</span>
            <p class="mt-2 text-heading-2 sm:text-number-xl">{{ o.volumeReserved | fcfa }}</p>
          </div>
          <div class="col-span-2 lg:col-span-1 rounded-lg border border-line bg-sky-soft p-4 sm:p-5">
            <span class="text-eyebrow text-sky">{{ 'adminOverview.escrowHeld' | t }}</span>
            <p class="mt-2 text-heading-2 sm:text-number-xl">{{ o.escrowHeld | fcfa }}</p>
          </div>
          <div class="rounded-lg border border-line bg-surface-card p-4 sm:p-5">
            <span class="text-eyebrow text-ink-muted">{{ 'adminOverview.cyclesPlanned' | t }}</span>
            <p class="mt-2 text-heading-2 sm:text-number-xl">{{ o.activeCyclesPlanned }}</p>
          </div>
          <div class="rounded-lg border border-line bg-forest-soft p-4 sm:p-5">
            <span class="text-eyebrow text-forest">{{ 'adminOverview.cyclesInProgress' | t }}</span>
            <p class="mt-2 text-heading-2 sm:text-number-xl">{{ o.activeCyclesInProgress }}</p>
          </div>
          <div class="col-span-2 lg:col-span-1 rounded-lg border border-line bg-alert-soft p-4 sm:p-5">
            <span class="text-eyebrow text-alert">{{ 'adminOverview.openReports' | t }}</span>
            <p class="mt-2 text-heading-2 sm:text-number-xl">{{ o.openReports }}</p>
          </div>
        </div>

        <div class="mt-6 rounded-lg border border-line bg-surface-card p-4 sm:p-6">
          <p class="text-body-strong">{{ 'adminOverview.reservationsChartTitle' | t }}</p>
          <div
            class="mt-4 grid items-end gap-0.5"
            style="height: 100px; grid-template-columns: repeat(30, minmax(0, 1fr))"
            role="img"
            [attr.aria-label]="('adminOverview.reservationsChartTitle' | t) + ' : ' + total(o.reservationsPerDay)"
          >
            @for (v of o.reservationsPerDay; track $index) {
              <div class="rounded-t-sm bg-ink" [style.height.%]="barHeightPct(v, o.reservationsPerDay)" [attr.title]="v"></div>
            }
          </div>
        </div>
      } @else if (overview.error()) {
        <div class="mt-6 flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="overview.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      }
    </div>
  `
})
export class AdminOverviewComponent {
  private readonly adminApi = inject(AdminApi);

  protected readonly overview = apiResource<AdminOverview>(() => this.adminApi.overviewUrl());

  protected total(all: number[]): number {
    return all.reduce((sum, v) => sum + v, 0);
  }

  protected barHeightPct(value: number, all: number[]): number {
    const max = Math.max(1, ...all);
    return Math.max(4, Math.round((value / max) * 100));
  }
}
