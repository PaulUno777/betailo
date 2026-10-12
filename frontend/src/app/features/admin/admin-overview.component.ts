import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { StaffSpaceService } from '../../core/auth/staff-space.service';
import { apiResource } from '../../core/http/api-resource';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { AdminApi } from '../../data-access/admin/admin.api';
import { OverviewPeriod } from '../../domain/enums';
import { AdminEscrowItem, AdminOverview } from '../../domain/models';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { AdminPageHeaderComponent } from './ui/admin-page-header.component';
import { elapsed } from './ui/elapsed';

const PERIODS: readonly OverviewPeriod[] = ['7d', '30d', '12m'];
const CHART_HEIGHT = 180;

/** `/admin` — the platform at a glance, per `Admin.dc.html`: activity
 * over the chosen period, what the platform holds right now, and the
 * work waiting for staff. Every figure is `GET /admin/overview`; the
 * design's "system health" card shows the queues and switches the API
 * really knows about rather than made-up infrastructure metrics. The
 * escrow table only appears for roles that may read escrow. */
@Component({
  selector: 'bt-admin-overview',
  imports: [RouterLink, AdminPageHeaderComponent, ButtonComponent, NoticeComponent, StatusBadgeComponent, LocalDatePipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-[22px] p-4 sm:p-7 lg:px-9">
      <bt-admin-page-header [eyebrow]="'adminOverview.eyebrow' | t" [title]="'adminOverview.title' | t">
        <div role="group" [attr.aria-label]="'adminOverview.periodLabel' | t" class="flex rounded-[10px] border border-line bg-surface-card p-[3px]">
          @for (p of periods; track p) {
            <button
              type="button"
              [attr.aria-pressed]="period() === p"
              (click)="period.set(p)"
              class="h-9 rounded-lg px-3 text-caption font-semibold whitespace-nowrap transition"
              [class]="period() === p ? 'bg-ink text-surface' : 'text-ink-muted hover:text-ink'"
            >
              {{ 'adminOverview.period.' + p | t }}
            </button>
          }
        </div>
      </bt-admin-page-header>

      @if (overview.value(); as o) {
        @if (o.autoApproveFarms) {
          <bt-notice tone="warning" icon="alert" [title]="'adminOverview.autoApproveTitle' | t">{{ 'adminOverview.autoApproveBody' | t }}</bt-notice>
        }

        <div class="grid grid-cols-2 gap-3 sm:gap-3.5 xl:grid-cols-4">
          <div class="bo-card flex flex-col gap-2 p-[18px]">
            <span class="text-eyebrow text-ink-muted">{{ 'adminOverview.volumeReserved' | t }}</span>
            <span class="font-display text-[26px] leading-none font-extrabold sm:text-[30px]">{{ compact(o.volumeReserved) }} <span class="text-small font-semibold text-ink-muted">FCFA</span></span>
            @if (trend(o); as tr) {
              <span class="text-[12.5px] font-semibold" [class]="tr.up ? 'text-forest' : 'text-alert'">{{ 'adminOverview.trend' | t: { delta: tr.label } }}</span>
            } @else {
              <span class="text-[12.5px] text-ink-muted">{{ 'adminOverview.noTrend' | t }}</span>
            }
          </div>
          <div class="bo-card flex flex-col gap-2 p-[18px]">
            <span class="text-eyebrow text-ink-muted">{{ 'adminOverview.escrowHeld' | t }}</span>
            <span class="font-display text-[26px] leading-none font-extrabold sm:text-[30px]">{{ compact(o.escrowHeld) }} <span class="text-small font-semibold text-ink-muted">FCFA</span></span>
            <span class="text-[12.5px] text-ink-muted">{{ 'adminOverview.heldReservations' | t: { count: o.heldReservations } }}</span>
          </div>
          <div class="bo-card flex flex-col gap-2 p-[18px]">
            <span class="text-eyebrow text-ink-muted">{{ 'adminOverview.activeCycles' | t }}</span>
            <span class="font-display text-[26px] leading-none font-extrabold sm:text-[30px]">{{ o.activeCyclesPlanned + o.activeCyclesInProgress }}</span>
            <span class="text-[12.5px] text-ink-muted">{{ 'adminOverview.cyclesSplit' | t: { planned: o.activeCyclesPlanned, running: o.activeCyclesInProgress } }}</span>
          </div>
          <div class="flex flex-col gap-2 rounded-2xl border p-[18px]" [class]="o.openReports + o.openDisputes > 0 ? 'border-alert/30 bg-alert-soft text-alert' : 'border-line bg-surface-card'">
            <span class="text-eyebrow" [class.text-ink-muted]="o.openReports + o.openDisputes === 0">{{ 'adminOverview.openReports' | t }}</span>
            <span class="font-display text-[26px] leading-none font-extrabold sm:text-[30px]">{{ o.openReports }}</span>
            @if (staff.can('reports.read')) {
              <a routerLink="/admin/disputes" class="text-[12.5px] font-bold text-inherit underline-offset-2 hover:underline">{{ 'adminOverview.disputes' | t: { count: o.openDisputes } }} →</a>
            } @else {
              <span class="text-[12.5px]">{{ 'adminOverview.disputes' | t: { count: o.openDisputes } }}</span>
            }
          </div>
        </div>

        <div class="grid gap-[18px] xl:grid-cols-[minmax(0,1fr)_340px]">
          <section class="bo-card flex flex-col gap-4 p-4 sm:p-[22px]">
            <div class="flex flex-wrap items-baseline justify-between gap-3">
              <h2 class="font-display text-[19px] font-bold">{{ (o.period === '12m' ? 'adminOverview.chartTitleMonths' : 'adminOverview.chartTitleDays') | t }}</h2>
              <div class="flex gap-3.5 text-[12.5px] text-ink-muted">
                <span class="flex items-center gap-1.5"><span class="h-2.5 w-2.5 rounded-[3px] bg-forest"></span>{{ 'adminOverview.legendPaid' | t }}</span>
                <span class="flex items-center gap-1.5"><span class="h-2.5 w-2.5 rounded-[3px] bg-clay"></span>{{ 'adminOverview.legendWaiting' | t }}</span>
              </div>
            </div>
            @if (chart(); as c) {
              <div class="flex items-end gap-[3px] border-b border-line sm:gap-[5px]" [style.height.px]="chartHeight + 10" role="img" [attr.aria-label]="'adminOverview.chartSummary' | t: { paid: c.paid, waiting: c.waiting }">
                @for (b of c.bars; track b.start) {
                  <div class="flex h-full flex-1 flex-col justify-end gap-0.5" [attr.title]="b.title">
                    <div class="rounded-t-[3px] bg-clay" [style.height.px]="b.waiting"></div>
                    <div class="rounded-t-[3px] bg-forest" [style.height.px]="b.paid"></div>
                  </div>
                }
              </div>
              <div class="flex justify-between font-mono text-[11px] text-ink-muted">
                @for (tick of c.ticks; track tick) {
                  <span>{{ tick | localDate: (o.period === '12m' ? 'monthYear' : 'dayMonth') }}</span>
                }
              </div>
              @if (c.paid + c.waiting === 0) {
                <p class="text-small text-ink-muted">{{ 'adminOverview.chartEmpty' | t }}</p>
              }
            }
          </section>

          <section class="flex flex-col gap-3.5 rounded-2xl bg-ink p-4 text-on-forest-deep sm:p-[22px]">
            <div class="flex items-center justify-between gap-3">
              <h2 class="font-display text-[19px] font-bold">{{ 'adminOverview.queueTitle' | t }}</h2>
              @if (waiting(o) === 0) {
                <span class="inline-flex h-6 items-center gap-1.5 rounded-full bg-forest-soft/20 px-2.5 text-caption font-semibold text-forest-soft">
                  <span class="h-[7px] w-[7px] rounded-full bg-forest-soft"></span>{{ 'adminOverview.queueClear' | t }}
                </span>
              } @else {
                <span class="inline-flex h-6 items-center gap-1.5 rounded-full bg-clay/20 px-2.5 text-caption font-semibold text-clay-soft">
                  <span class="h-[7px] w-[7px] rounded-full bg-clay"></span>{{ 'adminOverview.queueWaiting' | t: { count: waiting(o) } }}
                </span>
              }
            </div>
            <dl class="flex flex-col font-mono text-[13px]">
              <div class="flex justify-between gap-3 border-b border-white/10 py-2.5">
                <dt class="text-on-forest-deep/60">{{ 'adminOverview.queueFarms' | t }}</dt>
                <dd [class.text-clay-soft]="o.pendingFarms > 0">{{ o.pendingFarms }}</dd>
              </div>
              <div class="flex justify-between gap-3 border-b border-white/10 py-2.5">
                <dt class="text-on-forest-deep/60">{{ 'adminOverview.queueOldest' | t }}</dt>
                <dd>{{ o.oldestPendingSince ? since(o.oldestPendingSince) : '—' }}</dd>
              </div>
              <div class="flex justify-between gap-3 border-b border-white/10 py-2.5">
                <dt class="text-on-forest-deep/60">{{ 'adminOverview.queueReports' | t }}</dt>
                <dd [class.text-clay-soft]="o.openReports > 0">{{ o.openReports }}</dd>
              </div>
              <div class="flex justify-between gap-3 border-b border-white/10 py-2.5">
                <dt class="text-on-forest-deep/60">{{ 'adminOverview.queueDisputes' | t }}</dt>
                <dd [class.text-clay-soft]="o.openDisputes > 0">{{ o.openDisputes }}</dd>
              </div>
              <div class="flex justify-between gap-3 border-b border-white/10 py-2.5">
                <dt class="text-on-forest-deep/60">{{ 'adminOverview.settingFarms' | t }}</dt>
                <dd>{{ o.maxFarmsPerAccount }}</dd>
              </div>
              <div class="flex justify-between gap-3 py-2.5">
                <dt class="text-on-forest-deep/60">{{ 'adminOverview.settingAutoApprove' | t }}</dt>
                <dd [class.text-clay-soft]="o.autoApproveFarms">{{ (o.autoApproveFarms ? 'adminOverview.on' : 'adminOverview.off') | t }}</dd>
              </div>
            </dl>
            @if (staff.can('farms.read') && o.pendingFarms > 0) {
              <a routerLink="/admin/farms" class="mt-auto inline-flex min-h-11 items-center justify-center rounded-[10px] bg-on-forest-deep px-4 text-small font-semibold text-ink transition hover:bg-white">
                {{ 'adminOverview.queueOpen' | t }}
              </a>
            }
          </section>
        </div>

        @if (staff.can('escrow.read')) {
          <section class="bo-card flex flex-col gap-2.5 px-2 pt-[22px] pb-2">
            <div class="flex items-center justify-between gap-3 px-3.5">
              <h2 class="font-display text-[19px] font-bold">{{ 'adminOverview.escrowTitle' | t }}</h2>
              <a routerLink="/admin/escrow" class="text-[13.5px] font-semibold text-forest hover:text-forest-deep">{{ 'adminOverview.seeAll' | t }}</a>
            </div>
            @if (latest().length === 0) {
              <p class="px-3.5 pb-4 text-small text-ink-muted">{{ (escrow.isLoading() ? 'adminOverview.loading' : 'adminEscrow.empty') | t }}</p>
            } @else {
              <div class="overflow-x-auto">
                <table class="bo-table min-w-[640px]">
                  <thead>
                    <tr>
                      <th class="bo-th" scope="col">{{ 'adminEscrow.colReservation' | t }}</th>
                      <th class="bo-th" scope="col">{{ 'adminEscrow.colBuyer' | t }}</th>
                      <th class="bo-th" scope="col">{{ 'adminEscrow.colFarm' | t }}</th>
                      <th class="bo-th" scope="col">{{ 'adminEscrow.colAmount' | t }}</th>
                      <th class="bo-th" scope="col">{{ 'adminEscrow.colStatus' | t }}</th>
                      <th class="bo-th" scope="col">{{ 'adminEscrow.colDate' | t }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    @for (r of latest(); track r.id) {
                      <tr>
                        <td class="bo-td font-mono !text-[13px]">{{ r.reference }}</td>
                        <td class="bo-td">{{ r.buyerName || '—' }}</td>
                        <td class="bo-td">{{ r.farmName || '—' }}</td>
                        <td class="bo-td font-mono !text-[13px] tabular-nums">{{ plain(r.totalAmount) }}</td>
                        <td class="bo-td">
                          @if (r.payment; as payment) {
                            <bt-status-badge kind="escrow" [status]="payment.status" />
                          } @else {
                            <bt-status-badge kind="reservation" [status]="r.status" />
                          }
                        </td>
                        <td class="bo-td !text-[13px] text-ink-muted">{{ r.createdAt | localDate: 'dayMonth' }}</td>
                      </tr>
                    }
                  </tbody>
                </table>
              </div>
            }
          </section>
        }
      } @else if (overview.error()) {
        <div class="flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="overview.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      } @else {
        <div class="grid grid-cols-2 gap-3 xl:grid-cols-4" aria-hidden="true">
          @for (i of [1, 2, 3, 4]; track i) {
            <div class="h-[118px] animate-pulse rounded-2xl bg-surface-sunken"></div>
          }
        </div>
        <div class="h-72 animate-pulse rounded-2xl bg-surface-sunken" aria-hidden="true"></div>
      }
    </div>
  `
})
export class AdminOverviewComponent {
  private readonly adminApi = inject(AdminApi);
  private readonly locale = inject(LocaleService);
  protected readonly staff = inject(StaffSpaceService);

  protected readonly periods = PERIODS;
  protected readonly chartHeight = CHART_HEIGHT;
  protected readonly period = signal<OverviewPeriod>('30d');

  protected readonly overview = apiResource<AdminOverview>(() => this.adminApi.overviewUrl(this.period()));
  protected readonly escrow = apiResource<AdminEscrowItem[]>(() => (this.staff.can('escrow.read') ? this.adminApi.escrowUrl() : undefined));

  /** The latest few, as the artboard's table does. */
  protected readonly latest = computed(() => (this.escrow.value() ?? []).slice(0, 6));

  /** The series scaled to the tallest bar, with its totals for the
   * chart's text alternative and three dates along the axis. */
  protected readonly chart = computed(() => {
    const series = this.overview.value()?.series ?? [];
    if (series.length === 0) return null;
    const max = Math.max(1, ...series.map((b) => b.paid + b.waiting));
    const px = (n: number) => (n === 0 ? 0 : Math.max(3, Math.round((n / max) * CHART_HEIGHT)));
    return {
      bars: series.map((b) => ({
        start: b.start,
        paid: px(b.paid),
        waiting: px(b.waiting),
        title: `${this.locale.t('adminOverview.legendPaid')}: ${b.paid} · ${this.locale.t('adminOverview.legendWaiting')}: ${b.waiting}`
      })),
      paid: series.reduce((sum, b) => sum + b.paid, 0),
      waiting: series.reduce((sum, b) => sum + b.waiting, 0),
      ticks: [series[0].start, series[Math.floor(series.length / 2)].start, series[series.length - 1].start]
    };
  });

  /** How the period compares with the one before; null when there is
   * nothing to compare with. */
  protected trend(o: AdminOverview): { up: boolean; label: string } | null {
    if (o.volumePrevious <= 0) return null;
    const pct = Math.round(((o.volumeReserved - o.volumePrevious) / o.volumePrevious) * 100);
    return { up: pct >= 0, label: `${pct >= 0 ? '+' : '−'}${Math.abs(pct)} %` };
  }

  protected waiting(o: AdminOverview): number {
    return o.pendingFarms + o.openReports + o.openDisputes;
  }

  protected since(iso: string): string {
    return elapsed(iso, this.locale);
  }

  /** "18,4 M" above a million, the full figure below. */
  protected compact(value: number): string {
    if (value >= 1_000_000) {
      const millions = (value / 1_000_000).toLocaleString(this.locale.locale() === 'fr' ? 'fr-FR' : 'en-GB', { maximumFractionDigits: 1 });
      return `${millions} M`;
    }
    return this.plain(value);
  }

  protected plain(value: number): string {
    return Math.round(value)
      .toLocaleString('fr-FR')
      .replace(/\s/g, ' ');
  }
}
