import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { AdminOverview, StaffOverview } from '../../domain/models';
import { AdminApi } from '../../data-access/admin/admin.api';
import { apiResource } from '../../core/http/api-resource';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { AdminPageHeaderComponent } from './ui/admin-page-header.component';

/** `/admin/settings` — the platform's switches, read-only. They are set
 * in the server's configuration and read at startup, so this page shows
 * what the API runs with and where each value is changed; nothing here
 * can be edited. */
@Component({
  selector: 'bt-admin-settings',
  imports: [RouterLink, AdminPageHeaderComponent, ButtonComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-[22px] p-4 sm:p-7 lg:px-9">
      <bt-admin-page-header [eyebrow]="'adminNav.group.system' | t" [title]="'adminSettings.title' | t" [subtitle]="'adminSettings.subtitle' | t" />

      @if (overview.isLoading() && !overview.hasValue()) {
        <div class="h-56 animate-pulse rounded-2xl bg-surface-sunken" aria-busy="true"></div>
      } @else if (overview.error()) {
        <div class="flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="overview.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      } @else if (overview.value(); as o) {
        <section class="bo-card flex max-w-3xl flex-col px-5 py-2">
          <div class="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-line py-4">
            <span class="w-14 font-display text-[36px] leading-none font-extrabold">{{ o.maxFarmsPerAccount }}</span>
            <div class="flex min-w-0 flex-1 flex-col gap-0.5">
              <b class="text-small">{{ 'adminSettings.farmLimit' | t }}</b>
              <span class="font-mono text-caption text-ink-muted">MAX_FARMS_PER_ACCOUNT · {{ 'adminSettings.readOnly' | t }}</span>
            </div>
            <a routerLink="/admin/staff" fragment="farm-limits" class="text-small font-semibold text-forest underline-offset-2 hover:underline">{{ 'adminSettings.farmLimitLink' | t }}</a>
          </div>
          <div class="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-line py-4">
            <span class="w-14 font-display text-[22px] leading-none font-extrabold" [class.text-clay-ink]="o.autoApproveFarms">{{ (o.autoApproveFarms ? 'adminOverview.on' : 'adminOverview.off') | t }}</span>
            <div class="flex min-w-0 flex-1 flex-col gap-0.5">
              <b class="text-small">{{ 'adminSettings.autoApprove' | t }}</b>
              <span class="font-mono text-caption text-ink-muted">FARM_AUTO_APPROVE · {{ 'adminSettings.readOnly' | t }}</span>
              @if (o.autoApproveFarms) {
                <span class="text-caption font-semibold text-clay-ink">{{ 'adminSettings.autoApproveWarning' | t }}</span>
              }
            </div>
          </div>
          <div class="flex flex-wrap items-center gap-x-4 gap-y-2 py-4">
            <span class="w-14 font-display text-[22px] leading-none font-extrabold">1</span>
            <div class="flex min-w-0 flex-1 flex-col gap-0.5">
              <b class="text-small">{{ 'adminSettings.root' | t }}</b>
              <span class="font-mono text-caption wrap-anywhere text-ink-muted">{{ rootEmail() || '—' }} · SUPER_ADMIN_EMAIL</span>
              <span class="text-caption text-ink-muted">{{ 'adminSettings.rootHint' | t }}</span>
            </div>
          </div>
        </section>
      }
    </div>
  `
})
export class AdminSettingsComponent {
  private readonly adminApi = inject(AdminApi);

  protected readonly overview = apiResource<AdminOverview>(() => this.adminApi.overviewUrl('30d'));
  private readonly team = apiResource<StaffOverview>(() => this.adminApi.staffUrl());

  protected readonly rootEmail = computed(() => this.team.value()?.members.find((m) => m.isRoot)?.email ?? '');
}
