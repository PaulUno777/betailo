import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { CycleSummary, DashboardReservation, DashboardTodo, SellerDashboard } from '../../domain/models';
import { cycleTypeLabelKey } from '../../domain/reference/cycle-types';
import { speciesLabelKey } from '../../domain/reference/species';
import { DashboardApi } from '../../data-access/dashboard/dashboard.api';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
import { AvatarComponent, AvatarTone } from '../../shared/ui/avatar/avatar.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { IconName } from '../../shared/ui/icon/icon-paths';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { SellerOnboardingComponent } from '../sell/seller-onboarding.component';

const TODO_ICONS: Readonly<Record<DashboardTodo['kind'], IconName>> = {
  mark_delivered: 'check',
  health_log: 'leaf',
  awaiting_payment: 'wallet'
};

const AVATAR_TONES: readonly AvatarTone[] = ['clay', 'sky', 'forest', 'plum'];
const WEEK_MS = 7 * 24 * 60 * 60 * 1000;

/** ISO 8601 week number of a date. */
function isoWeek(date: Date): number {
  const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
  const day = d.getUTCDay() || 7;
  d.setUTCDate(d.getUTCDate() + 4 - day);
  const yearStart = Date.UTC(d.getUTCFullYear(), 0, 1);
  return Math.ceil(((d.getTime() - yearStart) / 86_400_000 + 1) / 7);
}

/** `/portal` — the seller's day, per `Portal.dc.html`: a greeting with
 * the one primary action, four KPI tiles, the active cycles as rows
 * with how full each is, the last eight weeks of reservations, the
 * latest reservations with who made them, and "À faire aujourd'hui".
 * Every figure comes from `GET /farms/{farmId}/dashboard`, for the farm
 * picked in the sidebar switcher; the server decides what needs doing.
 * An account without a farm gets the farm creation step instead. */
