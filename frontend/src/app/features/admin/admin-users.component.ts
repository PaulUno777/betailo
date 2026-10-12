import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { AdminUser } from '../../domain/models';
import { AdminApi } from '../../data-access/admin/admin.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { apiResource } from '../../core/http/api-resource';

/** `/admin/users` — per `Admin.dc.html`'s user table: verify/suspend,
 * backed directly by `GET /admin/users` + the two POST actions. */
@Component({
  selector: 'bt-admin-users',
  imports: [BadgeComponent, StatusBadgeComponent, ButtonComponent, DialogComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="p-6 sm:p-8">
      <h1 class="text-heading-2">{{ 'adminUsers.title' | t }}</h1>

      @if (users.isLoading()) {
        <p class="mt-6 text-body text-ink-muted">{{ 'adminUsers.loading' | t }}</p>
      } @else if (users.error()) {
        <div class="mt-6 flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="users.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      } @else if ((users.value() ?? []).length === 0) {
        <p class="mt-6 text-body text-ink-muted">{{ 'adminUsers.empty' | t }}</p>
      } @else {
        <div class="mt-6 flex flex-col gap-2">
          @for (u of users.value()!; track u.id) {
            <div class="flex flex-wrap items-center gap-4 rounded-lg border border-line bg-surface-card p-4">
              <span class="min-w-[200px] flex-1 wrap-anywhere text-body-strong">{{ u.email }}</span>
              <bt-badge tone="stone">{{ 'role.' + u.role | t }}</bt-badge>
              <bt-status-badge kind="trust" [status]="u.trustScore" />
              @if (u.isVerified) {
                <bt-badge tone="forest-solid">{{ 'adminUsers.verifiedBadge' | t }}</bt-badge>
              }
              @if (u.suspended) {
                <bt-badge tone="alert">{{ 'adminUsers.suspendedBadge' | t }}</bt-badge>
              }
              <div class="flex gap-2">
                @if (!u.isVerified) {
                  <bt-button variant="outline" size="sm" [disabled]="acting() === u.id" (click)="verify(u.id)">
                    {{ 'adminUsers.verifyButton' | t }}
                  </bt-button>
                }
                @if (!u.suspended) {
                  <bt-button variant="ghost" size="sm" [disabled]="acting() === u.id" (click)="suspendTarget.set(u)">
                    {{ 'adminUsers.suspendButton' | t }}
                  </bt-button>
                }
              </div>
            </div>
          }
        </div>
      }
    </div>

    <bt-dialog
      [open]="!!suspendTarget()"
      tone="alert"
      [title]="'adminUsers.suspendConfirmTitle' | t: { email: suspendTarget()?.email ?? '' }"
      [primaryLabel]="'adminUsers.suspendButton' | t"
      primaryVariant="danger"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="confirmSuspend()"
      (secondaryClick)="suspendTarget.set(null)"
      (closed)="suspendTarget.set(null)"
    >
      {{ 'adminReports.suspendConfirmBody' | t }}
    </bt-dialog>
  `
})
export class AdminUsersComponent {
  private readonly adminApi = inject(AdminApi);
  private readonly toast = inject(ToastService);

  protected readonly users = apiResource<AdminUser[]>(() => this.adminApi.usersUrl());
  protected readonly acting = signal<string | null>(null);
  /** Suspending locks someone out: asked for once, in a dialog. */
  protected readonly suspendTarget = signal<AdminUser | null>(null);

  protected verify(id: string): void {
    this.acting.set(id);
    this.adminApi.verifyUser(id).subscribe({
      next: () => {
        this.acting.set(null);
        this.users.reload();
      },
      error: (err: AppError) => {
        this.acting.set(null);
        this.toast.show({ tone: 'danger', title: 'adminUsers.actionFailedTitle', detail: err.message });
      }
    });
  }

  protected confirmSuspend(): void {
    const target = this.suspendTarget();
    this.suspendTarget.set(null);
    if (target) this.suspend(target.id);
  }

  private suspend(id: string): void {
    this.acting.set(id);
    this.adminApi.suspendUser(id).subscribe({
      next: () => {
        this.acting.set(null);
        this.users.reload();
      },
      error: (err: AppError) => {
        this.acting.set(null);
        this.toast.show({ tone: 'danger', title: 'adminUsers.actionFailedTitle', detail: err.message });
      }
    });
  }
}
