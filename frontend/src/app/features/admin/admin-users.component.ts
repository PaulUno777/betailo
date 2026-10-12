import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { Observable } from 'rxjs';
import { AdminUser } from '../../domain/models';
import { AdminApi } from '../../data-access/admin/admin.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { QuantityStepperComponent } from '../../shared/ui/quantity-stepper/quantity-stepper.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { apiResource } from '../../core/http/api-resource';

/** `/admin/users` — every account, from `GET /admin/users`: who is on
 * the team, their trust score, and the two things staff decide about an
 * account: how many farms it may own, and suspending it. (Whether an
 * account can sell is decided farm by farm, on the validation page.) */
@Component({
  selector: 'bt-admin-users',
  imports: [BadgeComponent, StatusBadgeComponent, ButtonComponent, DialogComponent, QuantityStepperComponent, TranslatePipe],
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
            <div class="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-lg border border-line bg-surface-card p-4">
              <div class="flex min-w-[200px] flex-1 flex-col">
                <span class="wrap-anywhere text-body-strong">{{ u.displayName || u.email }}</span>
                @if (u.displayName) {
                  <span class="wrap-anywhere text-small text-ink-muted">{{ u.email }}</span>
                }
              </div>
              @if (u.staffRole) {
                <bt-badge tone="ink">{{ 'staffRole.' + u.staffRole | t }}</bt-badge>
              }
              <!-- A trust score belongs to a seller; an account that never sold has none. -->
              @if (u.trustScore > 0) {
                <bt-status-badge kind="trust" [status]="u.trustScore" />
              }
              @if (u.suspended) {
                <bt-badge tone="alert">{{ 'adminUsers.suspendedBadge' | t }}</bt-badge>
              }
              <div class="flex gap-2">
                @if (!u.staffRole) {
                  <bt-button variant="outline" size="sm" [disabled]="acting() === u.id" (click)="openLimit(u)">
                    {{ (u.farmLimit ? 'adminUsers.farmLimitValue' : 'adminUsers.farmLimitDefault') | t: { count: u.farmLimit } }}
                  </bt-button>
                }
                @if (!u.suspended && !u.staffRole) {
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

    <bt-dialog
      [open]="!!limitTarget()"
      tone="forest"
      [title]="'adminUsers.farmLimitTitle' | t: { email: limitTarget()?.email ?? '' }"
      [primaryLabel]="'adminUsers.farmLimitSave' | t"
      primaryVariant="solid"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="saveLimit()"
      (secondaryClick)="limitTarget.set(null)"
      (closed)="limitTarget.set(null)"
    >
      <div class="flex flex-col gap-3">
        <p>{{ 'adminUsers.farmLimitBody' | t }}</p>
        <bt-quantity-stepper [(value)]="limit" [min]="0" [max]="maxLimit" />
        <p class="text-small">{{ (limit() === 0 ? 'adminUsers.farmLimitZero' : 'adminUsers.farmLimitChosen') | t: { count: limit() } }}</p>
      </div>
    </bt-dialog>
  `
})
export class AdminUsersComponent {
  private readonly adminApi = inject(AdminApi);
  private readonly toast = inject(ToastService);

  /** The API's own cap (admin.MaxFarmLimit); it refuses anything above. */
  protected readonly maxLimit = 50;

  protected readonly users = apiResource<AdminUser[]>(() => this.adminApi.usersUrl());
  protected readonly acting = signal<string | null>(null);
  /** Suspending locks someone out: asked for once, in a dialog. */
  protected readonly suspendTarget = signal<AdminUser | null>(null);
  protected readonly limitTarget = signal<AdminUser | null>(null);
  protected readonly limit = signal(0);

  protected openLimit(u: AdminUser): void {
    this.limit.set(u.farmLimit);
    this.limitTarget.set(u);
  }

  protected saveLimit(): void {
    const target = this.limitTarget();
    this.limitTarget.set(null);
    if (target) this.act(target.id, this.adminApi.setFarmLimit(target.id, this.limit()), 'adminUsers.farmLimitSaved');
  }

  protected confirmSuspend(): void {
    const target = this.suspendTarget();
    this.suspendTarget.set(null);
    if (target) this.act(target.id, this.adminApi.suspendUser(target.id), 'adminUsers.suspendedToast');
  }

  private act(id: string, request: Observable<unknown>, doneKey: string): void {
    this.acting.set(id);
    request.subscribe({
      next: () => {
        this.acting.set(null);
        this.users.reload();
        this.toast.show({ tone: 'success', title: doneKey });
      },
      error: (err: AppError) => {
        this.acting.set(null);
        this.toast.show({ tone: 'danger', title: 'adminUsers.actionFailedTitle', detail: err.message });
      }
    });
  }
}
