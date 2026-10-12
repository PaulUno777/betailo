import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { httpResource } from '@angular/common/http';
import { RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { ReportStatus } from '../../domain/enums';
import { Report } from '../../domain/models';
import { AdminApi } from '../../data-access/admin/admin.api';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { SelectFieldComponent } from '../../shared/ui/select-field/select-field.component';

const STATUS_TONE: Readonly<Record<ReportStatus, 'amber' | 'sky' | 'forest' | 'stone'>> = {
  open: 'amber',
  under_review: 'sky',
  handled: 'forest',
  rejected: 'stone'
};

/** `/admin/reports` — per `Moderation.dc.html`'s queue: every report,
 * filterable by status, linking into its detail (risk score + signals +
 * resolve actions). */
@Component({
  selector: 'bt-admin-reports-list',
  imports: [FormsModule, RouterLink, BadgeComponent, SelectFieldComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="p-6 sm:p-8">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <h1 class="text-heading-2">{{ 'adminReports.title' | t }}</h1>
        <div class="w-56">
          <bt-select-field
            [label]="'adminReports.statusFilterLabel' | t"
            [options]="statusOptions"
            [placeholder]="'adminReports.allStatuses' | t"
            [ngModel]="status()"
            (ngModelChange)="status.set($event)"
          />
        </div>
      </div>

      @if (reports.isLoading()) {
        <p class="mt-6 text-body text-ink-muted">{{ 'adminReports.loading' | t }}</p>
      } @else if ((reports.value() ?? []).length === 0) {
        <p class="mt-6 text-body text-ink-muted">{{ 'adminReports.empty' | t }}</p>
      } @else {
        <div class="mt-6 flex flex-col gap-2">
          @for (r of reports.value()!; track r.id) {
            <a
              [routerLink]="['/admin/reports', r.id]"
              class="flex flex-wrap items-center gap-4 rounded-lg border border-line bg-surface-card p-4 text-ink no-underline hover:border-line-strong"
            >
              <bt-badge tone="stone">{{ r.targetType }}</bt-badge>
              <span class="min-w-[200px] flex-1 text-body-strong">{{ r.reason }}</span>
              <bt-badge [tone]="statusTone(r.status)">{{ r.status }}</bt-badge>
            </a>
          }
        </div>
      }
    </div>
  `
})
export class AdminReportsListComponent {
  private readonly adminApi = inject(AdminApi);

  protected readonly statusOptions = [
    { value: 'open', label: 'open' },
    { value: 'under_review', label: 'under_review' },
    { value: 'handled', label: 'handled' },
    { value: 'rejected', label: 'rejected' }
  ];

  protected readonly status = signal('');

  protected readonly reports = httpResource<Report[]>(() => this.adminApi.reportsUrl(this.status() || undefined));

  protected statusTone(status: ReportStatus) {
    return STATUS_TONE[status];
  }
}
