import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { httpResource } from '@angular/common/http';
import { AdminUser } from '../../domain/models';
import { AdminApi } from '../../data-access/admin/admin.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';

/** `/admin/users` — per `Admin.dc.html`'s user table: verify/suspend,
 * backed directly by `GET /admin/users` + the two POST actions. */
@Component({
  selector: 'bt-admin-users',
  imports: [BadgeComponent, StatusBadgeComponent, ButtonComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="p-6 sm:p-8">
      <h1 class="text-heading-2">{{ 'adminUsers.title' | t }}</h1>

      @if (users.isLoading()) {
        <p class="mt-6 text-body text-ink-muted">{{ 'adminUsers.loading' | t }}</p>
      } @else if ((users.value() ?? []).length === 0) {
        <p class="mt-6 text-body text-ink-muted">{{ 'adminUsers.empty' | t }}</p>
      } @else {
        <div class="mt-6 flex flex-col gap-2">
          @for (u of users.value()!; track u.id) {
            <div class="flex flex-wrap items-center gap-4 rounded-lg border border-line bg-surface-card p-4">
              <span class="min-w-[200px] flex-1 text-body-strong">{{ u.email }}</span>
              <bt-badge tone="stone">{{ u.role }}</bt-badge>
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
                  <bt-button variant="ghost" size="sm" [disabled]="acting() === u.id" (click)="suspend(u.id)">
                    {{ 'adminUsers.suspendButton' | t }}
                  </bt-button>
                }
              </div>
            </div>
          }
        </div>
      }
    </div>
  `
})
export class AdminUsersComponent {
  private readonly adminApi = inject(AdminApi);
  private readonly toast = inject(ToastService);

  protected readonly users = httpResource<AdminUser[]>(() => this.adminApi.usersUrl());
  protected readonly acting = signal<string | null>(null);

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

  protected suspend(id: string): void {
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
