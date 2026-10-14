import { ChangeDetectionStrategy, Component, computed, effect, inject, signal, untracked } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { toSignal } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { map } from 'rxjs';
import { ReportLevel, ResolveAction } from '../../domain/enums';
import { Report, ReportDetail, ReportRiskSignal } from '../../domain/models';
import { AdminApi } from '../../data-access/admin/admin.api';
import { StaffSpaceService } from '../../core/auth/staff-space.service';
import { apiResource } from '../../core/http/api-resource';
import { AppError } from '../../core/http/app-error';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ToastService } from '../../core/notifications/toast.service';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { IconName } from '../../shared/ui/icon/icon-paths';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { elapsed } from './ui/elapsed';

type QueueFilter = 'open' | 'review' | 'handled';
const FILTERS: readonly QueueFilter[] = ['open', 'review', 'handled'];

const LEVEL_CHIP: Readonly<Record<ReportLevel, string>> = {
  high: 'bg-alert text-on-alert',
  medium: 'bg-clay-soft text-clay-ink',
  low: 'bg-stone-soft text-stone'
};

/** A signal's weight in the risk score decides how loud its row is. */
const STRONG_SIGNAL = 30;

/** `/admin/reports` and `/admin/disputes` — per `Moderation.dc.html`: the
 * queue on the left, the selected report on the right with its combined
 * risk signals, a preview of what was reported, every report the target
 * drew, and the decision bar.
 *
 * The same queue read two ways: reports are about a listing, a profile
 * or a forum post; disputes are about a reservation (the route says
 * which, see `data.disputes`). Names, counts, the urgency level, the
 * score and its signals all come from the API. A role that may read
 * reports but not decide sees the page without the buttons.
 *
 * On phones the queue and the detail are two screens. */