@Component({
  selector: 'bt-portal-dashboard',
  imports: [RouterLink, AvatarComponent, StatusBadgeComponent, IconComponent, ButtonComponent, SellerOnboardingComponent, LocalDatePipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (!seller.loaded()) {
      <p class="p-8 text-body text-ink-muted">{{ 'portal.loading' | t }}</p>
    } @else if (!myFarm()) {
      <div class="mx-auto max-w-lg p-4 sm:p-8">
        <bt-seller-onboarding (created)="seller.refresh()" />
      </div>
    } @else {
      <div class="flex flex-col gap-6 p-4 sm:p-8 lg:px-10">
        <header class="flex flex-wrap items-center justify-between gap-x-6 gap-y-4">
          <div class="flex min-w-0 flex-col gap-1">
            <span class="text-small text-ink-muted first-letter:uppercase">{{ today() }} · {{ myFarm()!.name }}</span>
            <h1 class="font-display text-[30px] leading-none font-extrabold tracking-[-0.03em] sm:text-[36px]">{{ 'portal.greeting' | t: { name: firstName() } }}</h1>
          </div>
          <bt-button variant="primary" href="/portal/cycles/new">
            <bt-icon name="plus" [size]="18" />
            {{ 'portal.newCycleButton' | t }}
          </bt-button>
        </header>

        @if (dashboard.value(); as d) {
          <div class="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4">
            <div class="flex flex-col gap-2 rounded-[20px] border border-line bg-surface-card p-4 sm:p-5">
              <span class="text-eyebrow text-ink-muted">{{ 'portal.kpiReservations' | t: { month: month() } }}</span>
              <span class="font-display text-[30px] leading-none font-extrabold tracking-[-0.02em] sm:text-[38px]">{{ d.reservationsThisPeriod }}</span>
              <span class="text-small text-ink-muted">{{ 'portal.kpiReservationsHint' | t }}</span>
            </div>
            <div class="flex flex-col gap-2 panel-dark rounded-[20px] bg-forest-deep p-4 text-on-forest-deep sm:p-5 xl:order-last">
              <span class="truncate text-eyebrow text-on-forest-deep/70">{{ 'portal.kpiTrustScore' | t }}</span>
              <span class="font-display text-[30px] leading-none font-extrabold sm:text-[38px]">
                {{ trust(d.trustScore) }}<span class="text-[18px] text-on-forest-deep/70">/100</span>
              </span>
              <a [routerLink]="['/farms', myFarm()!.id, 'reviews']" class="text-small font-semibold text-clay-soft underline-offset-2 hover:underline">{{ 'portal.kpiTrustLink' | t }}</a>
            </div>
            <div class="col-span-2 flex flex-col gap-2 rounded-[20px] border border-sky/15 bg-sky-soft p-4 sm:p-5 xl:col-span-1">
              <span class="text-eyebrow text-sky">{{ 'portal.kpiEscrowHeld' | t }}</span>
              <span class="font-display text-[30px] leading-none font-extrabold tracking-[-0.02em] text-sky sm:text-[38px]">{{ amount(d.escrowHeldTotal) }}</span>
              <span class="text-small text-sky">{{ 'portal.kpiEscrowHint' | t }}</span>
            </div>
            <div class="col-span-2 flex flex-col gap-2 rounded-[20px] border border-forest/15 bg-forest-soft p-4 sm:p-5 xl:col-span-1">
              <span class="text-eyebrow text-forest">{{ 'portal.kpiReleased' | t }}</span>
              <span class="font-display text-[30px] leading-none font-extrabold tracking-[-0.02em] text-on-forest-soft sm:text-[38px]">{{ amount(d.releasedThisPeriod) }}</span>
              <span class="text-small text-forest">{{ 'portal.kpiReleasedHint' | t }}</span>
            </div>
          </div>

          <div class="grid gap-5 xl:grid-cols-[minmax(0,1fr)_380px]">
            <section class="flex flex-col gap-4 rounded-[20px] border border-line bg-surface-card p-4 sm:p-6">
              <div class="flex items-center justify-between gap-3">
                <h2 class="font-display text-[20px] font-bold sm:text-[22px]">{{ 'portal.activeCyclesTitle' | t }}</h2>
                <a routerLink="/portal/cycles" class="text-small font-semibold text-forest underline-offset-2 hover:underline">{{ 'portal.seeAll' | t }}</a>
              </div>
              @if (d.activeCycles.length === 0) {
                <div class="flex flex-col items-start gap-3 rounded-[14px] border border-dashed border-line-strong p-5">
                  <p class="text-body text-ink-muted">{{ 'portal.noActiveCycles' | t }}</p>
                  <bt-button variant="outline" size="sm" href="/portal/cycles/new">{{ 'portal.newCycleButton' | t }}</bt-button>
                </div>
              } @else {
                <div class="hidden grid-cols-[minmax(0,2.2fr)_minmax(0,1fr)_minmax(0,1.6fr)_minmax(0,1fr)_40px] gap-4 border-b border-line px-3 pb-2.5 text-eyebrow text-ink-muted md:grid">
                  <span>{{ 'portal.cycleColHeader' | t }}</span>
                  <span>{{ 'portal.statusColHeader' | t }}</span>
                  <span>{{ 'portal.fillColHeader' | t }}</span>
                  <span>{{ 'portal.dueColHeader' | t }}</span>
                  <span></span>
                </div>
                <ul class="flex flex-col gap-2.5">
                  @for (c of d.activeCycles; track c.cycle.id) {
                    <li>
                      <a
                        [routerLink]="['/portal/cycles', c.cycle.id]"
                        class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-2.5 rounded-[14px] border border-line p-3 text-ink transition hover:border-line-strong hover:bg-surface-raised md:grid-cols-[minmax(0,2.2fr)_minmax(0,1fr)_minmax(0,1.6fr)_minmax(0,1fr)_40px]"
                      >
                        <span class="flex min-w-0 flex-col gap-0.5">
                          <b class="truncate text-[15px]">{{ speciesKey(c.cycle.species) | t }}</b>
                          <span class="truncate font-mono text-[11.5px] text-ink-muted">{{ c.cycle.reference || c.cycle.id.slice(-6) }} · {{ typeKey(c.cycle.type) | t }}</span>
                        </span>
                        <bt-status-badge class="justify-self-end md:justify-self-start" kind="cycle" [status]="c.cycle.status" />
                        <span class="col-span-2 flex flex-col gap-1.5 md:col-span-1">
                          <span
                            class="h-2 overflow-hidden rounded-full bg-surface-sunken"
                            role="progressbar"
                            [attr.aria-valuenow]="reserved(c)"
                            aria-valuemin="0"
                            [attr.aria-valuemax]="c.cycle.capacity"
                            [attr.aria-label]="'portal.fillColHeader' | t"
                          >
                            <span class="block h-full rounded-full" [class]="isFull(c) ? 'bg-clay' : 'bg-forest'" [style.width.%]="fillPct(c)"></span>
                          </span>
                          <span class="text-[12.5px] text-ink-muted">
                            {{ (isFull(c) ? 'portal.fillFull' : 'portal.fillPartial') | t: { reserved: reserved(c), capacity: c.cycle.capacity } }}
                          </span>
                        </span>
                        <span class="col-span-2 text-[13.5px] md:col-span-1">
                          {{ c.nextMilestone }}
                          @if (c.nextMilestoneDate) {
                            {{ c.nextMilestoneDate | localDate: 'dayMonth' }}
                          }
                        </span>
                        <span class="hidden h-10 w-10 items-center justify-center rounded-[10px] border border-line text-ink md:flex" aria-hidden="true">
                          <bt-icon name="chevron-right" [size]="18" />
                        </span>
                      </a>
                    </li>
                  }
                </ul>
              }

              <div class="mt-auto flex flex-col gap-3 border-t border-line pt-4">
                <div class="flex items-baseline justify-between gap-3">
                  <b class="text-[15px]">{{ 'portal.weeklyChartTitle' | t }}</b>
                  <span class="text-[12.5px] text-ink-muted">{{ 'portal.weeklyChartHint' | t }}</span>
                </div>
                <div class="grid h-[124px] grid-cols-8 items-end gap-2 sm:gap-3.5" role="img" [attr.aria-label]="weeksSummary()">
                  @for (w of weeks(); track w.label) {
                    <div class="flex h-full flex-col items-center justify-end gap-1.5">
                      <span class="font-mono text-[11px] text-ink-muted">{{ w.value }}</span>
                      <div class="w-full rounded-t-md rounded-b-xs" [class]="w.current ? 'bg-clay' : 'bg-forest'" [style.height.px]="w.height"></div>
                      <span class="font-mono text-[10.5px] text-ink-muted">{{ w.label }}</span>
                    </div>
                  }
                </div>
              </div>
            </section>

            <div class="flex flex-col gap-5">
              <section class="flex flex-col gap-3.5 rounded-[20px] border border-line bg-surface-card p-4 sm:p-[22px]">
                <div class="flex items-center justify-between gap-3">
                  <h2 class="font-display text-[20px] font-bold">{{ 'portal.latestReservationsTitle' | t }}</h2>
                  <a routerLink="/portal/reservations" class="text-small font-semibold text-forest underline-offset-2 hover:underline">{{ 'portal.seeAll' | t }}</a>
                </div>
                @if (d.latestReservations.length === 0) {
                  <p class="text-body text-ink-muted">{{ 'portal.noReservations' | t }}</p>
                } @else {
                  <ul class="flex flex-col gap-3.5">
                    @for (r of d.latestReservations; track r.id; let i = $index) {
                      <li class="flex items-center gap-3">
                        <bt-avatar [name]="buyer(r)" [size]="38" [tone]="avatarTone(i)" class="shrink-0" />
                        <span class="flex min-w-0 flex-1 flex-col">
                          <b class="truncate text-small">{{ buyer(r) }}</b>
                          <span class="truncate text-[12.5px] text-ink-muted">
                            {{ r.quantity }}
                            @if (r.species) {
                              {{ speciesKey(r.species) | t }}
                            }
                            · {{ r.cycleRef || r.reference }}
                          </span>
                        </span>
                        <span class="flex flex-col items-end gap-1">
                          <span class="font-mono text-[13px] font-medium tabular-nums">{{ amount(r.totalAmount) }}</span>
                          @if (r.payment && r.status !== 'cancelled') {
                            <bt-status-badge kind="escrow" [status]="r.payment.status" />
                          } @else {
                            <bt-status-badge kind="reservation" [status]="r.status" />
                          }
                        </span>
                      </li>
                    }
                  </ul>
                }
              </section>

              <section class="flex flex-col gap-3 rounded-[20px] bg-clay p-4 text-on-clay sm:p-[22px]">
                <div class="flex items-center gap-2.5">
                  <bt-icon name="bolt" [size]="20" />
                  <h2 class="font-display text-[18px] font-bold">{{ 'portal.todoTitle' | t }}</h2>
                </div>
                @if (d.todos.length === 0) {
                  <p class="text-small leading-relaxed">{{ 'portal.todoEmpty' | t }}</p>
                } @else {
                  <ul class="flex flex-col gap-2">
                    @for (todo of d.todos; track $index) {
                      <li>
                        <a [routerLink]="todoRoute(todo)" class="flex min-h-12 items-center gap-3 rounded-xl bg-ink px-3.5 py-2 text-small font-semibold text-surface transition hover:bg-forest-deep">
                          <bt-icon [name]="todoIcon(todo)" [size]="18" class="shrink-0 text-clay" />
                          <span class="flex-1">{{ 'portal.todo.' + todo.kind | t: { count: todo.count ?? 1, ref: todo.cycleRef ?? '' } }}</span>
                          <bt-icon name="chevron-right" [size]="16" class="shrink-0" />
                        </a>
                      </li>
                    }
                  </ul>
                }
              </section>
            </div>
          </div>
        } @else if (dashboard.error()) {
          <div class="flex flex-col items-start gap-3" role="alert">
            <p class="text-body">{{ 'apiErrors.server' | t }}</p>
            <bt-button variant="outline" (click)="dashboard.reload()">{{ 'common.retry' | t }}</bt-button>
          </div>
        } @else {
          <div class="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4" aria-hidden="true">
            @for (i of [1, 2, 3, 4]; track i) {
              <div class="h-[132px] animate-pulse rounded-[20px] bg-surface-sunken"></div>
            }
          </div>
          <div class="h-80 animate-pulse rounded-[20px] bg-surface-sunken" aria-hidden="true"></div>
        }
      </div>
    }
  `
})
export class PortalDashboardComponent {
  protected readonly seller = inject(SellerSpaceService);
  private readonly dashboardApi = inject(DashboardApi);
  private readonly locale = inject(LocaleService);

  protected readonly myFarm = computed(() => this.seller.currentFarm() ?? undefined);

  protected readonly dashboard = apiResource<SellerDashboard>(() => {
    const farmId = this.myFarm()?.id;
    return farmId ? this.dashboardApi.sellerDashboardUrl(farmId) : undefined;
  });

  private readonly intlLocale = computed(() => (this.locale.locale() === 'fr' ? 'fr-FR' : 'en-GB'));

  protected readonly today = computed(() => new Intl.DateTimeFormat(this.intlLocale(), { weekday: 'long', day: 'numeric', month: 'long' }).format(new Date()));
  protected readonly month = computed(() => new Intl.DateTimeFormat(this.intlLocale(), { month: 'short' }).format(new Date()));

  /** "Bonjour Bertrand": the first word of the account's name, or the
   * farm's when the account has none. */
  protected readonly firstName = computed(() => {
    const name = this.seller.account()?.displayName?.trim();
    return name ? name.split(/\s+/)[0] : (this.myFarm()?.name ?? '');
  });

  /** The API's eight weekly counts, oldest first, labelled by ISO week
   * and scaled to the tallest bar. */
  protected readonly weeks = computed(() => {
    const counts = this.dashboard.value()?.weeklyReservationCounts ?? [];
    const max = Math.max(1, ...counts);
    const now = Date.now();
    return counts.map((value, i) => ({
      value,
      label: this.locale.t('portal.weekLabel', { n: isoWeek(new Date(now - (counts.length - 1 - i) * WEEK_MS)) }),
      height: Math.max(4, Math.round((value / max) * 80)),
      current: i === counts.length - 1
    }));
  });

  protected readonly weeksSummary = computed(() =>
    this.weeks()
      .map((w) => `${w.label}: ${w.value}`)
      .join(', ')
  );

  protected amount(value: number): string {
    return Math.round(value)
      .toLocaleString('fr-FR')
      .replace(/\s/g, ' ');
  }

  protected trust(score: number): number {
    return Math.round(score);
  }

  protected reserved(c: CycleSummary): number {
    return Math.max(0, c.cycle.capacity - c.cycle.availableQuantity);
  }

  protected isFull(c: CycleSummary): boolean {
    return c.cycle.capacity > 0 && c.cycle.availableQuantity <= 0;
  }

  protected fillPct(c: CycleSummary): number {
    return c.cycle.capacity > 0 ? Math.min(100, Math.round((this.reserved(c) / c.cycle.capacity) * 100)) : 0;
  }

  protected buyer(r: DashboardReservation): string {
    return r.buyerName || this.locale.t('portal.buyerUnnamed');
  }

  protected avatarTone(index: number): AvatarTone {
    return AVATAR_TONES[index % AVATAR_TONES.length];
  }

  protected typeKey(type: string): string {
    return cycleTypeLabelKey(type);
  }

  protected speciesKey(value: string): string {
    return speciesLabelKey(value);
  }

  /** Each "À faire" item opens the page where it gets done. */
  protected todoRoute(todo: DashboardTodo): string | readonly string[] {
    return todo.kind === 'health_log' && todo.cycleId ? ['/portal/cycles', todo.cycleId] : '/portal/reservations';
  }

  protected todoIcon(todo: DashboardTodo): IconName {
    return TODO_ICONS[todo.kind];
  }
}
