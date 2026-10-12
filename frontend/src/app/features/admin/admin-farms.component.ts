import { ChangeDetectionStrategy, Component, effect, inject, signal, untracked } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';
import { StaffSpaceService } from '../../core/auth/staff-space.service';
import { FarmDecision, FarmQueue, RejectionCode } from '../../domain/enums';
import { AdminFarm, AdminFarmDetail, AdminFarmList } from '../../domain/models';
import { speciesLabelKey } from '../../domain/reference/species';
import { AdminApi } from '../../data-access/admin/admin.api';
import { AppError } from '../../core/http/app-error';
import { apiResource } from '../../core/http/api-resource';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ToastService } from '../../core/notifications/toast.service';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
import { AvatarComponent } from '../../shared/ui/avatar/avatar.component';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { elapsed, isLate } from './ui/elapsed';

const QUEUES: readonly FarmQueue[] = ['todo', 'resubmitted', 'done'];
/** The team's review target, shown above the queue. */
const TARGET_HOURS = 48;
const CHECKS = ['identity', 'document', 'photos', 'location'] as const;
type Check = (typeof CHECKS)[number];

/** `/admin/farms` — farm validation, per `AdminFarms.dc.html`: the
 * queue on the left in three tabs (to do, sent again after a refusal,
 * done), oldest request first; on the right the farm picked, the
 * account behind it, the identity document and the farm photos, a
 * checklist for the reviewer, and the decision.
 *
 * A refusal takes a reason from the API's list plus a message the owner
 * reads; suspending an approved farm takes a message. The API records
 * every decision, and every opening of this page, in the journal. A
 * role that may read farms but not decide sees the same page without
 * the buttons. The checklist is the reviewer's own aid: nothing in it
 * is sent.
 *
 * On phones the queue and the detail are two screens (`/admin/farms`
 * and `/admin/farms/:id`). The proof is served through short-lived
 * links the API signs for staff only. */
