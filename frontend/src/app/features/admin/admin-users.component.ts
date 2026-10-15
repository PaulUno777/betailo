import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Observable } from 'rxjs';
import { StaffSpaceService } from '../../core/auth/staff-space.service';
import { AdminUser } from '../../domain/models';
import { AdminApi } from '../../data-access/admin/admin.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
import { AvatarComponent } from '../../shared/ui/avatar/avatar.component';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { QuantityStepperComponent } from '../../shared/ui/quantity-stepper/quantity-stepper.component';
import { StaffRoleBadgeComponent } from '../../shared/ui/staff-role-badge/staff-role-badge.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { apiResource } from '../../core/http/api-resource';
import { AdminPageHeaderComponent } from './ui/admin-page-header.component';

type Filter = 'all' | 'sellers' | 'staff' | 'suspended';
const FILTERS: readonly Filter[] = ['all', 'sellers', 'staff', 'suspended'];

/** `/admin/users` — "Comptes": every account, newest first, in the
 * back office's table. Two actions live here, each offered only to a
 * role the API lets through: suspending (or reactivating) an account,
 * and making an exception to the farms-per-account limit, which needs
 * a reason. Staff accounts and the root are marked; who may touch them
 * is the API's rule (`admin.root_protected`,
 * `admin.staff_needs_super_admin`), surfaced as its message. */
