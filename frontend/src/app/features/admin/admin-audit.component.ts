import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { apiResource } from '../../core/http/api-resource';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { AdminApi } from '../../data-access/admin/admin.api';
import { AdminAction } from '../../domain/models';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { StaffRoleBadgeComponent } from '../../shared/ui/staff-role-badge/staff-role-badge.component';
import { AdminPageHeaderComponent } from './ui/admin-page-header.component';

/** `/admin/audit` — "Journal": what staff did, latest first, as the API
 * recorded it (`GET /admin/audit`): who, in which role, what, on which
 * farm, account or reservation, and the reason given. Every decision
 * and every opening of an identity document is a line here. Read-only:
 * nothing edits or deletes a line. */
@Component({
  selector: 'bt-admin-audit',
  imports: [AdminPageHeaderComponent, ButtonComponent, StaffRoleBadgeComponent, LocalDatePipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-[22px] p-4 sm:p-7 lg:px-9">
      <bt-admin-page-header [eyebrow]="'adminNav.group.system' | t" [title]="'adminAudit.title' | t" [subtitle]="'adminAudit.subtitle' | t" />

      @if (actions.isLoading() && !actions.hasValue()) {
        <div class="h-72 animate-pulse rounded-2xl bg-surface-sunken" aria-hidden="true"></div>
      } @else if (actions.error()) {
        <div class="flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="actions.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      } @else if ((actions.value() ?? []).length === 0) {
        <p class="bo-card p-6 text-body text-ink-muted">{{ 'adminAudit.empty' | t }}</p>
      } @else {
        <section class="bo-card px-2 pt-[18px] pb-2">
          <div class="overflow-x-auto">
            <table class="bo-table min-w-[820px]">
              <thead>
                <tr>
                  <th class="bo-th" scope="col">{{ 'adminAudit.colWhen' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminAudit.colWho' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminAudit.colAction' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminAudit.colTarget' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminAudit.colDetail' | t }}</th>
                </tr>
              </thead>
              <tbody>
                @for (a of actions.value()!; track a.id) {
                  <tr>
                    <td class="bo-td font-mono !text-[13px] whitespace-nowrap text-ink-muted">{{ a.createdAt | localDate: 'dayMonthTime' }}</td>
                    <td class="bo-td">
                      <div class="flex flex-wrap items-center gap-1.5">
                        <b>{{ a.actorName || '—' }}</b>
                        @if (a.actorRole; as role) {
                          <bt-staff-role-badge [role]="role" />
                        }
                      </div>
                    </td>
                    <td class="bo-td">{{ 'auditAction.' + a.action | t }}</td>
                    <td class="bo-td wrap-anywhere">{{ a.targetLabel || '—' }}</td>
                    <td class="bo-td wrap-anywhere text-ink-muted">{{ a.detail || '' }}</td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
        </section>
      }
    </div>
  `
})
export class AdminAuditComponent {
  private readonly adminApi = inject(AdminApi);

  protected readonly actions = apiResource<AdminAction[]>(() => this.adminApi.auditUrl());
}