@Component({
  selector: 'bt-admin-farms',
  imports: [FormsModule, RouterLink, AvatarComponent, BadgeComponent, ButtonComponent, IconComponent, StatusBadgeComponent, LocalDatePipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="grid min-h-dvh lg:grid-cols-[380px_minmax(0,1fr)]">
      <section class="flex-col gap-3.5 border-line bg-surface-raised px-4 py-6 sm:px-5 sm:py-7 lg:flex lg:border-r" [class]="id() ? 'hidden' : 'flex'" [attr.aria-label]="'adminFarms.queueLabel' | t">
        <div class="flex flex-col gap-0.5 px-1.5">
          <span class="text-eyebrow text-ink-muted">{{ 'adminFarms.eyebrow' | t }}</span>
          <h1 class="font-display text-[28px] leading-tight font-extrabold tracking-[-0.02em]">{{ 'adminFarms.title' | t }}</h1>
        </div>
        <div class="flex flex-wrap gap-1.5 px-1.5" role="group" [attr.aria-label]="'adminFarms.filterLabel' | t">
          @for (q of queues; track q) {
            <button
              type="button"
              [attr.aria-pressed]="queue() === q"
              (click)="queue.set(q)"
              class="inline-flex h-9 items-center rounded-full border px-3 text-caption font-semibold whitespace-nowrap transition"
              [class]="queue() === q ? 'border-ink bg-ink text-surface' : 'border-line bg-surface-card text-ink hover:border-line-strong'"
            >
              {{ 'adminFarms.queue.' + q | t }}
              @if (count(q); as n) {
                · {{ n }}
              }
            </button>
          }
        </div>
        <span class="px-1.5 text-[12.5px] text-ink-muted">{{ (queue() === 'done' ? 'adminFarms.orderDone' : 'adminFarms.orderTodo') | t: { hours: targetHours } }}</span>

        @if (list.isLoading() && !list.hasValue()) {
          @for (i of [1, 2, 3]; track i) {
            <div class="h-20 animate-pulse rounded-[14px] bg-surface-sunken"></div>
          }
        } @else if (list.error()) {
          <div class="flex flex-col items-start gap-3 px-1.5" role="alert">
            <p class="text-body">{{ 'apiErrors.server' | t }}</p>
            <bt-button variant="outline" (click)="list.reload()">{{ 'common.retry' | t }}</bt-button>
          </div>
        } @else {
          <ul class="flex flex-col gap-2">
            @for (farm of list.value()?.farms ?? []; track farm.id) {
              <li>
                <a
                  [routerLink]="['/admin/farms', farm.id]"
                  [attr.aria-current]="farm.id === id() ? 'true' : null"
                  class="flex gap-3 rounded-[14px] border-[1.5px] p-3 text-ink transition"
                  [class]="farm.id === id() ? 'border-forest bg-surface-card' : 'border-line hover:border-line-strong'"
                >
                  <bt-avatar [name]="farm.name" [size]="56" shape="square" tone="forest" class="shrink-0 !rounded-[10px]" />
                  <span class="flex min-w-0 flex-1 flex-col gap-1">
                    <span class="flex justify-between gap-2">
                      <b class="wrap-anywhere text-[14.5px]">{{ farm.name }}</b>
                      @if (farm.status === 'pending' && farm.review?.submittedAt; as at) {
                        <span class="font-mono text-[11px] whitespace-nowrap" [class]="late(at) ? 'font-semibold text-clay-ink' : 'text-ink-muted'">{{ since(at) }}</span>
                      }
                    </span>
                    <span class="wrap-anywhere text-[12.5px] text-ink-muted">{{ farm.owner?.displayName || farm.owner?.email }} · {{ farm.location }}</span>
                    <span class="flex flex-wrap gap-1.5">
                      @if (farm.status !== 'pending') {
                        <bt-status-badge kind="farm" [status]="farm.status" />
                      } @else if ((farm.review?.submissions ?? 0) > 1) {
                        <bt-badge tone="sky">{{ 'adminFarms.tagResubmitted' | t }}</bt-badge>
                      } @else if ((farm.owner?.farmCount ?? 1) > 1) {
                        <bt-badge tone="clay">{{ 'adminFarms.tagNth' | t: { n: farm.owner?.farmCount ?? 1 } }}</bt-badge>
                      } @else {
                        <bt-badge tone="neutral">{{ 'adminFarms.tagFirst' | t }}</bt-badge>
                      }
                    </span>
                  </span>
                </a>
              </li>
            } @empty {
              <li class="rounded-[14px] border border-dashed border-line-strong px-4 py-10 text-center text-body text-ink-muted">{{ 'adminFarms.empty.' + queue() | t }}</li>
            }
          </ul>
        }
      </section>

      <section class="min-w-0 flex-col gap-5 p-4 sm:p-7 lg:flex lg:px-9" [class]="id() ? 'flex' : 'hidden'">
        @if (!id()) {
          <p class="m-auto max-w-xs text-center text-body text-ink-muted">{{ 'adminFarms.pick' | t }}</p>
        } @else if (detail.value(); as d) {
          <a routerLink="/admin/farms" class="-my-2 inline-flex min-h-11 items-center self-start text-small text-ink-muted hover:text-forest lg:hidden">← {{ 'adminFarms.backToQueue' | t }}</a>

          <header class="flex flex-wrap items-start justify-between gap-5">
            <div class="flex min-w-0 flex-col gap-2">
              <div class="flex flex-wrap items-center gap-2">
                <bt-status-badge kind="farm" [status]="d.status" [suffix]="d.status === 'pending' && d.review?.submittedAt ? since(d.review!.submittedAt!) : undefined" />
                @if (d.reference) {
                  <span class="inline-flex h-6 items-center rounded-full border border-line bg-surface-card px-2.5 font-mono text-caption font-medium">{{ d.reference }}</span>
                }
                @if (d.review?.autoApproved) {
                  <bt-badge tone="amber">{{ 'adminFarms.autoApproved' | t }}</bt-badge>
                }
                @if ((d.review?.submissions ?? 0) > 1) {
                  <bt-badge tone="sky">{{ 'adminFarms.tagResubmitted' | t }}</bt-badge>
                }
              </div>
              <h2 class="wrap-anywhere font-display text-[26px] leading-tight font-extrabold tracking-[-0.02em] sm:text-[32px]">{{ d.name }}</h2>
              <p class="wrap-anywhere text-small text-ink-muted">
                {{ d.location }}{{ d.region ? ', ' + d.region : '' }}
                @for (s of d.species ?? []; track s) {
                  · {{ speciesKey(s) | t }}
                }
                @if (d.review?.submittedAt; as at) {
                  · {{ 'adminFarms.requestedOn' | t: { date: (at | localDate: 'dayMonthTime') } }}
                }
              </p>
              @if (d.contactPhone) {
                <p class="font-mono text-[13px] text-ink-muted">{{ d.contactPhone }}</p>
              }
            </div>

            <div class="bo-card flex w-full flex-col gap-1.5 px-4 py-3.5 sm:w-auto sm:min-w-[280px]">
              <span class="text-eyebrow text-ink-muted">{{ 'adminFarms.ownerTitle' | t }}</span>
              <div class="flex items-center gap-2.5">
                <bt-avatar [name]="d.owner?.displayName || d.owner?.email || '?'" [size]="36" tone="clay" class="shrink-0" />
                <div class="flex min-w-0 flex-col">
                  <b class="wrap-anywhere text-small">{{ d.owner?.displayName || d.owner?.email }}</b>
                  <span class="text-caption text-ink-muted">
                    {{ 'adminFarms.ownerSince' | t: { year: (d.owner?.memberSince | localDate: 'year') } }}
                    · {{ 'adminFarms.ownerPurchases' | t: { count: d.ownerPurchases } }}
                    · {{ 'adminFarms.ownerReports' | t: { count: d.ownerReports } }}
                  </span>
                </div>
              </div>
              @if (d.owner?.displayName) {
                <span class="wrap-anywhere text-caption text-ink-muted">{{ d.owner?.email }}</span>
              }
              @if (d.owner?.suspended) {
                <bt-badge tone="alert" class="self-start">{{ 'adminUsers.suspendedBadge' | t }}</bt-badge>
              }
              <div class="flex items-center justify-between gap-3 border-t border-line pt-2 text-[13px]">
                <span>
                  {{ 'adminFarms.ownerFarms' | t }} <b>{{ d.ownerFarmCount }} / {{ d.ownerFarmLimit }}</b>
                  @if (d.owner?.farmLimitNote; as note) {
                    <span class="text-ink-muted">({{ 'adminFarms.limitRaisedOn' | t: { date: (note.setAt | localDate: 'dayMonth') } }})</span>
                  }
                </span>
                @if (staff.can('accounts.farm_limit')) {
                  <a routerLink="/admin/staff" fragment="farm-limits" class="font-bold text-forest underline-offset-2 hover:underline">{{ 'adminFarms.manageLimit' | t }}</a>
                }
              </div>
            </div>
          </header>

          <div class="grid gap-[18px] xl:grid-cols-2">
            <div class="bo-card flex flex-col gap-3 p-[18px]">
              <div class="flex items-center justify-between gap-3">
                <b class="font-display text-[17px]">{{ 'adminFarms.idDocument' | t }}</b>
                <span class="flex items-center gap-1.5 text-caption text-ink-muted"><bt-icon name="lock" [size]="14" />{{ 'adminFarms.accessLogged' | t }}</span>
              </div>
              @if (!d.idDocumentUrl) {
                <p class="text-small text-ink-muted">{{ 'adminFarms.noProof' | t }}</p>
              } @else {
                <div class="grid grid-cols-2 gap-2.5">
                  <a [href]="d.idDocumentUrl" target="_blank" rel="noopener" class="group relative block h-[130px] overflow-hidden rounded-xl border border-line bg-surface-sunken">
                    <img [src]="d.idDocumentUrl" [alt]="'adminFarms.idFrontAlt' | t: { farm: d.name }" class="h-full w-full object-cover" />
                    <span class="absolute inset-x-0 bottom-0 bg-ink/75 px-2.5 py-1.5 text-caption font-semibold text-surface">{{ 'adminFarms.idFront' | t: { type: docType(d) } }}</span>
                  </a>
                  @if (d.idDocumentBackUrl; as back) {
                    <a [href]="back" target="_blank" rel="noopener" class="group relative block h-[130px] overflow-hidden rounded-xl border border-line bg-surface-sunken">
                      <img [src]="back" [alt]="'adminFarms.idBackAlt' | t: { farm: d.name }" class="h-full w-full object-cover" />
                      <span class="absolute inset-x-0 bottom-0 bg-ink/75 px-2.5 py-1.5 text-caption font-semibold text-surface">{{ 'adminFarms.idBack' | t: { type: docType(d) } }}</span>
                    </a>
                  } @else {
                    <div class="flex h-[130px] items-center justify-center rounded-xl border border-dashed border-line-strong px-3 text-center text-caption text-ink-muted">{{ 'adminFarms.idNoBack' | t }}</div>
                  }
                </div>
                <span class="text-[12.5px] text-ink-muted">
                  {{ 'adminFarms.accountName' | t }} <b class="text-ink">{{ d.owner?.displayName || ('adminFarms.accountNoName' | t) }}</b> · {{ 'adminFarms.clickToEnlarge' | t }}
                </span>
              }
            </div>

            <div class="bo-card flex flex-col gap-3 p-[18px]">
              <b class="font-display text-[17px]">{{ 'adminFarms.farmPhotos' | t: { count: d.photoUrls.length } }}</b>
              @if (d.photoUrls.length === 0) {
                <p class="text-small text-ink-muted">{{ 'adminFarms.noPhotos' | t }}</p>
              } @else {
                <div class="grid grid-cols-3 gap-2">
                  @for (url of d.photoUrls; track url; let i = $index) {
                    <a [href]="url" target="_blank" rel="noopener" class="block h-[130px] overflow-hidden rounded-xl border border-line bg-surface-sunken">
                      <img [src]="url" [alt]="'adminFarms.farmPhotoAlt' | t: { n: i + 1, farm: d.name }" class="h-full w-full object-cover" />
                    </a>
                  }
                </div>
              }
              @if (d.description) {
                <p class="wrap-anywhere text-[12.5px] text-ink-muted">{{ d.description }}</p>
              }
            </div>
          </div>

          @if (d.status === 'pending' && staff.can('farms.review')) {
            <fieldset class="bo-card flex flex-col gap-1 px-5 py-[18px]">
              <legend class="float-left mb-1.5 font-display text-[17px] font-bold">{{ 'adminFarms.checksTitle' | t }}</legend>
              <div class="clear-both flex flex-col">
                @for (c of checks; track c; let last = $last) {
                  <label class="flex min-h-11 cursor-pointer items-center gap-3 text-small" [class.border-b]="!last" [class.border-line]="!last">
                    <input type="checkbox" class="h-[18px] w-[18px] shrink-0 accent-forest" [checked]="checked().has(c)" (change)="toggle(c)" />
                    <span class="flex-1">{{ 'adminFarms.check.' + c | t }}</span>
                  </label>
                }
              </div>
              <p class="mt-1 text-caption text-ink-muted">{{ 'adminFarms.checksHint' | t }}</p>
            </fieldset>
          }

          @if (d.status !== 'pending' && d.status !== 'draft') {
            <div class="flex flex-col gap-1 rounded-2xl p-[18px]" [class]="d.status === 'approved' ? 'bg-forest-soft text-on-forest-soft' : 'bg-alert-soft text-alert'">
              <b class="text-body-strong">
                @if (d.review?.autoApproved) {
                  {{ 'adminFarms.decidedAuto' | t }}
                } @else {
                  {{ 'adminFarms.decided.' + d.status | t: { name: d.reviewerName || '—', date: (d.review?.reviewedAt | localDate: 'dayMonthYear') } }}
                }
              </b>
              @if (d.review?.rejectionCode; as code) {
                <span class="text-small">{{ 'rejectionCode.' + code | t }}</span>
              }
              @if (d.review?.rejectionReason; as reason) {
                <p class="wrap-anywhere text-small">« {{ reason }} »</p>
              }
            </div>
          }

          @if (asking(); as mode) {
            <div class="flex flex-col gap-3 rounded-2xl border-[1.5px] border-alert/30 bg-alert-soft px-5 py-[18px]">
              <b class="font-display text-[17px] text-alert">{{ (mode === 'suspend' ? 'adminFarms.suspendTitle' : 'adminFarms.rejectTitle') | t }}</b>
              <div class="grid gap-3.5" [class.md:grid-cols-[260px_minmax(0,1fr)]]="mode === 'reject'">
                @if (mode === 'reject') {
                  <div class="flex flex-col gap-1.5">
                    <label for="farm-reject-code" class="text-[13px] font-semibold">{{ 'adminFarms.reasonCodeLabel' | t }}</label>
                    <select
                      id="farm-reject-code"
                      class="h-11 rounded-[10px] border border-line-strong bg-surface-card px-3 text-small font-medium text-ink"
                      [ngModel]="reasonCode()"
                      (ngModelChange)="reasonCode.set($event)"
                    >
                      @for (code of d.rejectionCodes; track code) {
                        <option [value]="code">{{ 'rejectionCode.' + code | t }}</option>
                      }
                    </select>
                  </div>
                }
                <div class="flex flex-col gap-1.5">
                  <label for="farm-reject-message" class="text-[13px] font-semibold">{{ 'adminFarms.messageLabel' | t }}</label>
                  <textarea
                    id="farm-reject-message"
                    rows="2"
                    maxlength="500"
                    class="rounded-[10px] border border-line-strong bg-surface-card px-3 py-2.5 text-base font-medium text-ink sm:text-small"
                    [attr.aria-invalid]="reasonMissing() ? 'true' : null"
                    [attr.aria-describedby]="reasonMissing() ? 'farm-reject-error' : 'farm-reject-hint'"
                    [ngModel]="reason()"
                    (ngModelChange)="reason.set($event); reasonMissing.set(false)"
                  ></textarea>
                  @if (reasonMissing()) {
                    <span id="farm-reject-error" role="alert" class="text-caption font-semibold text-alert">{{ 'apiErrors.farm.reason_required' | t }}</span>
                  }
                </div>
              </div>
              <span id="farm-reject-hint" class="text-[12.5px] text-ink-muted">{{ (mode === 'suspend' ? 'adminFarms.suspendBody' : 'adminFarms.rejectBody') | t }}</span>
            </div>
          }

          <footer class="mt-auto flex flex-wrap items-center justify-between gap-3 border-t border-line pt-4">
            @if (!staff.can('farms.review')) {
              <span class="text-[12.5px] text-ink-muted">{{ 'adminFarms.readOnly' | t }}</span>
            } @else {
              <span class="text-[12.5px] text-ink-muted">{{ 'adminFarms.journaled' | t: { name: staff.name(), role: ('staffRole.' + staff.role() | t) } }}</span>
              <div class="flex flex-wrap gap-2.5">
                @if (asking(); as mode) {
                  <bt-button variant="outline" [disabled]="busy()" (click)="asking.set(null)">{{ (mode === 'suspend' ? 'adminFarms.cancelSuspend' : 'adminFarms.cancelReject') | t }}</bt-button>
                  <bt-button variant="danger" [disabled]="busy()" (click)="confirmAsked()">{{ (mode === 'suspend' ? 'adminFarms.sendSuspend' : 'adminFarms.sendReject') | t }}</bt-button>
                } @else {
                  @if (can(d, 'reject')) {
                    <bt-button variant="outline" [disabled]="busy()" (click)="ask('reject')">{{ 'adminFarms.reject' | t }}</bt-button>
                  }
                  @if (can(d, 'suspend')) {
                    <bt-button variant="outline" [disabled]="busy()" (click)="ask('suspend')">{{ 'adminFarms.suspend' | t }}</bt-button>
                  }
                  @if (can(d, 'approve')) {
                    <bt-button variant="solid" [disabled]="busy()" (click)="decide('approve')">
                      {{ (d.status === 'suspended' ? 'adminFarms.reinstate' : d.status === 'approved' ? 'adminFarms.confirm' : 'adminFarms.approve') | t }}
                    </bt-button>
                  }
                }
              </div>
            }
          </footer>
        } @else if (detail.isLoading()) {
          <div class="h-10 w-2/3 animate-pulse rounded-md bg-surface-sunken"></div>
          <div class="h-72 animate-pulse rounded-2xl bg-surface-sunken"></div>
        } @else if (detail.error()) {
          <div class="flex flex-col items-start gap-3" role="alert">
            <p class="text-body">{{ 'adminFarms.loadError' | t }}</p>
            <bt-button variant="outline" (click)="detail.reload()">{{ 'common.retry' | t }}</bt-button>
          </div>
        }
      </section>
    </div>
  `
})
export class AdminFarmsComponent {
  private readonly adminApi = inject(AdminApi);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);
  protected readonly staff = inject(StaffSpaceService);

  protected readonly queues = QUEUES;
  protected readonly checks = CHECKS;
  protected readonly targetHours = TARGET_HOURS;
  protected readonly queue = signal<FarmQueue>('todo');
  protected readonly id = toSignal(this.route.paramMap.pipe(map((p) => p.get('id') ?? '')), { initialValue: '' });

  protected readonly list = apiResource<AdminFarmList>(() => this.adminApi.farmsUrl(this.queue()));
  protected readonly detail = apiResource<AdminFarmDetail>(() => (this.id() ? this.adminApi.farmDetailUrl(this.id()) : undefined));

  protected readonly busy = signal(false);
  /** The decision being written up, in the panel above the buttons. */
  protected readonly asking = signal<'reject' | 'suspend' | null>(null);
  protected readonly reasonCode = signal<RejectionCode>('id_unreadable');
  protected readonly reason = signal('');
  protected readonly reasonMissing = signal(false);
  /** The reviewer's own checklist for the farm on screen. */
  protected readonly checked = signal<ReadonlySet<Check>>(new Set());

  constructor() {
    // Another farm on screen: a clean slate for the reviewer.
    effect(() => {
      this.id();
      untracked(() => {
        this.asking.set(null);
        this.checked.set(new Set());
      });
    });
  }

  protected count(queue: FarmQueue): number {
    const counts = this.list.value()?.counts;
    return queue === 'todo' ? (counts?.todo ?? 0) : queue === 'resubmitted' ? (counts?.resubmitted ?? 0) : 0;
  }

  protected since(iso: string): string {
    return elapsed(iso, this.locale);
  }

  protected late(iso: string): boolean {
    return isLate(iso, TARGET_HOURS);
  }

  protected speciesKey(value: string): string {
    return speciesLabelKey(value);
  }

  protected readonly docType = (d: AdminFarmDetail): string => this.locale.t(`idDocumentType.${d.review?.idDocumentType ?? 'national_id'}`);

  protected toggle(check: Check): void {
    const next = new Set(this.checked());
    if (!next.delete(check)) next.add(check);
    this.checked.set(next);
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
    this.reasonCode.set(this.detail.value()?.rejectionCodes[0] ?? 'other');
    this.asking.set(decision);
  }

  protected confirmAsked(): void {
    const decision = this.asking();
    if (!decision) return;
    if (!this.reason().trim()) {
      this.reasonMissing.set(true);
      return;
    }
    this.decide(decision, this.reason().trim(), decision === 'reject' ? this.reasonCode() : undefined);
  }

  protected decide(decision: FarmDecision, reason?: string, code?: RejectionCode): void {
    if (this.busy() || !this.id()) return;
    this.busy.set(true);
    this.adminApi.reviewFarm(this.id(), decision, reason, code).subscribe({
      next: () => {
        this.busy.set(false);
        this.asking.set(null);
        this.toast.show({ tone: 'success', title: `adminFarms.done.${decision}` });
        this.list.reload();
        this.detail.reload();
        // The farm left the queue: back to it for the next one.
        if (decision !== 'suspend') this.router.navigate(['/admin/farms']);
      },
      error: (err: AppError) => {
        this.busy.set(false);
        this.toast.show({ tone: 'danger', title: 'adminFarms.failed', detail: err.message });
      }
    });
  }
}
