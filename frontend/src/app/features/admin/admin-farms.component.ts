import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';
import { FarmDecision, FarmStatus } from '../../domain/enums';
import { AdminFarm, AdminFarmDetail } from '../../domain/models';
import { countryName } from '../../domain/reference/countries';
import { AdminApi } from '../../data-access/admin/admin.api';
import { AppError } from '../../core/http/app-error';
import { apiResource } from '../../core/http/api-resource';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ToastService } from '../../core/notifications/toast.service';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';

const FILTERS: readonly FarmStatus[] = ['pending', 'approved', 'rejected', 'suspended', 'draft'];

/** `/admin/farms` — farm validation, in the design's review layout: the
 * queue on the left (pending first, filterable by status), the farm
 * picked on the right with what its owner submitted, and the decision
 * at the bottom: the safe action on the left, the one that commits on
 * the right. Refusing or suspending needs a reason; the owner reads it.
 *
 * On phones the queue and the detail are two screens (`/admin/farms`
 * and `/admin/farms/:id`). The proof is served through short-lived
 * links the API signs for staff only. */
@Component({
  selector: 'bt-admin-farms',
  imports: [
    FormsModule,
    RouterLink,
    BadgeComponent,
    ButtonComponent,
    DialogComponent,
    StatusBadgeComponent,
    TextFieldComponent,
    LocalDatePipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="grid min-h-full lg:grid-cols-[400px_minmax(0,1fr)]">
      <section class="flex-col gap-4 border-line bg-surface-raised p-5 lg:flex lg:border-r" [class]="id() ? 'hidden' : 'flex'" [attr.aria-label]="'adminFarms.queueLabel' | t">
        <div class="flex flex-col gap-1 px-1">
          <span class="text-eyebrow text-ink-muted">{{ 'adminFarms.eyebrow' | t }}</span>
          <h1 class="text-heading-2">{{ 'adminFarms.title' | t }}</h1>
        </div>
        <div class="flex flex-wrap gap-1.5 px-1" role="group" [attr.aria-label]="'adminFarms.filterLabel' | t">
          @for (f of filters; track f) {
            <button
              type="button"
              [attr.aria-pressed]="status() === f"
              (click)="status.set(f)"
              class="h-9 rounded-full border px-3 text-small font-semibold transition"
              [class]="status() === f ? 'border-ink bg-ink text-surface' : 'border-line bg-surface-card text-ink hover:border-line-strong'"
            >
              {{ 'status.farm' + f[0].toUpperCase() + f.slice(1) | t }}
            </button>
          }
        </div>

        @if (queue.isLoading() && !queue.hasValue()) {
          @for (i of [1, 2, 3]; track i) {
            <div class="h-24 animate-pulse rounded-lg bg-surface-sunken"></div>
          }
        } @else if (queue.error()) {
          <div class="flex flex-col items-start gap-3 px-1" role="alert">
            <p class="text-body">{{ 'apiErrors.server' | t }}</p>
            <bt-button variant="outline" (click)="queue.reload()">{{ 'common.retry' | t }}</bt-button>
          </div>
        } @else {
          <ul class="flex flex-col gap-2">
            @for (farm of queue.value() ?? []; track farm.id) {
              <li>
                <a
                  [routerLink]="['/admin/farms', farm.id]"
                  [attr.aria-current]="farm.id === id() ? 'true' : null"
                  class="flex flex-col gap-2 rounded-lg border-2 p-3.5 text-ink transition"
                  [class]="farm.id === id() ? 'border-forest bg-surface-card' : 'border-line hover:border-line-strong'"
                >
                  <span class="flex items-center justify-between gap-2">
                    <bt-status-badge kind="farm" [status]="farm.status" />
                    <span class="font-mono text-caption font-medium text-ink-muted">{{ farm.review?.submittedAt | localDate: 'dayMonth' }}</span>
                  </span>
                  <span class="wrap-anywhere text-body-strong">{{ farm.name }}</span>
                  <span class="wrap-anywhere text-small text-ink-muted">{{ farm.owner?.displayName || farm.owner?.email }} · {{ farm.location }}</span>
                </a>
              </li>
            } @empty {
              <li class="rounded-lg border border-dashed border-line-strong px-4 py-10 text-center text-body text-ink-muted">{{ 'adminFarms.empty' | t }}</li>
            }
          </ul>
        }
      </section>

      <section class="min-w-0 flex-col gap-5 p-5 sm:p-8 lg:flex" [class]="id() ? 'flex' : 'hidden'">
        @if (!id()) {
          <p class="m-auto max-w-xs text-center text-body text-ink-muted">{{ 'adminFarms.pick' | t }}</p>
        } @else if (detail.value(); as d) {
          <a routerLink="/admin/farms" class="-my-2 inline-flex min-h-11 items-center self-start text-small text-ink-muted hover:text-forest lg:hidden">← {{ 'adminFarms.backToQueue' | t }}</a>
          <header class="flex flex-col gap-2">
            <div class="flex flex-wrap gap-2">
              <bt-status-badge kind="farm" [status]="d.status" />
              @if (d.review?.autoApproved) {
                <bt-badge tone="amber">{{ 'adminFarms.autoApproved' | t }}</bt-badge>
              }
              @if (d.owner?.suspended) {
                <bt-badge tone="alert">{{ 'adminUsers.suspendedBadge' | t }}</bt-badge>
              }
            </div>
            <h2 class="wrap-anywhere text-heading-2">{{ d.name }}</h2>
            <p class="wrap-anywhere text-body text-ink-muted">
              {{ 'adminFarms.ownedBy' | t }}
              <strong class="font-semibold text-ink">{{ d.owner?.displayName || d.owner?.email }}</strong>
              @if (d.owner?.displayName) {
                · {{ d.owner?.email }}
              }
            </p>
          </header>

          <div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_280px]">
            <div class="flex flex-col gap-4 rounded-lg border border-line bg-surface-card p-5">
              <h3 class="text-heading-4">{{ 'adminFarms.proofTitle' | t }}</h3>
              @if (!d.idDocumentUrl && d.photoUrls.length === 0) {
                <p class="text-body text-ink-muted">{{ 'adminFarms.noProof' | t }}</p>
              } @else {
                <div class="grid gap-3 sm:grid-cols-2">
                  @if (d.idDocumentUrl; as url) {
                    <figure class="flex flex-col gap-2 sm:col-span-2">
                      <figcaption class="text-eyebrow text-ink-muted">{{ 'adminFarms.idDocument' | t }}</figcaption>
                      <a [href]="url" target="_blank" rel="noopener" class="block overflow-hidden rounded-md border border-line bg-surface-sunken">
                        <img [src]="url" [alt]="'adminFarms.idDocumentAlt' | t: { farm: d.name }" class="max-h-80 w-full object-contain" />
                      </a>
                    </figure>
                  }
                  @for (url of d.photoUrls; track url; let i = $index) {
                    <figure class="flex flex-col gap-2">
                      <figcaption class="text-eyebrow text-ink-muted">{{ 'adminFarms.farmPhoto' | t: { n: i + 1 } }}</figcaption>
                      <a [href]="url" target="_blank" rel="noopener" class="block overflow-hidden rounded-md border border-line bg-surface-sunken">
                        <img [src]="url" [alt]="'adminFarms.farmPhotoAlt' | t: { n: i + 1, farm: d.name }" class="aspect-[4/3] w-full object-cover" />
                      </a>
                    </figure>
                  }
                </div>
              }
            </div>

            <dl class="flex flex-col gap-3 self-start rounded-lg border border-line bg-surface-card p-5 text-body">
              <div class="flex flex-col">
                <dt class="text-eyebrow text-ink-muted">{{ 'adminFarms.place' | t }}</dt>
                <dd class="wrap-anywhere">{{ d.location }}{{ d.region ? ' · ' + d.region : '' }} · {{ country(d.country) }}</dd>
              </div>
              @if (d.contactPhone) {
                <div class="flex flex-col">
                  <dt class="text-eyebrow text-ink-muted">{{ 'adminFarms.phone' | t }}</dt>
                  <dd class="font-mono">{{ d.contactPhone }}</dd>
                </div>
              }
              <div class="flex flex-col">
                <dt class="text-eyebrow text-ink-muted">{{ 'adminFarms.submitted' | t }}</dt>
                <dd>{{ d.review?.submittedAt ? (d.review?.submittedAt | localDate: 'dayMonthYear') : ('adminFarms.notSubmitted' | t) }}</dd>
              </div>
              <div class="flex flex-col">
                <dt class="text-eyebrow text-ink-muted">{{ 'adminFarms.ownerFarms' | t }}</dt>
                <dd>{{ d.ownerFarmCount }}</dd>
              </div>
              <div class="flex flex-col">
                <dt class="text-eyebrow text-ink-muted">{{ 'adminFarms.ownerTrust' | t }}</dt>
                <dd><bt-status-badge kind="trust" [status]="d.owner?.trustScore ?? 0" /></dd>
              </div>
            </dl>
          </div>

          @if (d.description) {
            <div class="rounded-lg border border-line bg-surface-card p-5">
              <h3 class="text-heading-4">{{ 'adminFarms.description' | t }}</h3>
              <p class="mt-2 wrap-anywhere text-body text-ink-muted">{{ d.description }}</p>
            </div>
          }
          @if (d.review?.rejectionReason; as reason) {
            <div class="rounded-lg bg-alert-soft p-5 text-alert">
              <h3 class="text-body-strong">{{ 'adminFarms.lastReason' | t }}</h3>
              <p class="mt-1 wrap-anywhere text-body">{{ reason }}</p>
            </div>
          }

          <footer class="mt-auto flex flex-wrap items-center justify-between gap-3 border-t border-line pt-4">
            @if (can(d, 'reject')) {
              <bt-button variant="outline" [disabled]="busy()" (click)="ask('reject')">{{ 'adminFarms.reject' | t }}</bt-button>
            } @else if (can(d, 'suspend')) {
              <bt-button variant="danger" [disabled]="busy()" (click)="ask('suspend')">{{ 'adminFarms.suspend' | t }}</bt-button>
            } @else {
              <span></span>
            }
            @if (can(d, 'approve')) {
              <bt-button variant="solid" [disabled]="busy()" (click)="decide('approve')">
                {{ (d.status === 'suspended' ? 'adminFarms.reinstate' : d.status === 'approved' ? 'adminFarms.confirm' : 'adminFarms.approve') | t }}
              </bt-button>
            }
          </footer>
        } @else if (detail.isLoading()) {
          <div class="h-10 w-2/3 animate-pulse rounded-md bg-surface-sunken"></div>
          <div class="h-72 animate-pulse rounded-lg bg-surface-sunken"></div>
        } @else if (detail.error()) {
          <div class="flex flex-col items-start gap-3" role="alert">
            <p class="text-body">{{ 'adminFarms.loadError' | t }}</p>
            <bt-button variant="outline" (click)="detail.reload()">{{ 'common.retry' | t }}</bt-button>
          </div>
        }
      </section>
    </div>

    <bt-dialog
      [open]="!!asking()"
      tone="alert"
      [title]="(asking() === 'suspend' ? 'adminFarms.suspendTitle' : 'adminFarms.rejectTitle') | t"
      [primaryLabel]="(asking() === 'suspend' ? 'adminFarms.suspend' : 'adminFarms.reject') | t"
      primaryVariant="danger"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="confirmAsked()"
      (secondaryClick)="asking.set(null)"
      (closed)="asking.set(null)"
    >
      <div class="flex flex-col gap-3">
        <p>{{ (asking() === 'suspend' ? 'adminFarms.suspendBody' : 'adminFarms.rejectBody') | t }}</p>
        <bt-text-field
          [label]="'adminFarms.reasonLabel' | t"
          multiline
          markRequired
          [ngModel]="reason()"
          (ngModelChange)="reason.set($event)"
          [error]="reasonMissing() ? ('apiErrors.farm.reason_required' | t) : undefined"
        />
      </div>
    </bt-dialog>
  `
})
export class AdminFarmsComponent {
  private readonly adminApi = inject(AdminApi);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);

  protected readonly filters = FILTERS;
  protected readonly status = signal<FarmStatus>('pending');
  protected readonly id = toSignal(this.route.paramMap.pipe(map((p) => p.get('id') ?? '')), { initialValue: '' });

  protected readonly queue = apiResource<AdminFarm[]>(() => this.adminApi.farmsUrl(this.status()));
  protected readonly detail = apiResource<AdminFarmDetail>(() => (this.id() ? this.adminApi.farmDetailUrl(this.id()) : undefined));

  protected readonly busy = signal(false);
  /** The decision waiting on a reason, in the dialog. */
  protected readonly asking = signal<'reject' | 'suspend' | null>(null);
  protected readonly reason = signal('');
  protected readonly reasonMissing = signal(false);

  protected readonly lang = computed(() => this.locale.locale());

  protected country(code: string | undefined): string {
    return code ? countryName(code, this.lang()) : '';
  }

  /** Which decisions the buttons offer for a status. The API has the
   * last word (`farm.decision_not_applicable`); this only avoids
   * offering what it would refuse. */
  protected can(farm: AdminFarm, decision: FarmDecision): boolean {
    switch (decision) {
      case 'approve':
        return farm.status === 'pending' || farm.status === 'suspended' || (farm.status === 'approved' && !!farm.review?.autoApproved);
      case 'reject':
        return farm.status === 'pending';
      case 'suspend':
        return farm.status === 'approved';
    }
  }

  protected ask(decision: 'reject' | 'suspend'): void {
    this.reason.set('');
    this.reasonMissing.set(false);
    this.asking.set(decision);
  }

  protected confirmAsked(): void {
    const decision = this.asking();
    if (!decision) return;
    if (!this.reason().trim()) {
      this.reasonMissing.set(true);
      return;
    }
    this.asking.set(null);
    this.decide(decision, this.reason().trim());
  }

  protected decide(decision: FarmDecision, reason?: string): void {
    if (this.busy() || !this.id()) return;
    this.busy.set(true);
    this.adminApi.reviewFarm(this.id(), decision, reason).subscribe({
      next: () => {
        this.busy.set(false);
        this.toast.show({ tone: 'success', title: `adminFarms.done.${decision}` });
        this.queue.reload();
        this.detail.reload();
        // On phones the detail is its own screen: back to the queue.
        if (decision !== 'suspend') this.router.navigate(['/admin/farms']);
      },
      error: (err: AppError) => {
        this.busy.set(false);
        this.toast.show({ tone: 'danger', title: 'adminFarms.failed', detail: err.message });
      }
    });
  }
}
