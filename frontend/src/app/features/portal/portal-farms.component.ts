import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { Farm } from '../../domain/models';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
import { AvatarComponent } from '../../shared/ui/avatar/avatar.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { FarmProofFormComponent } from '../sell/farm-proof-form.component';
import { SellerOnboardingComponent } from '../sell/seller-onboarding.component';

/** "Mes fermes": every farm the account owns, where each stands with
 * the platform's review, and what its owner can do about it (submit the
 * proof, read why it was refused and submit again). Also where another
 * farm is opened, while the account is under its limit. Which farm the
 * rest of the portal works on is chosen here or from the sidebar card.
 *
 * Status, reasons, the limit and whether another farm is allowed all
 * come from the API (`GET /me`). */
@Component({
  selector: 'bt-portal-farms',
  imports: [
    AvatarComponent,
    ButtonComponent,
    IconComponent,
    NoticeComponent,
    StatusBadgeComponent,
    FarmProofFormComponent,
    SellerOnboardingComponent,
    LocalDatePipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto flex max-w-3xl flex-col gap-6 p-4 sm:p-8">
      <header class="flex flex-wrap items-end justify-between gap-3">
        <div class="flex flex-col gap-1">
          <span class="text-eyebrow text-clay-ink">{{ 'portalFarms.count' | t: { count: seller.farms().length, max: seller.farmLimit() } }}</span>
          <h1 class="text-heading-2">{{ 'portalFarms.title' | t }}</h1>
        </div>
        @if (seller.canAddFarm() && !adding()) {
          <bt-button variant="solid" (click)="adding.set(true)">
            <bt-icon name="plus" [size]="18" />
            {{ 'portalFarms.add' | t }}
          </bt-button>
        }
      </header>

      @if (adding()) {
        <section class="rounded-xl border border-line bg-surface-card p-5" [attr.aria-label]="'portalFarms.add' | t">
          <bt-seller-onboarding (created)="onFarmAdded($event)" />
          <bt-button class="mt-3 block" variant="ghost" block (click)="adding.set(false)">{{ 'common.cancel' | t }}</bt-button>
        </section>
      }

      @if (!seller.loaded()) {
        <div class="h-40 animate-pulse rounded-xl bg-surface-sunken" aria-busy="true"></div>
      } @else {
        <ul class="flex flex-col gap-4">
          @for (farm of seller.farms(); track farm.id) {
            <li class="flex flex-col gap-4 rounded-xl border bg-surface-card p-5" [class]="isCurrent(farm) ? 'border-forest' : 'border-line'">
              <div class="flex items-start gap-3">
                <bt-avatar [name]="farm.name" [size]="48" shape="square" tone="forest" />
                <div class="flex min-w-0 flex-1 flex-col gap-1">
                  <h2 class="wrap-anywhere text-heading-4">{{ farm.name }}</h2>
                  <span class="text-small text-ink-muted">{{ farm.location }}{{ farm.region ? ' · ' + farm.region : '' }}</span>
                </div>
                <bt-status-badge kind="farm" [status]="farm.status" />
              </div>

              @switch (farm.status) {
                @case ('draft') {
                  <bt-notice tone="warning" [title]="'portalFarms.draftTitle' | t">{{ 'portalFarms.draftBody' | t }}</bt-notice>
                }
                @case ('pending') {
                  <bt-notice tone="info" icon="clock" [title]="'portalFarms.pendingTitle' | t">
                    {{ 'portalFarms.pendingBody' | t: { date: (farm.review?.submittedAt ?? '' | localDate: 'dayMonth') } }}
                  </bt-notice>
                }
                @case ('rejected') {
                  <bt-notice tone="danger" [title]="'portalFarms.rejectedTitle' | t">{{ 'portalFarms.rejectedBody' | t }}</bt-notice>
                }
                @case ('suspended') {
                  <bt-notice tone="danger" [title]="'portalFarms.suspendedTitle' | t">
                    {{ farm.review?.rejectionReason || ('portalFarms.suspendedBody' | t) }}
                  </bt-notice>
                }
                @case ('approved') {
                  @if (farm.review?.autoApproved) {
                    <bt-notice tone="info">{{ 'portalFarms.autoApproved' | t }}</bt-notice>
                  }
                }
              }

              @if (farm.review?.canSubmit) {
                @if (submitting() === farm.id) {
                  <bt-farm-proof-form [farm]="farm" (submitted)="onSubmitted($event)" />
                  <bt-button variant="ghost" block (click)="submitting.set(null)">{{ 'common.cancel' | t }}</bt-button>
                } @else {
                  <bt-button variant="primary" block (click)="submitting.set(farm.id)">
                    {{ (farm.status === 'rejected' ? 'portalFarms.resubmit' : 'portalFarms.submit') | t }}
                  </bt-button>
                }
              }

              <div class="flex flex-wrap items-center gap-2 border-t border-line pt-4">
                @if (isCurrent(farm)) {
                  <span class="inline-flex items-center gap-1.5 text-small font-semibold text-forest">
                    <bt-icon name="check" [size]="16" />
                    {{ 'portalFarms.current' | t }}
                  </span>
                  <bt-button class="ml-auto" variant="outline" size="sm" href="/portal/farm">{{ 'portalFarms.edit' | t }}</bt-button>
                } @else {
                  <bt-button class="ml-auto" variant="outline" size="sm" (click)="manage(farm)">{{ 'portalFarms.manage' | t }}</bt-button>
                }
              </div>
            </li>
          }
        </ul>

        @if (!seller.canAddFarm()) {
          <p class="text-small text-ink-muted">{{ 'portalFarms.limitReached' | t: { count: seller.farmLimit() } }}</p>
        }
      }
    </div>
  `
})
export class PortalFarmsComponent {
  protected readonly seller = inject(SellerSpaceService);
  private readonly router = inject(Router);
  private readonly toast = inject(ToastService);

  /** Opening another farm (the two-step flow, inline). */
  protected readonly adding = signal(false);
  /** The farm whose proof form is open. */
  protected readonly submitting = signal<string | null>(null);

  constructor() {
    // Staff may have decided since the last look.
    this.seller.refresh();
  }

  protected isCurrent(farm: Farm): boolean {
    return this.seller.currentFarm()?.id === farm.id;
  }

  protected manage(farm: Farm): void {
    this.seller.select(farm.id);
    this.router.navigateByUrl('/portal');
  }

  protected onFarmAdded(farm: Farm): void {
    this.adding.set(false);
    this.seller.select(farm.id);
    this.seller.refresh();
    this.toast.show({ tone: 'success', title: 'sellerOnboarding.done' });
  }

  /** The API says what became of it: pending, or approved on the spot
   * on a platform that approves without review. */
  protected onSubmitted(farm: Farm): void {
    this.submitting.set(null);
    this.seller.refresh();
    this.toast.show({
      tone: 'success',
      title: farm.status === 'approved' ? 'portalFarms.approvedToast' : 'portalFarms.submittedToast',
      detail: farm.status === 'approved' ? undefined : 'portalFarms.submittedToastDetail'
    });
  }
}
