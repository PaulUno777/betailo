import { ChangeDetectionStrategy, Component, computed, inject, linkedSignal, signal } from '@angular/core';
import { NgTemplateOutlet } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { map } from 'rxjs';
import { ListingDetail } from '../../domain/models';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { ReportsApi } from '../../data-access/reports/reports.api';
import { AuthService } from '../../core/auth/auth.service';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { priorityChoiceCopy } from '../../domain/rules/priority-choice';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { CycleTimelineComponent } from '../../shared/ui/cycle-timeline/cycle-timeline.component';
import { CapacityBarComponent } from '../../shared/ui/capacity-bar/capacity-bar.component';
import { RatingComponent } from '../../shared/ui/rating/rating.component';
import { RatingSummaryComponent } from '../../shared/ui/rating-summary/rating-summary.component';
import { ReviewCardComponent } from '../../shared/ui/review-card/review-card.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { ImageGalleryComponent } from '../../shared/ui/image-gallery/image-gallery.component';
import { StickyActionBarComponent } from '../../shared/ui/sticky-action-bar/sticky-action-bar.component';
import { FarmCardComponent } from '../../shared/ui/farm-card/farm-card.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';

/** The listing page, mobile-first per the design: full-bleed swipeable
 * photos with back/share/save over them, the content sheet sliding up
 * over the photos, and a sticky price + "Réserver" bar in the thumb
 * zone. From `lg` up the same content sits beside a booking panel.
 *
 * Everything comes from one request (`GET /listings/{id}/detail`), and
 * every decision about what the viewer may do (reserve, edit, contact)
 * is the API's (`canReserve`, `isOwner`, a contact phone only for
 * signed-in viewers). This page only renders them. */