@Component({
  selector: 'bt-admin-reports',
  imports: [FormsModule, RouterLink, ButtonComponent, DialogComponent, IconComponent, StatusBadgeComponent, FcfaCurrencyPipe, LocalDatePipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="grid min-h-dvh lg:grid-cols-[400px_minmax(0,1fr)]">
      <section class="flex-col gap-3.5 border-line bg-surface-raised px-4 py-6 sm:px-5 sm:py-7 lg:flex lg:border-r" [class]="id() ? 'hidden' : 'flex'" [attr.aria-label]="'adminReports.queueLabel' | t">
        <div class="flex flex-col gap-0.5 px-1.5">
          <span class="text-eyebrow text-ink-muted">{{ 'adminReports.eyebrow' | t }}</span>
          <h1 class="font-display text-[28px] leading-tight font-extrabold tracking-[-0.02em]">{{ (disputes() ? 'adminReports.disputesTitle' : 'adminReports.title') | t }}</h1>
        </div>
        <div class="flex flex-wrap gap-1.5 px-1.5" role="group" [attr.aria-label]="'adminReports.statusFilterLabel' | t">
          @for (f of filters; track f) {
            <button
              type="button"
              [attr.aria-pressed]="filter() === f"
              (click)="filter.set(f)"
              class="inline-flex h-9 items-center rounded-full border px-3 text-caption font-semibold whitespace-nowrap transition"
              [class]="filter() === f ? 'border-ink bg-ink text-surface' : 'border-line bg-surface-card text-ink hover:border-line-strong'"
            >
              {{ 'adminReports.queue.' + f | t }}
              @if (f !== 'handled' && count(f); as n) {
                · {{ n }}
              }
            </button>
          }
        </div>

        @if (reports.isLoading() && !reports.hasValue()) {
          @for (i of [1, 2, 3]; track i) {
            <div class="h-24 animate-pulse rounded-[14px] bg-surface-sunken"></div>
          }
        } @else if (reports.error()) {
          <div class="flex flex-col items-start gap-3 px-1.5" role="alert">
            <p class="text-body">{{ 'apiErrors.server' | t }}</p>
            <bt-button variant="outline" (click)="reports.reload()">{{ 'common.retry' | t }}</bt-button>
          </div>
        } @else {
          <ul class="flex flex-col gap-2">
            @for (r of shown(); track r.id) {
              <li>
                <a
                  [routerLink]="[base(), r.id]"
                  [attr.aria-current]="r.id === id() ? 'true' : null"
                  class="flex flex-col gap-2 rounded-[14px] border-[1.5px] p-3.5 text-ink transition"
                  [class]="r.id === id() ? 'border-alert bg-surface-card' : 'border-line hover:border-line-strong'"
                >
                  <span class="flex items-center justify-between gap-2">
                    <span class="inline-flex h-6 items-center rounded-full px-2.5 text-caption font-semibold" [class]="levelChip(r.level)">{{ 'adminReports.level.' + r.level | t }}</span>
                    @if (r.createdAt) {
                      <span class="font-mono text-[11px] text-ink-muted">{{ since(r.createdAt) }}</span>
                    }
                  </span>
                  <b class="wrap-anywhere text-[15px]">{{ r.targetLabel || ('reportTarget.' + r.targetType | t) }}</b>
                  <span class="flex justify-between gap-3 text-[12.5px] text-ink-muted">
                    <span class="line-clamp-1 min-w-0">{{ 'reportTarget.' + r.targetType | t }} · {{ r.reason }}</span>
                    <span class="font-mono">×{{ r.count }}</span>
                  </span>
                </a>
              </li>
            } @empty {
              <li class="rounded-[14px] border border-dashed border-line-strong px-4 py-10 text-center text-body text-ink-muted">
                {{ (disputes() ? 'adminReports.disputesEmpty' : 'adminReports.empty') | t }}
              </li>
            }
          </ul>
        }
      </section>

      <section class="min-w-0 flex-col gap-5 p-4 sm:p-7 lg:flex lg:px-9" [class]="id() ? 'flex' : 'hidden'">
        @if (!id()) {
          <p class="m-auto max-w-xs text-center text-body text-ink-muted">{{ 'adminReports.pick' | t }}</p>
        } @else if (detail.value(); as d) {
          <a [routerLink]="base()" class="-my-2 inline-flex min-h-11 items-center self-start text-small text-ink-muted hover:text-forest lg:hidden">← {{ 'adminReports.backToReports' | t }}</a>

          <header class="flex flex-wrap items-start justify-between gap-5">
            <div class="flex min-w-0 flex-col gap-2">
              <div class="flex flex-wrap gap-2">
                <span class="inline-flex h-6 items-center rounded-full px-2.5 text-caption font-semibold" [class]="levelChip(d.report.level)">{{ 'adminReports.risk.' + d.report.level | t }}</span>
                <span class="inline-flex h-6 items-center rounded-full bg-ink px-2.5 text-caption font-semibold text-surface">{{ 'reportStatus.' + d.report.status | t }}</span>
                <span class="inline-flex h-6 items-center rounded-full border border-line bg-surface-card px-2.5 text-caption font-semibold">{{ 'reportTarget.' + d.report.targetType | t }}</span>
                @if (d.report.reference) {
                  <span class="inline-flex h-6 items-center rounded-full border border-line bg-surface-card px-2.5 font-mono text-caption font-medium">{{ d.report.reference }}</span>
                }
              </div>
              <h2 class="wrap-anywhere font-display text-[26px] leading-tight font-extrabold tracking-[-0.02em] sm:text-[30px]">{{ d.report.targetLabel || ('reportTarget.' + d.report.targetType | t) }}</h2>
              @if (d.subject; as s) {
                <p class="wrap-anywhere text-small text-ink-muted">
                  {{ (d.listing ? 'adminReports.publishedBy' : 'adminReports.account') | t }} <b class="text-ink">{{ s.name }}</b>
                  · {{ 'adminReports.accountAge' | t: { age: since(s.createdAt) } }}
                  @if (s.city) {
                    · {{ s.city }}
                  }
                  @if (s.suspended) {
                    · <b class="text-alert">{{ 'adminReports.alreadySuspended' | t }}</b>
                  }
                </p>
              }
            </div>
            @if (d.listing; as l) {
              <bt-button variant="outline" [href]="'/marketplace/' + l.id">{{ 'adminReports.viewListing' | t }}</bt-button>
            }
          </header>

          <div class="grid gap-[18px]" [class.xl:grid-cols-[minmax(0,1fr)_300px]]="d.listing">
            <div class="bo-card flex flex-col gap-2.5 p-5">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <b class="font-display text-[17px]">{{ 'adminReports.signalsTitle' | t }}</b>
                <span class="font-mono text-caption" [class]="d.riskScore >= 60 ? 'text-alert' : 'text-ink-muted'">{{ 'adminReports.riskScore' | t: { score: d.riskScore } }}</span>
              </div>
              @for (s of d.signals; track s.name) {
                <div class="grid grid-cols-[28px_minmax(0,1fr)_auto] items-center gap-3 rounded-xl px-3.5 py-3" [class]="strong(s) ? 'bg-alert-soft' : 'bg-clay-soft'">
                  <bt-icon [name]="signalIcon(s)" [size]="20" [class]="strong(s) ? 'text-alert' : 'text-clay-ink'" />
                  <span class="wrap-anywhere text-small text-ink">{{ s.name }}</span>
                  <span class="font-mono text-[12.5px]" [class]="strong(s) ? 'text-alert' : 'text-clay-ink'">{{ s.detail }}</span>
                </div>
              } @empty {
                <div class="grid grid-cols-[28px_minmax(0,1fr)] items-center gap-3 rounded-xl bg-forest-soft px-3.5 py-3">
                  <bt-icon name="shield" [size]="20" class="text-forest" />
                  <span class="text-small text-on-forest-soft">{{ 'adminReports.noSignal' | t }}</span>
                </div>
              }
            </div>

            @if (d.listing; as l) {
              <div class="bo-card flex flex-col overflow-hidden">
                @if (l.photoUrl) {
                  <img [src]="l.photoUrl" alt="" class="h-[140px] w-full object-cover" />
                } @else {
                  <div class="h-[140px] bg-surface-sunken" aria-hidden="true"></div>
                }
                <div class="flex flex-col gap-1.5 p-4">
                  <b class="wrap-anywhere text-[15px]">{{ l.title }}</b>
                  @if (l.farmName) {
                    <span class="text-[13px] text-ink-muted">{{ l.farmName }}</span>
                  }
                  <div class="flex items-baseline justify-between gap-2">
                    <b class="font-display text-[20px]">{{ l.unitPrice | fcfa }}</b>
                    <bt-status-badge kind="listing" [status]="l.status" />
                  </div>
                </div>
              </div>
            }
          </div>

          <div class="bo-card flex flex-col gap-3 p-5">
            <b class="font-display text-[17px]">{{ 'adminReports.receivedTitle' | t }}</b>
            @for (h of d.history; track h.id; let last = $last) {
              <div class="grid gap-x-3 gap-y-1 text-small sm:grid-cols-[110px_170px_minmax(0,1fr)]" [class]="last ? '' : 'border-b border-line pb-2.5'">
                <span class="font-mono text-caption text-ink-muted">{{ h.createdAt | localDate: 'dayMonthTime' }}</span>
                <b class="wrap-anywhere">{{ h.reporterName || ('adminReports.reporterUnknown' | t) }}</b>
                <span class="wrap-anywhere text-ink-muted">{{ h.reason }}</span>
              </div>
            }
          </div>

          @if (staff.can('reports.resolve')) {
            <div class="flex flex-col gap-2">
              <label for="report-note" class="text-[13px] font-semibold">{{ 'adminReports.noteLabel' | t }}</label>
              <textarea
                id="report-note"
                rows="2"
                maxlength="1000"
                [ngModel]="note()"
                (ngModelChange)="note.set($event)"
                class="resize-none rounded-xl border border-line-strong bg-surface-card px-3.5 py-3 text-[14.5px] font-medium text-ink"
              ></textarea>
              @if (note().trim() !== (d.report.note ?? '')) {
                <bt-button class="self-start" variant="outline" size="sm" [disabled]="resolving()" (click)="saveNote()">{{ 'adminReports.noteSave' | t }}</bt-button>
              }
            </div>
          } @else if (d.report.note) {
            <div class="bo-card flex flex-col gap-1.5 p-5">
              <b class="text-[13px]">{{ 'adminReports.noteLabel' | t }}</b>
              <p class="wrap-anywhere text-small text-ink-muted">{{ d.report.note }}</p>
            </div>
          }

          @if (!staff.can('reports.resolve')) {
            <p class="mt-auto border-t border-line pt-4 text-small text-ink-muted">{{ 'adminReports.readOnly' | t }}</p>
          } @else if (d.actions.length > 0) {
            <!-- The API lists what this report can take; nothing here decides it. -->
            <div class="mt-auto flex flex-wrap items-center justify-between gap-3 border-t border-line pt-4">
              <bt-button variant="outline" [disabled]="resolving()" (click)="resolve('dismiss')">{{ 'adminReports.resolveDismiss' | t }}</bt-button>
              <div class="flex flex-wrap gap-2.5">
                @if (d.actions.includes('hide_listing')) {
                  <bt-button variant="outline" [disabled]="resolving()" (click)="resolve('hide_listing')">{{ 'adminReports.resolveHide' | t }}</bt-button>
                }
                @if (d.actions.includes('reverify')) {
                  <bt-button variant="primary" [disabled]="resolving()" (click)="resolve('reverify')">{{ 'adminReports.resolveReverify' | t }}</bt-button>
                }
                @if (d.actions.includes('suspend')) {
                  <bt-button variant="danger" [disabled]="resolving()" (click)="suspendConfirmOpen.set(true)">{{ 'adminReports.resolveSuspend' | t }}</bt-button>
                }
              </div>
            </div>
          } @else {
            <p class="mt-auto border-t border-line pt-4 text-small text-ink-muted">{{ 'adminReports.closed' | t }}</p>
          }
        } @else if (detail.isLoading()) {
          <div class="h-64 animate-pulse rounded-2xl bg-surface-sunken" aria-busy="true"></div>
        } @else if (detail.error()) {
          <p class="text-body text-alert" role="alert">{{ 'adminReports.notFound' | t }}</p>
        }
      </section>
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
      {{ 'adminReports.suspendConfirmBody' | t: { name: detail.value()?.subject?.name ?? '' } }}
    </bt-dialog>
  `
})
export class AdminReportsComponent {
  private readonly adminApi = inject(AdminApi);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);
  protected readonly staff = inject(StaffSpaceService);

  /** `/admin/disputes` sets it: reservations only. */
  protected readonly disputes = toSignal(this.route.data.pipe(map((d) => d['disputes'] === true)), { initialValue: false });
  protected readonly base = computed(() => (this.disputes() ? '/admin/disputes' : '/admin/reports'));
  protected readonly id = toSignal(this.route.paramMap.pipe(map((p) => p.get('id') ?? '')), { initialValue: '' });

  protected readonly filters = FILTERS;
  protected readonly filter = signal<QueueFilter>('open');

  protected readonly reports = apiResource<Report[]>(() => this.adminApi.reportsUrl());
  protected readonly detail = apiResource<ReportDetail>(() => (this.id() ? this.adminApi.reportDetailUrl(this.id()) : undefined));

  /** This page's half of the queue: disputes or everything else. */
  private readonly mine = computed(() => (this.reports.value() ?? []).filter((r) => (r.targetType === 'reservation') === this.disputes()));

  protected readonly shown = computed(() => this.matching(this.filter()));

  /** The internal note, as typed; it starts from what the API holds. */
  protected readonly note = signal('');

  constructor() {
    effect(() => {
      const saved = this.detail.value()?.report.note ?? '';
      untracked(() => this.note.set(saved));
    });
  }

  protected saveNote(): void {
    this.resolving.set(true);
    this.adminApi.saveReportNote(this.id(), this.note().trim()).subscribe({
      next: () => {
        this.resolving.set(false);
        this.detail.reload();
        this.toast.show({ tone: 'success', title: 'adminReports.noteSaved' });
      },
      error: (err: AppError) => {
        this.resolving.set(false);
        this.toast.show({ tone: 'danger', title: 'adminReports.resolveFailedTitle', detail: err.message });
      }
    });
  }

  protected readonly resolving = signal(false);
  protected readonly suspendConfirmOpen = signal(false);

  private matching(filter: QueueFilter): Report[] {
    return this.mine().filter((r) => {
      switch (filter) {
        case 'open':
          return r.status === 'open';
        case 'review':
          return r.status === 'under_review';
        default:
          return r.status === 'handled' || r.status === 'rejected';
      }
    });
  }

  protected count(filter: QueueFilter): number {
    return this.matching(filter).length;
  }

  protected levelChip(level: ReportLevel): string {
    return LEVEL_CHIP[level] ?? LEVEL_CHIP.low;
  }

  protected strong(signal: ReportRiskSignal): boolean {
    return signal.weight >= STRONG_SIGNAL;
  }

  protected signalIcon(signal: ReportRiskSignal): IconName {
    return this.strong(signal) ? 'alert' : 'flag';
  }

  protected since(iso: string): string {
    return elapsed(iso, this.locale);
  }

  protected resolve(action: ResolveAction): void {
    this.resolving.set(true);
    this.adminApi.resolveReport(this.id(), action, this.note().trim() || undefined).subscribe({
      next: () => {
        this.resolving.set(false);
        this.reports.reload();
        this.toast.show({ tone: 'success', title: `adminReports.done.${action}` });
        this.router.navigateByUrl(this.base());
      },
      error: (err: AppError) => {
        this.resolving.set(false);
        this.toast.show({ tone: 'danger', title: 'adminReports.resolveFailedTitle', detail: err.message });
      }
    });
  }
}