@Component({
  selector: 'bt-admin-users',
  imports: [
    FormsModule,
    AdminPageHeaderComponent,
    AvatarComponent,
    BadgeComponent,
    StatusBadgeComponent,
    StaffRoleBadgeComponent,
    ButtonComponent,
    DialogComponent,
    QuantityStepperComponent,
    TextFieldComponent,
    LocalDatePipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-[22px] p-4 sm:p-7 lg:px-9">
      <bt-admin-page-header [eyebrow]="'adminNav.group.steer' | t" [title]="'adminUsers.title' | t" [subtitle]="'adminUsers.subtitle' | t" />

      <div class="flex flex-wrap items-center gap-3">
        <div class="flex flex-wrap gap-1.5" role="group" [attr.aria-label]="'adminUsers.filterLabel' | t">
          @for (f of filters; track f) {
            <button
              type="button"
              [attr.aria-pressed]="filter() === f"
              (click)="filter.set(f)"
              class="inline-flex h-9 items-center rounded-full border px-3 text-caption font-semibold whitespace-nowrap transition"
              [class]="filter() === f ? 'border-ink bg-ink text-surface' : 'border-line bg-surface-card text-ink hover:border-line-strong'"
            >
              {{ 'adminUsers.filter.' + f | t }} · {{ counts()[f] }}
            </button>
          }
        </div>
        <label class="ml-auto flex h-11 w-full items-center gap-2 rounded-[10px] border border-line-strong bg-surface-card px-3 sm:w-72">
          <span class="sr-only">{{ 'adminUsers.searchLabel' | t }}</span>
          <input
            type="search"
            class="w-full bg-transparent text-base outline-none placeholder:text-ink-muted sm:text-small"
            [placeholder]="'adminUsers.searchLabel' | t"
            [ngModel]="query()"
            (ngModelChange)="query.set($event)"
          />
        </label>
      </div>

      @if (users.isLoading() && !users.hasValue()) {
        <div class="h-72 animate-pulse rounded-2xl bg-surface-sunken" aria-hidden="true"></div>
      } @else if (users.error()) {
        <div class="flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="users.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      } @else {
        <section class="bo-card px-2 pt-[18px] pb-2">
          <p class="px-3.5 pb-3 text-[12.5px] text-ink-muted" aria-live="polite">{{ 'adminUsers.resultCount' | t: { count: shown().length } }}</p>
          @if (shown().length === 0) {
            <p class="px-3.5 pb-6 text-body text-ink-muted">{{ 'adminUsers.empty' | t }}</p>
          } @else {
            <div class="overflow-x-auto">
              <table class="bo-table min-w-[860px]">
                <thead>
                  <tr>
                    <th class="bo-th" scope="col">{{ 'adminUsers.colAccount' | t }}</th>
                    <th class="bo-th" scope="col">{{ 'adminUsers.colSince' | t }}</th>
                    <th class="bo-th" scope="col">{{ 'adminUsers.colFarms' | t }}</th>
                    <th class="bo-th" scope="col">{{ 'adminUsers.colTrust' | t }}</th>
                    <th class="bo-th" scope="col">{{ 'adminUsers.colState' | t }}</th>
                    <th class="bo-th" scope="col"><span class="sr-only">{{ 'adminUsers.colActions' | t }}</span></th>
                  </tr>
                </thead>
                <tbody>
                  @for (u of shown(); track u.id) {
                    <tr>
                      <td class="bo-td">
                        <div class="flex items-center gap-3">
                          <bt-avatar [name]="u.displayName || u.email" [src]="u.avatarUrl" [size]="34" [tone]="u.staffRole ? 'sky' : 'clay'" class="shrink-0" />
                          <div class="flex min-w-0 flex-col">
                            <b class="wrap-anywhere">{{ u.displayName || u.email }}</b>
                            @if (u.displayName) {
                              <span class="wrap-anywhere text-caption text-ink-muted">{{ u.email }}</span>
                            }
                          </div>
                        </div>
                      </td>
                      <td class="bo-td whitespace-nowrap text-ink-muted">{{ u.memberSince | localDate: 'dayMonthYear' }}</td>
                      <td class="bo-td whitespace-nowrap">
                        @if (u.staffRole) {
                          <span class="text-ink-muted">—</span>
                        } @else {
                          <span class="font-mono tabular-nums">{{ u.farmCount }}</span>
                          @if (u.farmLimit) {
                            <span class="font-mono tabular-nums text-ink-muted"> / {{ u.farmLimit }}</span>
                            <bt-badge tone="clay" class="ml-1.5">{{ 'adminUsers.exception' | t }}</bt-badge>
                          }
                        }
                      </td>
                      <td class="bo-td">
                        <!-- A trust score belongs to a seller; an account that never sold has none. -->
                        @if (u.farmCount > 0) {
                          <bt-status-badge kind="trust" [status]="u.trustScore" />
                        } @else {
                          <span class="text-ink-muted">—</span>
                        }
                      </td>
                      <td class="bo-td">
                        <div class="flex flex-wrap items-center gap-1.5">
                          @if (u.suspended) {
                            <bt-badge tone="alert">{{ 'adminUsers.suspendedBadge' | t }}</bt-badge>
                          } @else if (!u.staffRole) {
                            <bt-badge tone="forest">{{ 'adminUsers.activeBadge' | t }}</bt-badge>
                          }
                          @if (u.staffRole; as role) {
                            <bt-staff-role-badge [role]="role" />
                          }
                          @if (u.isRoot) {
                            <bt-badge tone="ink">{{ 'adminStaff.rootBadge' | t }}</bt-badge>
                          }
                        </div>
                      </td>
                      <td class="bo-td">
                        <div class="flex justify-end gap-2">
                          @if (!u.staffRole && staff.can('accounts.farm_limit')) {
                            <bt-button variant="outline" size="sm" [disabled]="acting() === u.id" (click)="openLimit(u)">{{ 'adminUsers.farmLimitButton' | t }}</bt-button>
                          }
                          @if (!u.isRoot && staff.can('accounts.suspend')) {
                            @if (u.suspended) {
                              <bt-button variant="ghost" size="sm" [disabled]="acting() === u.id" (click)="reactivate(u)">{{ 'adminUsers.reactivateButton' | t }}</bt-button>
                            } @else {
                              <bt-button variant="ghost" size="sm" [disabled]="acting() === u.id" (click)="suspendTarget.set(u)">{{ 'adminUsers.suspendButton' | t }}</bt-button>
                            }
                          }
                        </div>
                      </td>
                    </tr>
                  }
                </tbody>
              </table>
            </div>
          }
        </section>
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
        <p>{{ 'adminUsers.farmLimitBody' | t: { owned: limitTarget()?.farmCount ?? 0 } }}</p>
        <bt-quantity-stepper [(value)]="limit" [min]="0" [max]="maxLimit" />
        <p class="text-small">{{ (limit() === 0 ? 'adminUsers.farmLimitZero' : 'adminUsers.farmLimitChosen') | t: { count: limit() } }}</p>
        @if (limit() > 0) {
          <bt-text-field
            [label]="'adminStaff.limitReasonLabel' | t"
            markRequired
            [ngModel]="limitReason()"
            (ngModelChange)="limitReason.set($event); limitError.set('')"
            [error]="limitError() ? (limitError() | t) : undefined"
          />
        }
      </div>
    </bt-dialog>
  `
})
export class AdminUsersComponent {
  private readonly adminApi = inject(AdminApi);
  private readonly toast = inject(ToastService);
  protected readonly staff = inject(StaffSpaceService);

  /** The API's own cap (admin.MaxFarmLimit); it refuses anything above. */
  protected readonly maxLimit = 50;
  protected readonly filters = FILTERS;

  protected readonly users = apiResource<AdminUser[]>(() => this.adminApi.usersUrl());
  protected readonly filter = signal<Filter>('all');
  protected readonly query = signal('');
  protected readonly acting = signal<string | null>(null);
  /** Suspending locks someone out: asked for once, in a dialog. */
  protected readonly suspendTarget = signal<AdminUser | null>(null);
  protected readonly limitTarget = signal<AdminUser | null>(null);
  protected readonly limit = signal(0);
  protected readonly limitReason = signal('');
  protected readonly limitError = signal('');

  private matches(u: AdminUser, filter: Filter): boolean {
    switch (filter) {
      case 'sellers':
        return u.farmCount > 0;
      case 'staff':
        return !!u.staffRole;
      case 'suspended':
        return u.suspended;
      default:
        return true;
    }
  }

  protected readonly counts = computed<Record<Filter, number>>(() => {
    const all = this.users.value() ?? [];
    return {
      all: all.length,
      sellers: all.filter((u) => this.matches(u, 'sellers')).length,
      staff: all.filter((u) => this.matches(u, 'staff')).length,
      suspended: all.filter((u) => this.matches(u, 'suspended')).length
    };
  });

  /** Narrowing the list already loaded: a display filter, not a rule. */
  protected readonly shown = computed(() => {
    const q = this.query().trim().toLowerCase();
    return (this.users.value() ?? []).filter(
      (u) => this.matches(u, this.filter()) && (!q || u.email.toLowerCase().includes(q) || (u.displayName ?? '').toLowerCase().includes(q))
    );
  });

  protected openLimit(u: AdminUser): void {
    this.limit.set(u.farmLimit);
    this.limitReason.set(u.farmLimitNote?.reason ?? '');
    this.limitError.set('');
    this.limitTarget.set(u);
  }

  protected saveLimit(): void {
    const target = this.limitTarget();
    if (!target) return;
    if (this.limit() > 0 && !this.limitReason().trim()) {
      this.limitError.set('apiErrors.admin.farm_limit_reason_required');
      return;
    }
    this.limitTarget.set(null);
    this.act(target.id, this.adminApi.setFarmLimit(target.id, this.limit(), this.limitReason().trim()), 'adminUsers.farmLimitSaved');
  }

  protected confirmSuspend(): void {
    const target = this.suspendTarget();
    this.suspendTarget.set(null);
    if (target) this.act(target.id, this.adminApi.suspendUser(target.id), 'adminUsers.suspendedToast');
  }

  protected reactivate(u: AdminUser): void {
    this.act(u.id, this.adminApi.reactivateUser(u.id), 'adminUsers.reactivatedToast');
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