@Component({
  selector: 'bt-listing-detail',
  imports: [
    NgTemplateOutlet,
    FormsModule,
    StatusBadgeComponent,
    CycleTimelineComponent,
    CapacityBarComponent,
    RatingComponent,
    RatingSummaryComponent,
    ReviewCardComponent,
    ButtonComponent,
    NoticeComponent,
    DialogComponent,
    TextFieldComponent,
    IconComponent,
    ImageGalleryComponent,
    StickyActionBarComponent,
    FarmCardComponent,
    FcfaCurrencyPipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    .overlay-btn {
      display: inline-flex;
      height: 44px;
      width: 44px;
      align-items: center;
      justify-content: center;
      border-radius: 9999px;
      background: color-mix(in srgb, var(--color-surface-raised) 88%, transparent);
      color: var(--color-ink);
      backdrop-filter: blur(6px);
      box-shadow: 0 4px 14px rgba(22, 33, 28, 0.18);
    }
  `,
  template: `
    @if (detail.value(); as d) {
      <article class="pb-28 lg:pb-16">
        <div class="lg:mx-auto lg:grid lg:max-w-content-max lg:grid-cols-[minmax(0,1fr)_380px] lg:gap-10 lg:px-18 lg:pt-8">
          <div class="min-w-0">
            <bt-image-gallery
              [photos]="d.listing.photos ?? []"
              [alt]="d.listing.title"
              frameClass="aspect-[4/3] max-h-[62dvh] w-full lg:aspect-[16/10] lg:max-h-none lg:rounded-xl"
            >
              <button type="button" class="overlay-btn lg:invisible" (click)="back()" [attr.aria-label]="'listingDetail.back' | t">
                <bt-icon name="chevron-left" [size]="22" />
              </button>
              <div class="flex gap-2">
                <button type="button" class="overlay-btn" (click)="share(d)" [attr.aria-label]="'listingDetail.share' | t">
                  <bt-icon name="share" [size]="20" />
                </button>
                <button
                  type="button"
                  class="overlay-btn"
                  (click)="toggleFavorite(d)"
                  [attr.aria-pressed]="favorited()"
                  [attr.aria-label]="(favorited() ? 'listingDetail.unfavorite' : 'listingDetail.favorite') | t"
                >
                  <bt-icon name="heart" [size]="20" [filled]="favorited()" [class]="favorited() ? 'text-clay' : ''" />
                </button>
              </div>
            </bt-image-gallery>

            <div class="relative -mt-6 flex flex-col gap-6 rounded-t-xl bg-surface px-4 pt-5 lg:mt-8 lg:rounded-none lg:px-0 lg:pt-0">
              <header class="flex flex-col gap-3">
                <div class="flex flex-wrap items-center gap-2">
                  <bt-status-badge kind="cycle" [status]="d.cycle.status" />
                  <span class="rounded-full bg-clay-soft px-2.5 py-1 text-caption text-clay-ink">{{ d.cycle.type }}</span>
                  @if (d.isOwner && d.listing.status !== 'published') {
                    <bt-status-badge kind="listing" [status]="d.listing.status" />
                  }
                </div>
                <h1 class="text-heading-1">{{ d.listing.title }}</h1>
                @if (d.reviewSummary.count > 0) {
                  <a href="#avis" class="flex items-center gap-2 text-small">
                    <bt-rating [value]="d.reviewSummary.average" [size]="16" />
                    <span class="text-body-strong">{{ d.reviewSummary.average.toFixed(1) }}</span>
                    <span class="text-ink-muted underline">{{ 'listingDetail.verifiedReviews' | t: { count: d.reviewSummary.count } }}</span>
                  </a>
                }
              </header>

              <bt-farm-card
                [farm]="d.farm"
                [signedIn]="auth.isAuthenticated()"
                [returnUrl]="'/marketplace/' + d.listing.id"
                [whatsappMessage]="'listingDetail.whatsappMessage' | t: { title: d.listing.title }"
              />

              <bt-capacity-bar [capacity]="d.cycle.capacity" [availableQuantity]="d.cycle.availableQuantity" />

              @if (d.listing.description; as description) {
                <p class="whitespace-pre-line text-body text-ink">{{ description }}</p>
              }

              <section class="flex flex-col gap-4">
                <h2 class="text-heading-3">{{ 'listingDetail.cycleTimelineTitle' | t }}</h2>
                <bt-cycle-timeline [steps]="d.cycle.timeline" [cancelled]="d.cycle.status === 'cancelled'" />
              </section>

              <section id="avis" class="flex scroll-mt-20 flex-col gap-4">
                <h2 class="text-heading-3">{{ 'listingDetail.reviewsTitle' | t }}</h2>
                @if (d.reviewSummary.count > 0) {
                  <bt-rating-summary [average]="d.reviewSummary.average" [count]="d.reviewSummary.count" [histogram]="d.reviewSummary.histogram" />
                  @for (review of d.topReviews; track review.id) {
                    <bt-review-card [rating]="review.rating" [comment]="review.comment" />
                  }
                } @else {
                  <p class="rounded-lg border border-dashed border-line-strong p-5 text-center text-body text-ink-muted">
                    {{ 'listingDetail.noReviews' | t }}
                  </p>
                }
              </section>

              <bt-notice tone="danger" [title]="'listingDetail.safetyNoticeTitle' | t">{{ 'listingDetail.safetyNoticeBody' | t }}</bt-notice>

              @if (!d.isOwner) {
                <button
                  type="button"
                  (click)="openReport()"
                  class="flex min-h-control-md items-center gap-2 self-start text-body-strong text-alert"
                >
                  <bt-icon name="flag" [size]="18" />
                  {{ 'listingDetail.reportLink' | t }}
                </button>
              }
            </div>
          </div>

          <aside class="hidden lg:block">
            <div class="sticky top-24 flex flex-col gap-4 rounded-lg border border-line bg-surface-card p-6" style="box-shadow: var(--shadow-float)">
              <div class="flex items-baseline gap-1.5">
                <span class="text-heading-2">{{ d.listing.unitPrice | fcfa }}</span>
                @if (d.listing.unit; as unit) {
                  <span class="text-small text-ink-muted">/ {{ unit }}</span>
                }
              </div>
              <bt-capacity-bar [capacity]="d.cycle.capacity" [availableQuantity]="d.cycle.availableQuantity" />
              <ng-container *ngTemplateOutlet="cta; context: { $implicit: d, block: true }" />
              <bt-notice tone="warning">{{ priorityNotice.noticeBodyKey | t }}</bt-notice>
            </div>
          </aside>
        </div>
      </article>

      <bt-sticky-action-bar>
        <div class="flex min-w-0 flex-1 flex-col">
          <span class="text-heading-4">
            {{ d.listing.unitPrice | fcfa }}
            @if (d.listing.unit; as unit) {
              <span class="text-small font-normal text-ink-muted">/ {{ unit }}</span>
            }
          </span>
          <span class="flex items-center gap-1 text-caption text-forest">
            <bt-icon name="lock" [size]="12" />
            <span class="truncate">{{ 'listingDetail.priorityHint' | t }}</span>
          </span>
        </div>
        <ng-container *ngTemplateOutlet="cta; context: { $implicit: d, block: false }" />
      </bt-sticky-action-bar>

      <ng-template #cta let-d let-block="block">
        @if (d.isOwner) {
          <bt-button variant="outline" size="lg" [block]="block" href="/portal">{{ 'listingDetail.ownerManage' | t }}</bt-button>
        } @else if (d.canReserve) {
          <bt-button size="lg" [block]="block" (click)="reserve(d.listing.id)">{{ 'listingDetail.reserveButton' | t }}</bt-button>
        } @else {
          <bt-button size="lg" [block]="block" disabled>
            {{ (d.cycle.availableQuantity === 0 ? 'listingDetail.soldOut' : 'listingDetail.unavailable') | t }}
          </bt-button>
        }
      </ng-template>

      <bt-dialog
        [open]="reportOpen()"
        tone="alert"
        [title]="'listingDetail.reportDialogTitle' | t"
        [primaryLabel]="'listingDetail.reportSubmit' | t"
        primaryVariant="danger"
        [secondaryLabel]="'common.cancel' | t"
        (primaryClick)="submitReport(d.listing.id)"
        (secondaryClick)="reportOpen.set(false)"
        (closed)="reportOpen.set(false)"
      >
        <bt-text-field [label]="'listingDetail.reportReasonLabel' | t" multiline [ngModel]="reportReason()" (ngModelChange)="reportReason.set($event)" />
      </bt-dialog>
    } @else if (detail.isLoading()) {
      <div class="animate-pulse" aria-busy="true" [attr.aria-label]="'listingDetail.loading' | t">
        <div class="aspect-[4/3] w-full bg-surface-sunken lg:mx-auto lg:mt-8 lg:max-w-content-max lg:rounded-xl"></div>
        <div class="flex flex-col gap-3 px-4 pt-6 lg:mx-auto lg:max-w-content-max lg:px-18">
          <div class="h-5 w-24 rounded-full bg-surface-sunken"></div>
          <div class="h-8 w-3/4 rounded-md bg-surface-sunken"></div>
          <div class="h-20 w-full rounded-lg bg-surface-sunken"></div>
        </div>
      </div>
    } @else if (detail.error()) {
      <div class="mx-auto flex max-w-md flex-col items-center gap-4 px-5 py-20 text-center">
        <bt-icon name="search" [size]="36" class="text-ink-muted" />
        <p class="text-body text-ink">{{ 'listingDetail.notFound' | t }}</p>
        <bt-button variant="outline" href="/marketplace">{{ 'listingDetail.backToMarket' | t }}</bt-button>
      </div>
    }
  `
})
export class ListingDetailComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly listingsApi = inject(ListingsApi);
  private readonly reportsApi = inject(ReportsApi);
  private readonly toast = inject(ToastService);
  protected readonly auth = inject(AuthService);

  private readonly id = toSignal(this.route.paramMap.pipe(map((params) => params.get('id') ?? '')), { initialValue: '' });

  protected readonly detail = apiResource<ListingDetail>(() => (this.id() ? this.listingsApi.fullDetailUrl(this.id()) : undefined));

  /** Optimistic heart: follows the server's value on each load, flips
   * instantly on tap, and rolls back if the request fails. */
  protected readonly favorited = linkedSignal(() => this.detail.value()?.favorited ?? false);

  protected readonly priorityNotice = priorityChoiceCopy(false);
  protected readonly reportOpen = signal(false);
  protected readonly reportReason = signal('');

  private readonly currentUrl = computed(() => `/marketplace/${this.id()}`);

  /** Back to wherever the buyer came from inside the app; a visitor who
   * landed here from a shared link goes to the marketplace instead of
   * leaving the site. */
  protected back(): void {
    if (this.router.lastSuccessfulNavigation()?.previousNavigation) {
      history.back();
    } else {
      this.router.navigateByUrl('/marketplace');
    }
  }

  protected async share(d: ListingDetail): Promise<void> {
    const url = `${location.origin}/marketplace/${d.listing.id}`;
    if (navigator.share) {
      try {
        await navigator.share({ title: d.listing.title, url });
      } catch {
        // The user dismissed the share sheet: nothing to do.
      }
      return;
    }
    try {
      await navigator.clipboard.writeText(url);
      this.toast.show({ tone: 'success', title: 'listingDetail.linkCopied' });
    } catch {
      this.toast.show({ tone: 'danger', title: 'listingDetail.linkCopyFailed' });
    }
  }

  protected toggleFavorite(d: ListingDetail): void {
    if (!this.requireSignIn()) {
      return;
    }
    const next = !this.favorited();
    this.favorited.set(next);
    const request = next ? this.listingsApi.favorite(d.listing.id) : this.listingsApi.unfavorite(d.listing.id);
    request.subscribe({
      next: () => {
        if (next) {
          this.toast.show({ tone: 'success', title: 'listingDetail.savedTitle' });
        }
      },
      error: () => {
        this.favorited.set(!next);
        this.toast.show({ tone: 'danger', title: 'listingDetail.favoriteFailed' });
      }
    });
  }

  protected reserve(listingId: string): void {
    if (!this.auth.isAuthenticated()) {
      this.router.navigate(['/login'], { queryParams: { redirectTo: `/checkout/${listingId}` } });
      return;
    }
    this.router.navigate(['/checkout', listingId]);
  }

  protected openReport(): void {
    if (this.requireSignIn()) {
      this.reportOpen.set(true);
    }
  }

  protected submitReport(listingId: string): void {
    this.reportsApi.create({ targetId: listingId, targetType: 'listing', reason: this.reportReason() }).subscribe({
      next: () => {
        this.reportOpen.set(false);
        this.reportReason.set('');
        this.toast.show({ tone: 'success', title: 'listingDetail.reportSentTitle', detail: 'listingDetail.reportSentDetail' });
      },
      error: () => {
        this.toast.show({ tone: 'danger', title: 'listingDetail.reportFailedTitle', detail: 'listingDetail.reportFailedRetry' });
      }
    });
  }

  /** Signed-in actions send a visitor to sign in first and back here
   * afterwards. */
  private requireSignIn(): boolean {
    if (this.auth.isAuthenticated()) {
      return true;
    }
    this.router.navigate(['/login'], { queryParams: { redirectTo: this.currentUrl() } });
    return false;
  }
}
