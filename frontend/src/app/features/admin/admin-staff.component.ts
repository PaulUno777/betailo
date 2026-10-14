import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Observable } from 'rxjs';
import { StaffSpaceService } from '../../core/auth/staff-space.service';
import { AppError } from '../../core/http/app-error';
import { apiResource } from '../../core/http/api-resource';
import { LocaleService } from '../../core/i18n/locale.service';
import { elapsed } from './ui/elapsed';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ToastService } from '../../core/notifications/toast.service';
import { AdminApi } from '../../data-access/admin/admin.api';
import { Permission, StaffRole } from '../../domain/enums';
import { AdminUser, FarmLimits, StaffInvitation, StaffOverview } from '../../domain/models';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
import { AvatarComponent } from '../../shared/ui/avatar/avatar.component';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { QuantityStepperComponent } from '../../shared/ui/quantity-stepper/quantity-stepper.component';
import { SelectFieldComponent } from '../../shared/ui/select-field/select-field.component';
import { StaffRoleBadgeComponent } from '../../shared/ui/staff-role-badge/staff-role-badge.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { AdminPageHeaderComponent } from './ui/admin-page-header.component';

/** `/admin/staff` — "Équipe & rôles", per `AdminStaff.dc.html`: who is
 * on the team, what each role may do, and the farms-per-account limit
 * with its exceptions.
 *
 * Nothing here is decided by the page. The role table is the API's own
 * (`GET /admin/staff`), drawn as returned; adding someone gives a role
 * to an account that already exists (there are no invitations); the
 * platform's default limit comes from the deployment and is read-only.
 * The API refuses what must not happen (touching the root account,
 * changing your own role, removing the last super admin, an exception
 * below the farms already owned) and its message is shown as is. */
