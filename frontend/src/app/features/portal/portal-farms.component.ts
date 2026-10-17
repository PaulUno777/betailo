import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { Farm, FarmCard } from '../../domain/models';
import { speciesLabelKey } from '../../domain/reference/species';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { apiResource } from '../../core/http/api-resource';
import { LocaleService } from '../../core/i18n/locale.service';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { PlacePipe } from '../../core/i18n/place.pipe';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
import { AppError } from '../../core/http/app-error';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { elapsed } from '../admin/ui/elapsed';
import { FarmProofFormComponent } from '../sell/farm-proof-form.component';
import { SellerOnboardingComponent } from '../sell/seller-onboarding.component';

/** "Mes fermes" — per `Farms.dc.html`: one wide card per farm (its
 * photo, where it stands with the review, its figures once it sells,
 * and what its owner can do next), the account's quota beside the
 * "create" action, and the three steps of a review underneath.
 *
 * Status, refusal reason, figures, the limit and whether another farm
 * is allowed all come from the API (`GET /farms/mine/cards`, `GET /me`).
 * The photo is the first one sent with the farm's proof, served through
 * a short-lived link only its owner gets. */
@Component({
  selector: 'bt-portal-farms',
  imports: [RouterLink, ButtonComponent, DialogComponent, IconComponent, StatusBadgeComponent, FarmProofFormComponent, SellerOnboardingComponent, LocalDatePipe, TranslatePipe, PlacePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6 p-4 sm:p-8 lg:px-10">
      <header class="flex flex-wrap items-end justify-between gap-6">
        <div class="flex flex-col gap-1.5">
          <span class="text-eyebrow text-ink-muted">{{ 'portalFarms.eyebrow' | t }}</span>
          <h1 class="font-display text-[32px] leading-none font-extrabold tracking-[-0.03em] sm:text-[40px]">{{ 'portalFarms.title' | t }}</h1>
          <p class="max-w-[620px] text-[15px] text-ink-muted">{{ 'portalFarms.intro' | t }}</p>
        </div>
        <div class="flex flex-col items-start gap-2 sm:items-end">
          <bt-button variant="primary" [disabled]="!seller.canAddFarm() || adding()" (click)="adding.set(true)">
            <bt-icon name="plus" [size]="16" />
            {{ 'portalFarms.add' | t }}
          </bt-button>
          <div class="flex flex-wrap items-center gap-2 text-[13px] text-ink-muted">
            <span class="inline-flex gap-1" aria-hidden="true">
              @for (used of quotaDots(); track $index) {
                <span class="h-2.5 w-2.5 rounded-full" [class]="used ? 'bg-forest' : 'border-[1.5px] border-line-strong'"></span>
              }
            </span>
            <span>
              <b class="text-ink">{{ 'portalFarms.quota' | t: { count: seller.farms().length, max: seller.farmLimit() } }}</b>
              @if (!seller.canAddFarm()) {
                · {{ 'portalFarms.quotaReached' | t }}
              }
            </span>
          </div>
          @if (seller.account()?.farmLimitRaisedAt; as at) {
            <span class="text-[12.5px] font-semibold text-clay-ink">{{ 'portalFarms.limitRaised' | t: { max: seller.farmLimit(), date: (at | localDate: 'dayMonth') } }}</span>
          } @else if (!seller.canAddFarm()) {
            <span class="max-w-xs text-[12.5px] text-ink-muted sm:text-right">{{ 'portalFarms.limitHelp' | t }}</span>
          }
        </div>
      </header>

      @if (adding()) {
        <section class="rounded-[20px] border border-line bg-surface-card p-5 sm:p-6" [attr.aria-label]="'portalFarms.add' | t">
          <bt-seller-onboarding (created)="onFarmAdded($event)" />
          <bt-button class="mt-3 block" variant="ghost" block (click)="adding.set(false)">{{ 'common.cancel' | t }}</bt-button>
        </section>
      }

      @if (cards.isLoading() && !cards.hasValue()) {
        <div class="h-56 animate-pulse rounded-[20px] bg-surface-sunken" aria-busy="true"></div>
      } @else if (cards.error()) {
        <div class="flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="cards.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      } @else {
        @for (farm of cards.value() ?? []; track farm.id) {
          <article
            class="grid overflow-hidden rounded-[20px] border bg-surface-card lg:grid-cols-[220px_minmax(0,1fr)_260px]"
            [class]="farm.status === 'rejected' || farm.status === 'suspended' ? 'border-alert/40' : 'border-line'"
          >
            @if (farm.bannerUrl || farm.coverUrl; as cover) {
              <img [src]="cover" [alt]="'portalFarms.photoAlt' | t: { name: farm.name }" class="h-44 w-full object-cover lg:h-full lg:min-h-[220px]" [class.saturate-50]="farm.status === 'rejected'" />
            } @else {
              <div class="flex h-32 w-full items-center justify-center bg-forest-soft font-display text-[40px] font-extrabold text-forest lg:h-full lg:min-h-[220px]" aria-hidden="true">
                {{ initials(farm.name) }}
              </div>
            }

            <div class="flex min-w-0 flex-col gap-3.5 p-5 sm:p-6">
              <div class="flex flex-wrap items-center gap-2.5">
                <h2 class="wrap-anywhere font-display text-[24px] leading-tight font-extrabold">{{ farm.name }}</h2>
                <bt-status-badge kind="farm" [status]="farm.status" />
                @if (isCurrent(farm)) {
                  <span class="inline-flex h-6 items-center gap-1 rounded-full border border-line px-2.5 text-caption font-semibold text-ink-muted">
                    <bt-icon name="check" [size]="13" />{{ 'portalFarms.current' | t }}
                  </span>
                }
              </div>
              <span class="wrap-anywhere text-small text-ink-muted">
                {{ farm.location | place }}{{ farm.region ? ', ' + (farm.region | place) : '' }}
                @for (s of farm.species ?? []; track s) {
                  · {{ speciesKey(s) | t }}
                }
                @if (statusDate(farm); as line) {
                  · {{ line.key | t: { date: (line.at | localDate: 'dayMonthTime') } }}
                  @if (farm.status === 'rejected' && farm.review?.reviewerRole; as role) {
                    {{ 'portalFarms.byTeam' | t: { team: ('staffRole.' + role | t) } }}
                  }
                }
              </span>

              @switch (farm.status) {
                @case ('approved') {
                  <dl class="mt-1 grid grid-cols-2 gap-3 sm:grid-cols-4">
                    <div class="flex flex-col gap-0.5">
                      <dd class="font-display text-[26px] leading-tight font-bold">{{ farm.activeCycles }}</dd>
                      <dt class="text-[12.5px] text-ink-muted">{{ 'portalFarms.statCycles' | t }}</dt>
                    </div>
                    <div class="flex flex-col gap-0.5">
                      <dd class="font-display text-[26px] leading-tight font-bold">{{ farm.salesCount }}</dd>
                      <dt class="text-[12.5px] text-ink-muted">{{ 'portalFarms.statSales' | t }}</dt>
                    </div>
                    <div class="flex flex-col gap-0.5">
                      <dd class="font-display text-[26px] leading-tight font-bold text-forest">{{ round(farm.trustScore) }}</dd>
                      <dt class="text-[12.5px] text-ink-muted">{{ 'portalFarms.statTrust' | t }}</dt>
                    </div>
                    <div class="flex flex-col gap-0.5">
                      <dd class="font-display text-[26px] leading-tight font-bold">
                        @if (farm.reviewCount > 0) {
                          {{ rating(farm.reviewAverage) }}<bt-icon name="star" [size]="16" class="ml-1 text-clay" />
                        } @else {
                          —
                        }
                      </dd>
                      <dt class="text-[12.5px] text-ink-muted">{{ 'portalFarms.statReviews' | t: { count: farm.reviewCount } }}</dt>
                    </div>
                  </dl>
                  @if (farm.review?.autoApproved) {
                    <p class="text-[13px] text-ink-muted">{{ 'portalFarms.autoApproved' | t }}</p>
                  }
                }
                @case ('pending') {
                  <ol class="grid gap-3 sm:grid-cols-3">
                    <li class="flex flex-col gap-1.5 text-[13.5px]">
                      <span class="flex h-[22px] w-[22px] items-center justify-center rounded-full bg-forest text-on-forest"><bt-icon name="check" [size]="12" /></span>
                      <b>{{ 'portalFarms.stepSent' | t }}</b>
                      <span class="text-[12.5px] text-ink-muted">{{ 'portalFarms.stepSentHint' | t }}</span>
                    </li>
                    <li class="flex flex-col gap-1.5 text-[13.5px]">
                      <span class="h-[22px] w-[22px] rounded-full bg-clay ring-[5px] ring-clay-soft"></span>
                      <b>{{ 'portalFarms.stepReview' | t }}</b>
                      <span class="text-[12.5px] text-ink-muted">
                        @if (farm.review?.submittedAt; as at) {
                          {{ 'portalFarms.stepReviewHint' | t: { elapsed: since(at) } }}
                        }
                      </span>
                    </li>
                    <li class="flex flex-col gap-1.5 text-[13.5px]">
                      <span class="h-[22px] w-[22px] rounded-full border-2 border-line-strong"></span>
                      <b>{{ 'portalFarms.stepDecision' | t }}</b>
                      <span class="text-[12.5px] text-ink-muted">{{ 'portalFarms.stepDecisionHint' | t }}</span>
                    </li>
                  </ol>
                  <div class="flex items-start gap-2.5 rounded-xl bg-amber-soft px-3.5 py-3 text-[13.5px] leading-normal text-amber">
                    <bt-icon name="lock" [size]="16" class="mt-0.5" />
                    <span>{{ 'portalFarms.pendingLock' | t }}</span>
                  </div>
                }
                @case ('rejected') {
                  <div class="flex flex-col gap-1.5 rounded-[14px] bg-alert-soft px-4 py-3.5">
                    <b class="text-small text-alert">{{ 'portalFarms.rejectedReason' | t: { reason: ('rejectionCode.' + (farm.review?.rejectionCode || 'other') | t) } }}</b>
                    @if (farm.review?.rejectionReason; as message) {
                      <span class="wrap-anywhere text-small leading-normal text-ink">« {{ message }} »</span>
                    }
                  </div>
                  <ul class="flex flex-wrap gap-x-[18px] gap-y-1.5 text-[13.5px]">
                    @for (part of farm.review?.toRedo ?? []; track part) {
                      <li class="flex items-center gap-1.5 text-alert"><bt-icon name="x" [size]="15" />{{ 'portalFarms.redo.' + part | t }}</li>
                    }
                    @for (part of farm.review?.accepted ?? []; track part) {
                      <li class="flex items-center gap-1.5 text-forest">
                        <bt-icon name="check" [size]="15" />{{ 'portalFarms.ok.' + part | t: { count: farm.review?.photoCount ?? 0 } }}
                      </li>
                    }
                  </ul>
                }
                @case ('suspended') {
                  <div class="flex flex-col gap-1.5 rounded-[14px] bg-alert-soft px-4 py-3.5">
                    <b class="text-small text-alert">{{ 'portalFarms.suspendedTitle' | t }}</b>
                    <span class="wrap-anywhere text-small leading-normal text-ink">{{ farm.review?.rejectionReason || ('portalFarms.suspendedBody' | t) }}</span>
                  </div>
                }
                @case ('draft') {
                  <div class="flex items-start gap-2.5 rounded-xl bg-amber-soft px-3.5 py-3 text-[13.5px] leading-normal text-amber">
                    <bt-icon name="upload" [size]="16" class="mt-0.5" />
                    <span>{{ 'portalFarms.draftBody' | t }}</span>
                  </div>
                }
              }
            </div>

            <div class="flex flex-col justify-center gap-2.5 border-t border-line p-5 sm:p-6 lg:border-t-0 lg:border-l">
              @switch (farm.status) {
                @case ('approved') {
                  <bt-button variant="solid" block (click)="open(farm, '/portal/cycles/new')">{{ 'portalFarms.newCycle' | t }}</bt-button>
                  <bt-button variant="outline" block (click)="open(farm, '/portal')">{{ 'portalFarms.dashboard' | t }}</bt-button>
                  <bt-button variant="ghost" block class="text-center" [href]="'/farms/' + farm.id + '/reviews'">{{ 'portalFarms.publicPage' | t }}</bt-button>
                }
                @case ('pending') {
                  <bt-button variant="outline" block (click)="open(farm, '/portal/cycles/new')">{{ 'portalFarms.prepareDraft' | t }}</bt-button>
                  <bt-button variant="solid" block disabled [attr.title]="'portalFarms.publishLocked' | t">{{ 'portalFarms.publishCycle' | t }}</bt-button>
                  @if (farm.review?.canWithdraw) {
                    <bt-button variant="ghost" block class="[&_button]:text-alert" [disabled]="withdrawing() === farm.id" (click)="withdrawTarget.set(farm)">
                      {{ 'portalFarms.withdraw' | t }}
                    </bt-button>
                  }
                }
                @case ('suspended') {
                  <span class="text-center text-[12.5px] text-ink-muted">{{ 'portalFarms.suspendedHelp' | t }}</span>
                }
                @default {
                  @if (farm.review?.canSubmit) {
                    <bt-button variant="primary" block (click)="submitting.set(farm.id)">
                      {{ (farm.status === 'rejected' ? 'portalFarms.resubmit' : 'portalFarms.submit') | t }}
                    </bt-button>
                  }
                  <span class="text-center text-[12.5px] text-ink-muted">{{ (farm.status === 'rejected' ? 'portalFarms.rejectedHint' : 'portalFarms.draftHint') | t }}</span>
                }
              }
              <a routerLink="/portal/farm" (click)="seller.select(farm.id)" class="inline-flex min-h-11 items-center justify-center text-small font-semibold text-forest underline-offset-2 hover:underline">
                {{ 'portalFarms.edit' | t }}
              </a>
            </div>

            @if (submitting() === farm.id) {
              <div class="col-span-full flex flex-col gap-3 border-t border-line p-5 sm:p-6">
                <bt-farm-proof-form [farm]="farm" (submitted)="onSubmitted($event)" />
                <bt-button variant="ghost" block (click)="submitting.set(null)">{{ 'common.cancel' | t }}</bt-button>
              </div>
            }
          </article>
        }
      }

      <bt-dialog
        [open]="withdrawTarget() !== null"
        [title]="'portalFarms.withdrawTitle' | t: { farm: withdrawTarget()?.name ?? '' }"
        [primaryLabel]="'portalFarms.withdraw' | t"
        primaryVariant="danger"
        [secondaryLabel]="'common.cancel' | t"
        (primaryClick)="withdraw()"
        (secondaryClick)="withdrawTarget.set(null)"
        (closed)="withdrawTarget.set(null)"
      >
        {{ 'portalFarms.withdrawBody' | t }}
      </bt-dialog>

      <section class="mt-1 grid gap-4 sm:grid-cols-3" [attr.aria-label]="'portalFarms.howLabel' | t">
        @for (step of [1, 2, 3]; track step) {
          <div class="flex items-start gap-3">
            <span class="font-display text-[28px] leading-none font-extrabold text-forest">{{ step }}</span>
            <div class="flex flex-col gap-0.5">
              <b>{{ 'portalFarms.how' + step + 'Title' | t }}</b>
              <span class="text-[12.5px] text-ink-muted">{{ 'portalFarms.how' + step + 'Body' | t }}</span>
            </div>
          </div>
        }
      </section>
    </div>
  `
})
export class PortalFarmsComponent {
  protected readonly seller = inject(SellerSpaceService);
  private readonly farmsApi = inject(FarmsApi);
  private readonly router = inject(Router);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);

  protected readonly cards = apiResource<FarmCard[]>(() => this.farmsApi.mineCardsUrl());

  /** Opening another farm (the two-step flow, inline). */
  protected readonly adding = signal(false);
  /** The farm whose proof form is open. */
  protected readonly submitting = signal<string | null>(null);
  /** The pending farm whose request the owner is about to withdraw. */
  protected readonly withdrawTarget = signal<Farm | null>(null);
  protected readonly withdrawing = signal<string | null>(null);

  protected withdraw(): void {
    const farm = this.withdrawTarget();
    if (!farm) return;
    this.withdrawTarget.set(null);
    this.withdrawing.set(farm.id);
    this.farmsApi.withdrawSubmission(farm.id).subscribe({
      next: () => {
        this.withdrawing.set(null);
        this.seller.refresh();
        this.cards.reload();
        this.toast.show({ tone: 'success', title: 'portalFarms.withdrawnToast' });
      },
      error: (err: AppError) => {
        this.withdrawing.set(null);
        this.toast.show({ tone: 'danger', title: 'portalFarms.withdrawFailed', detail: err.message });
      }
    });
  }

  /** One dot per allowed farm, filled for each one owned. */
  protected readonly quotaDots = computed(() => {
    const owned = this.seller.farms().length;
    const max = Math.max(this.seller.farmLimit(), owned);
    return Array.from({ length: Math.min(max, 10) }, (_, i) => i < owned);
  });

  constructor() {
    // Staff may have decided since the last look.
    this.seller.refresh();
  }

  protected isCurrent(farm: Farm): boolean {
    return this.seller.farms().length > 1 && this.seller.currentFarm()?.id === farm.id;
  }

  /** Works on this farm, then goes where the action leads. */
  protected open(farm: Farm, url: string): void {
    this.seller.select(farm.id);
    this.router.navigateByUrl(url);
  }

  protected speciesKey(value: string): string {
    return speciesLabelKey(value);
  }

  protected initials(name: string): string {
    return name
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((word) => word[0]?.toUpperCase() ?? '')
      .join('');
  }

  protected round(value: number): number {
    return Math.round(value);
  }

  protected rating(value: number): string {
    return value.toLocaleString(this.locale.locale() === 'fr' ? 'fr-FR' : 'en-GB', { minimumFractionDigits: 1, maximumFractionDigits: 1 });
  }

  protected since(iso: string): string {
    return elapsed(iso, this.locale);
  }

  /** The date that matters for the farm's state, with its wording. */
  protected statusDate(farm: Farm): { key: string; at: string } | null {
    const review = farm.review;
    if (farm.status === 'pending' && review?.submittedAt) {
      return { key: 'portalFarms.sentOn', at: review.submittedAt };
    }
    if (review?.reviewedAt && (farm.status === 'approved' || farm.status === 'rejected')) {
      return { key: farm.status === 'approved' ? 'portalFarms.approvedOn' : 'portalFarms.rejectedOn', at: review.reviewedAt };
    }
    return null;
  }

  protected onFarmAdded(farm: Farm): void {
    this.adding.set(false);
    this.seller.select(farm.id);
    this.seller.refresh();
    this.cards.reload();
    this.toast.show({ tone: 'success', title: 'sellerOnboarding.done' });
  }

  /** The API says what became of it: pending, or approved on the spot
   * on a platform that approves without review. */
  protected onSubmitted(farm: Farm): void {
    this.submitting.set(null);
    this.seller.refresh();
    this.cards.reload();
    this.toast.show({
      tone: 'success',
      title: farm.status === 'approved' ? 'portalFarms.approvedToast' : 'portalFarms.submittedToast',
      detail: farm.status === 'approved' ? undefined : 'portalFarms.submittedToastDetail'
    });
  }
}
