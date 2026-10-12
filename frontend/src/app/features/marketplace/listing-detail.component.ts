import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';
import { Cycle, Farm, FarmReviews, Listing } from '../../domain/models';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { CyclesApi } from '../../data-access/cycles/cycles.api';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { ReportsApi } from '../../data-access/reports/reports.api';
import { AuthService } from '../../core/auth/auth.service';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { priorityChoiceCopy } from '../../domain/rules/priority-choice';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { CycleTimelineComponent } from '../../shared/ui/cycle-timeline/cycle-timeline.component';
import { CapacityBarComponent } from '../../shared/ui/capacity-bar/capacity-bar.component';
import { RatingSummaryComponent } from '../../shared/ui/rating-summary/rating-summary.component';
import { ReviewCardComponent } from '../../shared/ui/review-card/review-card.component';
import { QuantityStepperComponent } from '../../shared/ui/quantity-stepper/quantity-stepper.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { AvatarComponent } from '../../shared/ui/avatar/avatar.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { FormsModule } from '@angular/forms';
import { apiResource } from '../../core/http/api-resource';

@Component({
  selector: 'bt-listing-detail',
  imports: [
    RouterLink,
    FormsModule,
    StatusBadgeComponent,
    CycleTimelineComponent,
    CapacityBarComponent,
    RatingSummaryComponent,
    ReviewCardComponent,
    QuantityStepperComponent,
    ButtonComponent,
    NoticeComponent,
    AvatarComponent,
    DialogComponent,
    TextFieldComponent,
    FcfaCurrencyPipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (listing.value(); as item) {
      <div class="mx-auto max-w-content-max px-5 py-8 sm:px-18">
        <nav class="mb-4 text-small text-ink-muted">
          <a routerLink="/marketplace" class="hover:text-forest">{{ 'listingDetail.breadcrumbMarketplace' | t }}</a> · {{ item.title }}
        </nav>

        <div class="grid gap-8 lg:grid-cols-[1fr_360px]">
          <div class="flex flex-col gap-8">
            <div class="grid grid-cols-2 gap-2 overflow-hidden rounded-xl">
              @for (photo of shownPhotos(item.photos); track photo; let last = $last) {
                <div class="relative aspect-[4/3] w-full">
                  <img [src]="photo" [alt]="item.title" class="h-full w-full object-cover" />
                  @if (last && extraPhotoCount(item.photos) > 0) {
                    <div class="absolute inset-0 flex items-center justify-center bg-ink/50 text-heading-4 text-on-forest-deep">
                      +{{ extraPhotoCount(item.photos) }}
                    </div>
                  }
                </div>
              }
              @if (!item.photos?.length) {
                <div class="col-span-2 aspect-[16/9] w-full bg-surface-sunken"></div>
              }
            </div>

            <div class="flex flex-col gap-2">
              <div class="flex items-center gap-2">
                <bt-status-badge kind="listing" [status]="item.status" />
                @if (cycle.value(); as c) {
                  <bt-status-badge kind="cycle" [status]="c.status" />
                }
              </div>
              <h1 class="text-heading-1">{{ item.title }}</h1>
              @if (item.description; as description) {
                <p class="text-body text-ink-muted">{{ description }}</p>
              }
            </div>

            @if (cycle.value(); as c) {
              <div class="rounded-lg border border-line bg-surface-card p-6">
                <h2 class="mb-4 text-heading-3">{{ 'listingDetail.cycleTimelineTitle' | t }}</h2>
                <bt-cycle-timeline [status]="c.status" />
              </div>
            }

            @if (farm.value(); as f) {
              <div class="flex items-center gap-4 rounded-lg border border-line bg-surface-card p-6">
                <bt-avatar [name]="f.name" shape="square" tone="forest" [size]="48" />
                <div class="flex-1">
                  <p class="text-body-strong">{{ f.name }}</p>
                  <p class="text-small text-ink-muted">{{ f.location }}</p>
                </div>
                <bt-button variant="outline" size="sm" [href]="'/marketplace?farmId=' + f.id">{{ 'listingDetail.viewFarm' | t }}</bt-button>
              </div>
            }

            @if (reviews.value(); as r) {
              <div class="flex flex-col gap-4">
                <h2 class="text-heading-3">{{ 'listingDetail.reviewsTitle' | t }}</h2>
                <bt-rating-summary [average]="r.summary.average" [count]="r.summary.count" [histogram]="r.summary.histogram" />
                @for (review of r.reviews.slice(0, 3); track review.id) {
                  <bt-review-card [rating]="review.rating" [comment]="review.comment" />
                }
              </div>
            }
          </div>

          <aside class="flex flex-col gap-4">
            <div class="sticky top-20 flex flex-col gap-4 rounded-lg border border-line bg-surface-card p-6" style="box-shadow: var(--shadow-float)">
              <div class="flex items-baseline justify-between">
                <span class="text-amount text-heading-4">{{ item.unitPrice | fcfa }}</span>
                @if (item.unit; as unit) {
                  <span class="text-small text-ink-muted">/ {{ unit }}</span>
                }
              </div>
              @if (cycle.value(); as c) {
                <bt-capacity-bar [capacity]="c.capacity" [availableQuantity]="c.availableQuantity" />
                <bt-quantity-stepper [(value)]="quantity" [max]="c.availableQuantity" />
              }
              <div class="flex items-center justify-between border-t border-line pt-3 text-body-strong">
                <span>{{ 'listingDetail.subtotalLabel' | t }}</span>
                <span class="text-amount">{{ subtotal() | fcfa }}</span>
              </div>
              <p class="text-small text-ink-muted">{{ 'listingDetail.feeNote' | t }}</p>
              <bt-button size="lg" block (click)="reserve()">{{ 'listingDetail.reserveButton' | t }}</bt-button>
              <bt-notice tone="warning">{{ priorityNotice.noticeBodyKey | t }}</bt-notice>
              <bt-notice tone="danger" [title]="'listingDetail.safetyNoticeTitle' | t">{{ 'listingDetail.safetyNoticeBody' | t }}</bt-notice>
              <button type="button" (click)="reportOpen.set(true)" class="text-small text-alert underline">
                {{ 'listingDetail.reportLink' | t }}
              </button>
            </div>
          </aside>
        </div>
      </div>

      <bt-dialog
        [open]="reportOpen()"
        tone="alert"
        [title]="'listingDetail.reportDialogTitle' | t"
        [primaryLabel]="'listingDetail.reportSubmit' | t"
        primaryVariant="danger"
        [secondaryLabel]="'common.cancel' | t"
        (primaryClick)="submitReport(item.id)"
        (secondaryClick)="reportOpen.set(false)"
        (closed)="reportOpen.set(false)"
      >
        <bt-text-field [label]="'listingDetail.reportReasonLabel' | t" multiline [ngModel]="reportReason()" (ngModelChange)="reportReason.set($event)" />
      </bt-dialog>
    } @else if (listing.isLoading()) {
      <p class="mx-auto max-w-content-max px-5 py-16 text-body text-ink-muted sm:px-18">{{ 'listingDetail.loading' | t }}</p>
    } @else if (listing.error()) {
      <p class="mx-auto max-w-content-max px-5 py-16 text-body text-alert sm:px-18">{{ 'listingDetail.notFound' | t }}</p>
    }
  `
})
export class ListingDetailComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly listingsApi = inject(ListingsApi);
  private readonly cyclesApi = inject(CyclesApi);
  private readonly farmsApi = inject(FarmsApi);
  private readonly reportsApi = inject(ReportsApi);
  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);

  private readonly id = toSignal(this.route.paramMap.pipe(map((params) => params.get('id') ?? '')), { initialValue: '' });

  protected readonly listing = apiResource<Listing>(() => (this.id() ? this.listingsApi.detailUrl(this.id()) : undefined));

  protected readonly cycle = apiResource<Cycle>(() => {
    const cycleId = this.listing.value()?.cycleId;
    return cycleId ? this.cyclesApi.detailUrl(cycleId) : undefined;
  });

  protected readonly farm = apiResource<Farm>(() => {
    const farmId = this.cycle.value()?.farmId;
    return farmId ? this.farmsApi.detailUrl(farmId) : undefined;
  });

  protected readonly reviews = apiResource<FarmReviews>(() => {
    const farmId = this.cycle.value()?.farmId;
    return farmId ? this.farmsApi.reviewsUrl(farmId) : undefined;
  });

  protected shownPhotos(photos: string[] | undefined): string[] {
    return (photos ?? []).slice(0, 4);
  }

  protected extraPhotoCount(photos: string[] | undefined): number {
    return Math.max(0, (photos?.length ?? 0) - 4);
  }

  protected readonly quantity = signal(1);
  protected readonly subtotal = computed(() => this.quantity() * (this.listing.value()?.unitPrice ?? 0));
  protected readonly priorityNotice = priorityChoiceCopy(false);

  protected readonly reportOpen = signal(false);
  protected readonly reportReason = signal('');

  protected reserve(): void {
    const listingId = this.id();
    if (!this.auth.isAuthenticated()) {
      this.router.navigate(['/login'], { queryParams: { redirectTo: `/checkout/${listingId}` } });
      return;
    }
    this.router.navigate(['/checkout', listingId], { queryParams: { quantity: this.quantity() } });
  }

  protected submitReport(listingId: string): void {
    if (!this.auth.isAuthenticated()) {
      this.reportOpen.set(false);
      this.router.navigate(['/login']);
      return;
    }
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
}
