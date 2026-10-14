import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { StaffSpaceService } from '../../core/auth/staff-space.service';
import { PaymentStatus } from '../../domain/enums';
import { AdminEscrowItem } from '../../domain/models';
import { AdminApi } from '../../data-access/admin/admin.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { apiResource } from '../../core/http/api-resource';
import { AdminPageHeaderComponent } from './ui/admin-page-header.component';

type Filter = 'held' | 'all' | 'released' | 'refunded';
const FILTERS: readonly Filter[] = ['held', 'all', 'released', 'refunded'];

/** `/admin/escrow` — "Séquestre": every reservation platform-wide, the
 * ones the platform holds money for first, in the back office's table
 * (`GET /admin/escrow`). Releasing or refunding a held payment moves
 * money and can't be undone: each is confirmed once, and offered only
 * to a role that may settle escrow (the others read). */
@Component({
  selector: 'bt-admin-escrow',
  imports: [AdminPageHeaderComponent, StatusBadgeComponent, ButtonComponent, DialogComponent, FcfaCurrencyPipe, LocalDatePipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-[22px] p-4 sm:p-7 lg:px-9">
      <bt-admin-page-header [eyebrow]="'adminNav.group.steer' | t" [title]="'adminEscrow.title' | t" [subtitle]="(staff.can('escrow.settle') ? 'adminEscrow.subtitle' : 'adminEscrow.subtitleReadOnly') | t" />

      <div class="flex flex-wrap gap-1.5" role="group" [attr.aria-label]="'adminEscrow.filterLabel' | t">
        @for (f of filters; track f) {
          <button
            type="button"
            [attr.aria-pressed]="filter() === f"
            (click)="filter.set(f)"
            class="inline-flex h-9 items-center rounded-full border px-3 text-caption font-semibold whitespace-nowrap transition"
            [class]="filter() === f ? 'border-ink bg-ink text-surface' : 'border-line bg-surface-card text-ink hover:border-line-strong'"
          >
            {{ 'adminEscrow.filter.' + f | t }} · {{ count(f) }}
          </button>
        }
      </div>

      @if (escrow.isLoading() && !escrow.hasValue()) {
        <div class="h-72 animate-pulse rounded-2xl bg-surface-sunken" aria-hidden="true"></div>
      } @else if (escrow.error()) {
        <div class="flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="escrow.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      } @else if (shown().length === 0) {
        <p class="bo-card p-6 text-body text-ink-muted">{{ 'adminEscrow.empty' | t }}</p>
      } @else {
        <section class="bo-card px-2 pt-[18px] pb-2">
          <div class="overflow-x-auto">
            <table class="bo-table min-w-[860px]">
              <thead>
                <tr>
                  <th class="bo-th" scope="col">{{ 'adminEscrow.colReservation' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminEscrow.colBuyer' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminEscrow.colFarm' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminEscrow.colAmount' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminEscrow.colStatus' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminEscrow.colDate' | t }}</th>
                  <th class="bo-th" scope="col"><span class="sr-only">{{ 'adminUsers.colActions' | t }}</span></th>
                </tr>
              </thead>
              <tbody>
                @for (r of shown(); track r.id) {
                  <tr>
                    <td class="bo-td font-mono !text-[13px] whitespace-nowrap">{{ r.reference }}</td>
                    <td class="bo-td wrap-anywhere">{{ r.buyerName || '—' }}</td>
                    <td class="bo-td wrap-anywhere">{{ r.farmName || '—' }}</td>
                    <td class="bo-td font-mono !text-[13px] whitespace-nowrap tabular-nums">{{ r.totalAmount | fcfa }}</td>
                    <td class="bo-td">
                      @if (r.payment; as payment) {
                        <bt-status-badge kind="escrow" [status]="payment.status" />
                      } @else {
                        <bt-status-badge kind="reservation" [status]="r.status" />
                      }
                    </td>
                    <td class="bo-td !text-[13px] whitespace-nowrap text-ink-muted">{{ r.createdAt | localDate: 'dayMonthYear' }}</td>
                    <td class="bo-td">
                      @if (r.payment?.status === 'held' && staff.can('escrow.settle')) {
                        <div class="flex justify-end gap-2">
                          <bt-button variant="outline" size="sm" [disabled]="acting() === r.id" (click)="pending.set({ kind: 'release', reservation: r })">
                            {{ 'adminEscrow.releaseButton' | t }}
                          </bt-button>
                          <bt-button variant="ghost" size="sm" [disabled]="acting() === r.id" (click)="pending.set({ kind: 'refund', reservation: r })">
                            {{ 'adminEscrow.refundButton' | t }}
                          </bt-button>
                        </div>
                      }
                    </td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
        </section>
      }
    </div>

    <bt-dialog
      [open]="!!pending()"
      [tone]="pending()?.kind === 'refund' ? 'alert' : 'forest'"
      [title]="(pending()?.kind === 'refund' ? 'adminEscrow.refundConfirmTitle' : 'adminEscrow.releaseConfirmTitle') | t"
      [primaryLabel]="(pending()?.kind === 'refund' ? 'adminEscrow.refundButton' : 'adminEscrow.releaseButton') | t"
      [primaryVariant]="pending()?.kind === 'refund' ? 'danger' : 'solid'"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="confirmPending()"
      (secondaryClick)="pending.set(null)"
      (closed)="pending.set(null)"
    >
      {{
        (pending()?.kind === 'refund' ? 'adminEscrow.refundConfirmBody' : 'adminEscrow.releaseConfirmBody')
          | t: { amount: (pending()?.reservation?.totalAmount ?? 0 | fcfa) }
      }}
    </bt-dialog>
  `
})
export class AdminEscrowComponent {
  private readonly adminApi = inject(AdminApi);
  private readonly toast = inject(ToastService);
  protected readonly staff = inject(StaffSpaceService);

  protected readonly filters = FILTERS;
  protected readonly filter = signal<Filter>('held');
  protected readonly escrow = apiResource<AdminEscrowItem[]>(() => this.adminApi.escrowUrl());
  protected readonly acting = signal<string | null>(null);
  /** Both actions move money and can't be undone: asked for once. */
  protected readonly pending = signal<{ kind: 'release' | 'refund'; reservation: AdminEscrowItem } | null>(null);

  private matching(filter: Filter): AdminEscrowItem[] {
    const all = this.escrow.value() ?? [];
    return filter === 'all' ? all : all.filter((r) => r.payment?.status === (filter satisfies PaymentStatus));
  }

  protected count(filter: Filter): number {
    return this.matching(filter).length;
  }

  protected readonly shown = computed(() => this.matching(this.filter()));

  protected confirmPending(): void {
    const p = this.pending();
    this.pending.set(null);
    if (!p) return;
    this.settle(p.reservation.id, p.kind);
  }

  private settle(id: string, kind: 'release' | 'refund'): void {
    this.acting.set(id);
    const request = kind === 'refund' ? this.adminApi.refundEscrow(id) : this.adminApi.releaseEscrow(id);
    request.subscribe({
      next: () => {
        this.acting.set(null);
        this.toast.show({ tone: 'success', title: `adminEscrow.done.${kind}` });
        this.escrow.reload();
      },
      error: (err: AppError) => {
        this.acting.set(null);
        this.toast.show({ tone: 'danger', title: 'adminEscrow.actionFailedTitle', detail: err.message });
      }
    });
  }
}
