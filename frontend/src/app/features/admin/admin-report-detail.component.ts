import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';
import { ReportDetail } from '../../domain/models';
import { AdminApi } from '../../data-access/admin/admin.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { apiResource } from '../../core/http/api-resource';

/** `/admin/reports/:id` — per `Moderation.dc.html`'s detail view: the
 * risk score + signal breakdown from `GET /admin/reports/{id}`, plus the
 * three resolve actions. */
@Component({
  selector: 'bt-admin-report-detail',
  imports: [RouterLink, BadgeComponent, ButtonComponent, DialogComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-2xl p-6 sm:p-8">
      <a routerLink="/admin/reports" class="-my-2 inline-flex min-h-11 items-center text-small text-ink-muted hover:text-forest">← {{ 'adminReports.backToReports' | t }}</a>

      @if (detail.value(); as d) {
        <div class="mt-4 flex flex-col gap-2 border-b border-line pb-6">
          <bt-badge tone="stone" class="w-fit">{{ 'reportTarget.' + d.report.targetType | t }}</bt-badge>
          <h1 class="wrap-anywhere text-heading-2">{{ d.report.reason }}</h1>
          <bt-badge tone="amber" class="w-fit">{{ 'reportStatus.' + d.report.status | t }}</bt-badge>
        </div>

        <div class="mt-6 flex items-center gap-4 rounded-lg border border-line bg-surface-card p-5">
          <span class="text-eyebrow text-ink-muted">{{ 'adminReports.riskScoreLabel' | t }}</span>
          <span class="text-number-xl">{{ d.riskScore }}</span>
        </div>

        <section class="mt-6">
          <h2 class="text-heading-4">{{ 'adminReports.signalsTitle' | t }}</h2>
          <div class="mt-3 flex flex-col gap-2">
            @for (s of d.signals; track s.name) {
              <div class="flex items-center justify-between rounded-md border border-line p-3">
                <div class="flex flex-col">
                  <span class="text-body-strong">{{ s.name }}</span>
                  <span class="text-small text-ink-muted">{{ s.detail }}</span>
                </div>
                <span class="text-small text-ink-muted">+{{ s.weight }}</span>
              </div>
            }
          </div>
        </section>

        @if (d.history.length > 0) {
          <section class="mt-6">
            <h2 class="text-heading-4">{{ 'adminReports.historyTitle' | t }}</h2>
            <div class="mt-3 flex flex-col gap-2">
              @for (h of d.history; track h.id) {
                <div class="rounded-md border border-line p-3 text-small text-ink-muted">{{ h.reason }} · {{ 'reportStatus.' + h.status | t }}</div>
              }
            </div>
          </section>
        }

        <div class="mt-6 flex flex-wrap gap-3">
          <bt-button variant="outline" [disabled]="resolving()" (click)="resolve('dismiss')">{{ 'adminReports.resolveDismiss' | t }}</bt-button>
          <bt-button variant="outline" [disabled]="resolving()" (click)="resolve('verify_identity')">{{ 'adminReports.resolveVerify' | t }}</bt-button>
          <bt-button variant="danger" [disabled]="resolving()" (click)="suspendConfirmOpen.set(true)">{{ 'adminReports.resolveSuspend' | t }}</bt-button>
        </div>
      } @else if (detail.isLoading()) {
        <p class="mt-6 text-body text-ink-muted">{{ 'adminReports.loading' | t }}</p>
      } @else if (detail.error()) {
        <p class="mt-6 text-body text-alert">{{ 'adminReports.notFound' | t }}</p>
      }
    </div>

    <bt-dialog
      [open]="suspendConfirmOpen()"
      tone="alert"
      [title]="'adminReports.resolveSuspend' | t"
      [primaryLabel]="'adminReports.resolveSuspend' | t"
      primaryVariant="danger"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="suspendConfirmOpen.set(false); resolve('suspend')"
      (secondaryClick)="suspendConfirmOpen.set(false)"
      (closed)="suspendConfirmOpen.set(false)"
    >
      {{ 'adminReports.suspendConfirmBody' | t }}
    </bt-dialog>
  `
})
export class AdminReportDetailComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly adminApi = inject(AdminApi);
  private readonly toast = inject(ToastService);

  private readonly id = toSignal(this.route.paramMap.pipe(map((p) => p.get('id') ?? '')), { initialValue: '' });

  protected readonly detail = apiResource<ReportDetail>(() => (this.id() ? this.adminApi.reportDetailUrl(this.id()) : undefined));

  protected readonly resolving = signal(false);
  protected readonly suspendConfirmOpen = signal(false);

  protected resolve(action: 'dismiss' | 'verify_identity' | 'suspend'): void {
    this.resolving.set(true);
    this.adminApi.resolveReport(this.id(), action).subscribe({
      next: () => this.router.navigateByUrl('/admin/reports'),
      error: (err: AppError) => {
        this.resolving.set(false);
        this.toast.show({ tone: 'danger', title: 'adminReports.resolveFailedTitle', detail: err.message });
      }
    });
  }
}
