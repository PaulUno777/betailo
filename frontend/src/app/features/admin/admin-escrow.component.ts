import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { Reservation } from '../../domain/models';
import { AdminApi } from '../../data-access/admin/admin.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { apiResource } from '../../core/http/api-resource';

/** `/admin/escrow` — every reservation with a payment, plus the two
 * admin-only escrow actions (`release`/`refund`), from `GET
 * /admin/escrow`. */
@Component({
  selector: 'bt-admin-escrow',
  imports: [StatusBadgeComponent, ButtonComponent, DialogComponent, FcfaCurrencyPipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="p-6 sm:p-8">
      <h1 class="text-heading-2">{{ 'adminEscrow.title' | t }}</h1>

      @if (escrow.isLoading()) {
        <p class="mt-6 text-body text-ink-muted">{{ 'adminEscrow.loading' | t }}</p>
      } @else if (escrow.error()) {
        <div class="mt-6 flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="escrow.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      } @else if ((escrow.value() ?? []).length === 0) {
        <p class="mt-6 text-body text-ink-muted">{{ 'adminEscrow.empty' | t }}</p>
      } @else {
        <div class="mt-6 flex flex-col gap-2">
          @for (r of escrow.value()!; track r.id) {
            <div class="flex flex-wrap items-center gap-4 rounded-lg border border-line bg-surface-card p-4">
              <span class="text-mono-id text-ink-muted">RSV-{{ r.id.slice(-4).toUpperCase() }}</span>
              <span class="min-w-[100px] flex-1 text-body-strong">{{ r.totalAmount | fcfa }}</span>
              @if (r.payment; as payment) {
                <bt-status-badge kind="escrow" [status]="payment.status" />
              }
              @if (r.payment?.status === 'held') {
                <div class="flex gap-2">
                  <bt-button variant="outline" size="sm" [disabled]="acting() === r.id" (click)="pending.set({ kind: 'release', reservation: r })">
                    {{ 'adminEscrow.releaseButton' | t }}
                  </bt-button>
                  <bt-button variant="ghost" size="sm" [disabled]="acting() === r.id" (click)="pending.set({ kind: 'refund', reservation: r })">
                    {{ 'adminEscrow.refundButton' | t }}
                  </bt-button>
                </div>
              }
            </div>
          }
        </div>
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

  protected readonly escrow = apiResource<Reservation[]>(() => this.adminApi.escrowUrl());
  protected readonly acting = signal<string | null>(null);
  /** Both actions move money and can't be undone: asked for once. */
  protected readonly pending = signal<{ kind: 'release' | 'refund'; reservation: Reservation } | null>(null);

  protected confirmPending(): void {
    const p = this.pending();
    this.pending.set(null);
    if (!p) return;
    if (p.kind === 'refund') this.refund(p.reservation.id);
    else this.release(p.reservation.id);
  }

  private release(id: string): void {
    this.acting.set(id);
    this.adminApi.releaseEscrow(id).subscribe({
      next: () => {
        this.acting.set(null);
        this.escrow.reload();
      },
      error: (err: AppError) => {
        this.acting.set(null);
        this.toast.show({ tone: 'danger', title: 'adminEscrow.actionFailedTitle', detail: err.message });
      }
    });
  }

  private refund(id: string): void {
    this.acting.set(id);
    this.adminApi.refundEscrow(id).subscribe({
      next: () => {
        this.acting.set(null);
        this.escrow.reload();
      },
      error: (err: AppError) => {
        this.acting.set(null);
        this.toast.show({ tone: 'danger', title: 'adminEscrow.actionFailedTitle', detail: err.message });
      }
    });
  }
}