@Component({
  selector: 'bt-admin-staff',
  imports: [
    FormsModule,
    AdminPageHeaderComponent,
    AvatarComponent,
    BadgeComponent,
    ButtonComponent,
    DialogComponent,
    IconComponent,
    QuantityStepperComponent,
    SelectFieldComponent,
    StaffRoleBadgeComponent,
    TextFieldComponent,
    LocalDatePipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-[22px] p-4 sm:p-7 lg:px-9">
      <bt-admin-page-header [eyebrow]="'adminNav.group.system' | t" [title]="'adminStaff.title' | t" [subtitle]="'adminStaff.subtitle' | t">
        @if (canManage()) {
          <bt-button variant="solid" (click)="openGrant()">
            <bt-icon name="plus" [size]="18" />
            {{ 'adminStaff.addMember' | t }}
          </bt-button>
        }
      </bt-admin-page-header>

      @if (team.value(); as t) {
        <section class="bo-card px-2 pt-[18px] pb-2">
          <div class="flex flex-wrap items-baseline justify-between gap-2 px-3.5 pb-3">
            <h2 class="font-display text-[19px] font-bold">{{ 'adminStaff.membersTitle' | t: { count: t.members.length } }}</h2>
            <span class="text-[12.5px] text-ink-muted">{{ 'adminStaff.membersHint' | t }}</span>
          </div>
          <div class="overflow-x-auto">
            <table class="bo-table min-w-[820px]">
              <thead>
                <tr>
                  <th class="bo-th" scope="col">{{ 'adminStaff.colMember' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminStaff.colRole' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminStaff.colActivity' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminStaff.colMfa' | t }}</th>
                  <th class="bo-th" scope="col"><span class="sr-only">{{ 'adminUsers.colActions' | t }}</span></th>
                </tr>
              </thead>
              <tbody>
                @for (m of t.members; track m.id) {
                  <tr>
                    <td class="bo-td">
                      <div class="flex items-center gap-3">
                        <bt-avatar [name]="m.displayName || m.email" [size]="34" tone="sky" class="shrink-0" />
                        <div class="flex min-w-0 flex-col">
                          <b class="wrap-anywhere">{{ m.displayName || m.email }}</b>
                          @if (m.displayName) {
                            <span class="wrap-anywhere text-caption text-ink-muted">{{ m.email }}</span>
                          }
                        </div>
                      </div>
                    </td>
                    <td class="bo-td">
                      <div class="flex flex-wrap items-center gap-1.5">
                        @if (m.staffRole; as role) {
                          <bt-staff-role-badge [role]="role" />
                        }
                        @if (m.isRoot) {
                          <bt-badge tone="ink">{{ 'adminStaff.rootBadge' | t }}</bt-badge>
                        }
                        @if (m.suspended) {
                          <bt-badge tone="alert">{{ 'adminUsers.suspendedBadge' | t }}</bt-badge>
                        }
                      </div>
                    </td>
                    <td class="bo-td !text-[13px] whitespace-nowrap text-ink-muted">
                      {{ m.lastActiveAt ? ('adminOverview.ago' | t: { age: since(m.lastActiveAt) }) : ('adminStaff.neverActive' | t) }}
                    </td>
                    <td class="bo-td">
                      <span class="inline-flex h-6 items-center rounded-full px-2.5 text-caption font-semibold" [class]="m.mfaEnabled ? 'bg-forest-soft text-forest' : 'bg-amber-soft text-amber'">
                        {{ (m.mfaEnabled ? 'adminStaff.mfaOn' : 'adminStaff.mfaPending') | t }}
                      </span>
                    </td>
                    <td class="bo-td">
                      <div class="flex justify-end gap-2">
                        @if (m.id === me()) {
                          <span class="text-[13px] text-ink-muted">{{ 'adminStaff.you' | t }}</span>
                        } @else if (m.isRoot) {
                          <span class="flex items-center gap-1.5 text-[13px] text-ink-muted"><bt-icon name="lock" [size]="14" />{{ 'adminStaff.rootLocked' | t }}</span>
                        } @else if (canManage()) {
                          <bt-button variant="outline" size="sm" [disabled]="busy()" (click)="openChange(m)">{{ 'adminStaff.changeRole' | t }}</bt-button>
                          <bt-button variant="ghost" size="sm" [disabled]="busy()" (click)="revokeTarget.set(m)">{{ 'adminStaff.revoke' | t }}</bt-button>
                        }
                      </div>
                    </td>
                  </tr>
                }
                @for (inv of t.invitations; track inv.id) {
                  <tr>
                    <td class="bo-td">
                      <div class="flex items-center gap-3">
                        <bt-avatar [name]="inv.email" [size]="34" tone="amber" class="shrink-0" />
                        <div class="flex min-w-0 flex-col">
                          <b class="wrap-anywhere">{{ inv.email }}</b>
                          @if (inv.inviterName) {
                            <span class="text-caption text-ink-muted">{{ 'adminStaff.invitedBy' | t: { name: inv.inviterName } }}</span>
                          }
                        </div>
                      </div>
                    </td>
                    <td class="bo-td"><bt-staff-role-badge [role]="inv.role" /></td>
                    <td class="bo-td !text-[13px] whitespace-nowrap text-ink-muted">{{ 'adminStaff.invitationSent' | t }}</td>
                    <td class="bo-td">
                      <span class="inline-flex h-6 items-center rounded-full bg-amber-soft px-2.5 text-caption font-semibold text-amber">{{ 'adminStaff.mfaPending' | t }}</span>
                    </td>
                    <td class="bo-td">
                      @if (canManage()) {
                        <div class="flex justify-end gap-2">
                          <bt-button variant="outline" size="sm" [disabled]="busy()" (click)="resend(inv)">{{ 'adminStaff.invitationResend' | t }}</bt-button>
                          <bt-button variant="ghost" size="sm" [disabled]="busy()" (click)="cancel(inv)">{{ 'common.cancel' | t }}</bt-button>
                        </div>
                      }
                    </td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
        </section>

        <section class="bo-card px-2 pt-[18px] pb-2">
          <div class="flex flex-col gap-1 px-3.5 pb-3">
            <h2 class="font-display text-[19px] font-bold">{{ 'adminStaff.matrixTitle' | t }}</h2>
            <p class="text-[12.5px] text-ink-muted">{{ 'adminStaff.matrixHint' | t }}</p>
          </div>
          <div class="overflow-x-auto">
            <table class="bo-table min-w-[620px]">
              <thead>
                <tr>
                  <th class="bo-th" scope="col">{{ 'adminStaff.colPermission' | t }}</th>
                  @for (r of t.roles; track r.role) {
                    <th class="bo-th !text-center" scope="col">{{ 'staffRole.' + r.role | t }}</th>
                  }
                </tr>
              </thead>
              <tbody>
                @for (p of t.permissions; track p) {
                  <tr>
                    <th class="bo-td text-left font-normal" scope="row">{{ 'permission.' + p | t }}</th>
                    @for (r of t.roles; track r.role) {
                      <td class="bo-td text-center">
                        @if (r.permissions.includes(p)) {
                          <bt-icon name="check" [size]="18" class="inline-block text-forest" />
                          <span class="sr-only">{{ 'adminStaff.allowed' | t }}</span>
                        } @else {
                          <span class="text-line-strong" aria-hidden="true">—</span>
                          <span class="sr-only">{{ 'adminStaff.notAllowed' | t }}</span>
                        }
                      </td>
                    }
                  </tr>
                }
              </tbody>
            </table>
          </div>
        </section>
      } @else if (team.error()) {
        <div class="flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="team.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      } @else {
        <div class="h-64 animate-pulse rounded-2xl bg-surface-sunken" aria-hidden="true"></div>
      }

      @if (staff.can('accounts.read')) {
        <section id="farm-limits" class="bo-card flex flex-col gap-4 p-4 sm:p-[22px]">
          <div class="flex flex-col gap-1">
            <h2 class="font-display text-[19px] font-bold">{{ 'adminStaff.limitsTitle' | t }}</h2>
            <p class="text-[12.5px] text-ink-muted">{{ 'adminStaff.limitsHint' | t }}</p>
          </div>
          @if (limits.value(); as l) {
            <div class="grid gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
              <div class="flex flex-col gap-1.5 self-start rounded-[14px] bg-surface-sunken p-4">
                <span class="text-eyebrow text-ink-muted">{{ 'adminStaff.limitDefault' | t }}</span>
                <span class="font-display text-[38px] leading-none font-extrabold">{{ l.default }}</span>
                <span class="font-mono text-caption text-ink-muted">MAX_FARMS_PER_ACCOUNT</span>
                <span class="flex items-center gap-1.5 text-caption text-ink-muted"><bt-icon name="lock" [size]="14" />{{ 'adminStaff.limitReadOnly' | t }}</span>
              </div>
              <div class="flex min-w-0 flex-col gap-3">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <b class="text-[15px]">{{ 'adminStaff.exceptionsTitle' | t: { count: l.exceptions.length } }}</b>
                  @if (staff.can('accounts.farm_limit')) {
                    <bt-button variant="outline" size="sm" (click)="openException()">{{ 'adminStaff.addException' | t }}</bt-button>
                  }
                </div>
                @if (l.exceptions.length === 0) {
                  <p class="rounded-[14px] border border-dashed border-line-strong p-5 text-small text-ink-muted">{{ 'adminStaff.exceptionsEmpty' | t }}</p>
                } @else {
                  <ul class="flex flex-col gap-2">
                    @for (ex of l.exceptions; track ex.id) {
                      <li class="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-[14px] border border-line p-3.5">
                        <div class="flex min-w-[200px] flex-1 flex-col gap-0.5">
                          <b class="wrap-anywhere text-small">{{ ex.displayName || ex.email }}</b>
                          @if (ex.farmLimitNote; as note) {
                            <span class="wrap-anywhere text-[13px]">« {{ note.reason }} »</span>
                            <span class="text-caption text-ink-muted">
                              {{ 'adminStaff.exceptionBy' | t: { name: note.setByName || '—', date: (note.setAt | localDate: 'dayMonthYear') } }}
                              @if (note.setByRole; as role) {
                                · {{ 'staffRole.' + role | t }}
                              }
                            </span>
                          }
                        </div>
                        <span class="font-mono text-small tabular-nums whitespace-nowrap">{{ 'adminStaff.exceptionUsage' | t: { owned: ex.farmCount, limit: ex.farmLimit } }}</span>
                        @if (staff.can('accounts.farm_limit')) {
                          <bt-button variant="ghost" size="sm" [disabled]="busy()" (click)="endTarget.set(ex)">{{ 'adminStaff.endException' | t }}</bt-button>
                        }
                      </li>
                    }
                  </ul>
                }
              </div>
            </div>
          } @else if (limits.error()) {
            <div class="flex flex-col items-start gap-3" role="alert">
              <p class="text-body">{{ 'apiErrors.server' | t }}</p>
              <bt-button variant="outline" (click)="limits.reload()">{{ 'common.retry' | t }}</bt-button>
            </div>
          } @else {
            <div class="h-32 animate-pulse rounded-[14px] bg-surface-sunken" aria-hidden="true"></div>
          }
        </section>
      }
    </div>

    <bt-dialog
      [open]="!!roleDialog()"
      tone="forest"
      [title]="(roleDialog() === 'grant' ? 'adminStaff.grantTitle' : 'adminStaff.changeTitle') | t: { name: changeTarget()?.displayName || changeTarget()?.email || '' }"
      [primaryLabel]="(roleDialog() === 'grant' ? 'adminStaff.grantConfirm' : 'adminStaff.changeConfirm') | t"
      primaryVariant="solid"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="saveRole()"
      (secondaryClick)="roleDialog.set(null)"
      (closed)="roleDialog.set(null)"
    >
      <div class="flex flex-col gap-3">
        @if (roleDialog() === 'grant') {
          <p>{{ 'adminStaff.grantBody' | t }}</p>
          <bt-text-field
            [label]="'adminStaff.emailLabel' | t"
            type="email"
            autocomplete="off"
            markRequired
            [ngModel]="email()"
            (ngModelChange)="email.set($event); formError.set('')"
            [error]="formError() ? (formError() | t) : undefined"
          />
        }
        <bt-select-field [label]="'adminStaff.roleLabel' | t" [options]="roleOptions()" [ngModel]="role()" (ngModelChange)="role.set($event)" />
        <p class="text-small text-ink-muted">{{ 'adminStaff.roleSummary.' + role() | t }}</p>
      </div>
    </bt-dialog>

    <bt-dialog
      [open]="!!sentInvitation()"
      [title]="'adminStaff.invitationSentTitle' | t: { email: sentInvitation()?.email ?? '' }"
      [primaryLabel]="'adminStaff.invitationCopy' | t"
      primaryVariant="solid"
      [secondaryLabel]="'common.close' | t"
      (primaryClick)="copyLink()"
      (secondaryClick)="sentInvitation.set(null)"
      (closed)="sentInvitation.set(null)"
    >
      <div class="flex flex-col gap-3">
        <p>{{ (sentInvitation()?.mailed ? 'adminStaff.invitationMailed' : 'adminStaff.invitationNotMailed') | t }}</p>
        <span class="rounded-md bg-surface-sunken px-3 py-2 font-mono text-caption wrap-anywhere select-all">{{ sentInvitation()?.link }}</span>
        <p class="text-small text-ink-muted">{{ 'adminStaff.invitationLinkOnce' | t }}</p>
      </div>
    </bt-dialog>

    <bt-dialog
      [open]="!!revokeTarget()"
      tone="alert"
      [title]="'adminStaff.revokeTitle' | t: { name: revokeTarget()?.displayName || revokeTarget()?.email || '' }"
      [primaryLabel]="'adminStaff.revoke' | t"
      primaryVariant="danger"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="confirmRevoke()"
      (secondaryClick)="revokeTarget.set(null)"
      (closed)="revokeTarget.set(null)"
    >
      {{ 'adminStaff.revokeBody' | t }}
    </bt-dialog>

    <bt-dialog
      [open]="exceptionOpen()"
      tone="forest"
      [title]="'adminStaff.exceptionTitle' | t"
      [primaryLabel]="'adminStaff.exceptionConfirm' | t"
      primaryVariant="solid"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="saveException()"
      (secondaryClick)="exceptionOpen.set(false)"
      (closed)="exceptionOpen.set(false)"
    >
      <div class="flex flex-col gap-3">
        <bt-text-field
          [label]="'adminStaff.exceptionAccount' | t"
          type="email"
          autocomplete="off"
          markRequired
          [ngModel]="email()"
          (ngModelChange)="email.set($event); formError.set('')"
          [error]="formError() === 'apiErrors.admin.account_required' ? (formError() | t) : undefined"
        />
        <div class="flex flex-col gap-1.5">
          <span class="text-small text-ink-muted">{{ 'adminStaff.exceptionLimit' | t }}</span>
          <bt-quantity-stepper [(value)]="limit" [min]="1" [max]="limits.value()?.max ?? 50" />
        </div>
        <bt-text-field
          [label]="'adminStaff.limitReasonLabel' | t"
          markRequired
          [hint]="'adminStaff.exceptionFloor' | t"
          [ngModel]="reason()"
          (ngModelChange)="reason.set($event); formError.set('')"
          [error]="formError() === 'apiErrors.admin.farm_limit_reason_required' ? (formError() | t) : undefined"
        />
      </div>
    </bt-dialog>

    <bt-dialog
      [open]="!!endTarget()"
      tone="alert"
      [title]="'adminStaff.endTitle' | t: { name: endTarget()?.displayName || endTarget()?.email || '' }"
      [primaryLabel]="'adminStaff.endException' | t"
      primaryVariant="danger"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="confirmEnd()"
      (secondaryClick)="endTarget.set(null)"
      (closed)="endTarget.set(null)"
    >
      {{ 'adminStaff.endBody' | t: { default: limits.value()?.default ?? 1 } }}
    </bt-dialog>
  `
})
export class AdminStaffComponent {
  private readonly adminApi = inject(AdminApi);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);

  protected since(iso: string): string {
    return elapsed(iso, this.locale);
  }
  protected readonly staff = inject(StaffSpaceService);

  protected readonly team = apiResource<StaffOverview>(() => this.adminApi.staffUrl());
  protected readonly limits = apiResource<FarmLimits>(() => (this.staff.can('accounts.read') ? this.adminApi.farmLimitsUrl() : undefined));

  protected readonly canManage = computed(() => this.staff.can('staff.manage' satisfies Permission));
  /** The signed-in member's own row is marked, not editable. */
  protected readonly me = computed(() => this.team.value()?.members.find((m) => m.email === this.staff.email())?.id ?? '');

  protected readonly busy = signal(false);
  protected readonly roleDialog = signal<'grant' | 'change' | null>(null);
  protected readonly changeTarget = signal<AdminUser | null>(null);
  protected readonly revokeTarget = signal<AdminUser | null>(null);
  protected readonly endTarget = signal<AdminUser | null>(null);
  protected readonly exceptionOpen = signal(false);

  protected readonly email = signal('');
  protected readonly role = signal<StaffRole>('support');
  protected readonly limit = signal(2);
  protected readonly reason = signal('');
  protected readonly formError = signal('');

  protected readonly roleOptions = computed(() =>
    (this.team.value()?.roles ?? []).map((r) => ({ value: r.role, label: this.locale.t(`staffRole.${r.role}`) }))
  );

  protected openGrant(): void {
    this.email.set('');
    this.role.set('support');
    this.formError.set('');
    this.changeTarget.set(null);
    this.roleDialog.set('grant');
  }

  protected openChange(member: AdminUser): void {
    this.role.set(member.staffRole ?? 'support');
    this.changeTarget.set(member);
    this.roleDialog.set('change');
  }

  protected saveRole(): void {
    const mode = this.roleDialog();
    if (mode === 'grant') {
      if (!this.email().trim()) {
        this.formError.set('apiErrors.admin.account_required');
        return;
      }
      this.roleDialog.set(null);
      this.invite(this.adminApi.inviteStaff(this.email().trim(), this.role()));
    } else if (mode === 'change') {
      const target = this.changeTarget();
      this.roleDialog.set(null);
      if (target) this.run(this.adminApi.setStaffRole(target.id, this.role()), 'adminStaff.changed', () => this.team.reload());
    }
  }

  /** The invitation just sent or renewed, with its link, shown once. */
  protected readonly sentInvitation = signal<StaffInvitation | null>(null);

  private invite(request: Observable<StaffInvitation>): void {
    this.busy.set(true);
    request.subscribe({
      next: (sent) => {
        this.busy.set(false);
        this.team.reload();
        this.sentInvitation.set(sent);
      },
      error: (err: AppError) => {
        this.busy.set(false);
        this.toast.show({ tone: 'danger', title: 'adminStaff.invitationFailed', detail: err.message });
      }
    });
  }

  protected resend(invitation: StaffInvitation): void {
    this.invite(this.adminApi.resendInvitation(invitation.id));
  }

  protected cancel(invitation: StaffInvitation): void {
    this.run(this.adminApi.cancelInvitation(invitation.id), 'adminStaff.invitationCancelled', () => this.team.reload());
  }

  protected copyLink(): void {
    const link = this.sentInvitation()?.link;
    if (!link) return;
    navigator.clipboard?.writeText(link).then(
      () => this.toast.show({ tone: 'success', title: 'adminStaff.invitationCopied' }),
      () => undefined
    );
  }

  protected confirmRevoke(): void {
    const target = this.revokeTarget();
    this.revokeTarget.set(null);
    if (target) this.run(this.adminApi.setStaffRole(target.id, ''), 'adminStaff.revoked', () => this.team.reload());
  }

  protected openException(): void {
    this.email.set('');
    this.reason.set('');
    this.formError.set('');
    this.limit.set((this.limits.value()?.default ?? 1) + 1);
    this.exceptionOpen.set(true);
  }

  protected saveException(): void {
    if (!this.email().trim()) {
      this.formError.set('apiErrors.admin.account_required');
      return;
    }
    if (!this.reason().trim()) {
      this.formError.set('apiErrors.admin.farm_limit_reason_required');
      return;
    }
    this.exceptionOpen.set(false);
    this.run(this.adminApi.setFarmLimitByEmail(this.email().trim(), this.limit(), this.reason().trim()), 'adminStaff.exceptionSaved', () => this.limits.reload());
  }

  protected confirmEnd(): void {
    const target = this.endTarget();
    this.endTarget.set(null);
    if (target) this.run(this.adminApi.setFarmLimit(target.id, 0, ''), 'adminStaff.exceptionEnded', () => this.limits.reload());
  }

  private run(request: Observable<unknown>, doneKey: string, then: () => void): void {
    this.busy.set(true);
    request.subscribe({
      next: () => {
        this.busy.set(false);
        this.toast.show({ tone: 'success', title: doneKey });
        then();
      },
      error: (err: AppError) => {
        this.busy.set(false);
        this.toast.show({ tone: 'danger', title: 'adminUsers.actionFailedTitle', detail: err.message });
      }
    });
  }
}
