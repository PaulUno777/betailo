import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { map } from 'rxjs';
import { ReportStatus } from '../../domain/enums';
import { Report } from '../../domain/models';
import { AdminApi } from '../../data-access/admin/admin.api';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { apiResource } from '../../core/http/api-resource';
import { AdminPageHeaderComponent } from './ui/admin-page-header.component';

const STATUS_TONE: Readonly<Record<ReportStatus, 'amber' | 'sky' | 'forest' | 'stone'>> = {
  open: 'amber',
  under_review: 'sky',
  handled: 'forest',
  rejected: 'stone'
};

type StatusFilter = 'open' | 'all' | 'handled';
const STATUS_FILTERS: readonly StatusFilter[] = ['open', 'all', 'handled'];

/** `/admin/reports` and `/admin/disputes` — per `Moderation.dc.html`'s
 * queue. The same list read two ways: reports are about a listing, a
 * profile or a forum post; disputes are about a reservation, between a
 * buyer and a seller (the route says which, see `data.disputes`). Each
 * row opens the detail, with its risk signals and the decision. */
@Component({
  selector: 'bt-admin-reports-list',
  imports: [RouterLink, AdminPageHeaderComponent, BadgeComponent, ButtonComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-[22px] p-4 sm:p-7 lg:px-9">
      <bt-admin-page-header
        [eyebrow]="'adminNav.group.trust' | t"
        [title]="(disputes() ? 'adminReports.disputesTitle' : 'adminReports.title') | t"
        [subtitle]="(disputes() ? 'adminReports.disputesSubtitle' : 'adminReports.subtitle') | t"
      />

      <div class="flex flex-wrap gap-1.5" role="group" [attr.aria-label]="'adminReports.statusFilterLabel' | t">
        @for (f of statusFilters; track f) {
          <button
            type="button"
            [attr.aria-pressed]="status() === f"
            (click)="status.set(f)"
            class="inline-flex h-9 items-center rounded-full border px-3 text-caption font-semibold whitespace-nowrap transition"
            [class]="status() === f ? 'border-ink bg-ink text-surface' : 'border-line bg-surface-card text-ink hover:border-line-strong'"
          >
            {{ 'adminReports.filter.' + f | t }} · {{ count(f) }}
          </button>
        }
      </div>

      @if (reports.isLoading() && !reports.hasValue()) {
        <div class="h-56 animate-pulse rounded-2xl bg-surface-sunken" aria-hidden="true"></div>
      } @else if (reports.error()) {
        <div class="flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="reports.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      } @else if (shown().length === 0) {
        <p class="bo-card p-6 text-body text-ink-muted">{{ (disputes() ? 'adminReports.disputesEmpty' : 'adminReports.empty') | t }}</p>
      } @else {
        <ul class="flex flex-col gap-2">
          @for (r of shown(); track r.id) {
            <li>
              <a [routerLink]="['/admin/reports', r.id]" class="bo-card flex flex-wrap items-center gap-x-4 gap-y-2 p-4 text-ink transition hover:border-line-strong">
                <bt-badge tone="stone">{{ 'reportTarget.' + r.targetType | t }}</bt-badge>
                <span class="min-w-[200px] flex-1 wrap-anywhere text-body-strong">{{ r.reason }}</span>
                <bt-badge [tone]="statusTone(r.status)">{{ 'reportStatus.' + r.status | t }}</bt-badge>
                <bt-icon name="chevron-right" [size]="18" class="text-ink-muted" />
              </a>
            </li>
          }
        </ul>
      }
    </div>
  `
})
export class AdminReportsListComponent {
  private readonly adminApi = inject(AdminApi);
  private readonly route = inject(ActivatedRoute);

  /** `/admin/disputes` sets it: reservations only. */
  protected readonly disputes = toSignal(this.route.data.pipe(map((d) => d['disputes'] === true)), { initialValue: false });
  protected readonly statusFilters = STATUS_FILTERS;
  protected readonly status = signal<StatusFilter>('open');

  protected readonly reports = apiResource<Report[]>(() => this.adminApi.reportsUrl());

  /** This page's half of the list: disputes or everything else. */
  private readonly mine = computed(() => (this.reports.value() ?? []).filter((r) => (r.targetType === 'reservation') === this.disputes()));

  private matching(filter: StatusFilter): Report[] {
    const open = (r: Report) => r.status === 'open' || r.status === 'under_review';
    return filter === 'all' ? this.mine() : this.mine().filter((r) => open(r) === (filter === 'open'));
  }

  protected count(filter: StatusFilter): number {
    return this.matching(filter).length;
  }

  protected readonly shown = computed(() => this.matching(this.status()));

  protected statusTone(status: ReportStatus) {
    return STATUS_TONE[status];
  }
}
