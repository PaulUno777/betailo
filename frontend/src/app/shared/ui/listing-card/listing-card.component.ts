import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { MarketplaceListing } from '../../../domain/models';
import { FcfaCurrencyPipe } from '../../pipes/fcfa-currency.pipe';
import { StatusBadgeComponent } from '../status-badge/status-badge.component';
import { BadgeComponent } from '../badge/badge.component';
import { CapacityBarComponent } from '../capacity-bar/capacity-bar.component';
import { RatingComponent } from '../rating/rating.component';
import { IconComponent } from '../icon/icon.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

export type ListingCardLayout = 'tile' | 'row';

/** A marketplace result. `tile` is the grid card (photo on top, capacity
 * bar). `row` is the compact phone layout from the design: thumbnail on
 * the left, so a screen shows four or five results instead of one. */
@Component({
  selector: 'bt-listing-card',
  imports: [RouterLink, FcfaCurrencyPipe, StatusBadgeComponent, BadgeComponent, CapacityBarComponent, RatingComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (layout() === 'row') {
      <a
        [routerLink]="['/marketplace', item().listing.id]"
        class="flex gap-3 rounded-lg border border-line bg-surface-card p-3 transition active:scale-[0.99]"
      >
        <div class="h-24 w-24 shrink-0 overflow-hidden rounded-md bg-surface-sunken">
          @if (item().listing.photos?.[0]; as photo) {
            <img [src]="photo" [alt]="item().listing.title" loading="lazy" decoding="async" class="h-full w-full object-cover" />
          }
        </div>
        <div class="flex min-w-0 flex-1 flex-col gap-1">
          <div class="flex items-center gap-1.5">
            <bt-status-badge kind="cycle" [status]="item().cycleStatus" />
            @if (item().farmVerified) {
              <bt-icon name="shield" [size]="16" class="text-forest" [label]="'listing.verifiedBadge' | t" />
            }
            @if (item().favorited) {
              <bt-icon name="heart" [size]="14" class="ml-auto text-clay" [filled]="true" [label]="'listing.saved' | t" />
            }
          </div>
          <h3 class="line-clamp-2 text-body-strong leading-snug">{{ item().listing.title }}</h3>
          <span class="truncate text-small text-ink-muted">{{ item().farmName }} · {{ item().farmLocation }}</span>
          <div class="mt-auto flex items-end justify-between gap-2">
            <span class="text-body-strong">
              {{ item().listing.unitPrice | fcfa }}
              @if (item().listing.unit; as unit) {
                <span class="text-caption font-normal text-ink-muted">/{{ unit }}</span>
              }
            </span>
            @if (item().reviewCount > 0) {
              <span class="flex shrink-0 items-center gap-1 text-small">
                <bt-icon name="star" [size]="13" class="text-star" />
                <span class="font-semibold">{{ item().reviewAverage.toFixed(1) }}</span>
                <span class="text-ink-muted">({{ item().reviewCount }})</span>
              </span>
            }
          </div>
        </div>
      </a>
    } @else {
      <a
        [routerLink]="['/marketplace', item().listing.id]"
        class="flex h-full flex-col overflow-hidden rounded-lg border border-line bg-surface-card transition hover:shadow-[var(--shadow-hover)]"
      >
        <div class="relative aspect-[4/3] bg-surface-sunken">
          @if (item().listing.photos?.[0]; as photo) {
            <img [src]="photo" [alt]="item().listing.title" loading="lazy" decoding="async" class="h-full w-full object-cover" />
          }
          <div class="absolute left-3 top-3 flex gap-1.5">
            <bt-status-badge kind="cycle" [status]="item().cycleStatus" />
            @if (item().farmVerified) {
              <bt-badge tone="forest-solid">{{ 'listing.verifiedBadge' | t }}</bt-badge>
              <bt-status-badge kind="trust" [status]="item().farmTrustScore" />
            }
          </div>
          @if (item().favorited) {
            <span class="absolute right-3 top-3 flex h-8 w-8 items-center justify-center rounded-full bg-surface-raised/90">
              <bt-icon name="heart" [size]="16" class="text-clay" [filled]="true" [label]="'listing.saved' | t" />
            </span>
          }
        </div>
        <div class="flex flex-1 flex-col gap-2 p-4">
          <span class="text-eyebrow text-clay-ink">{{ item().cycleType }} · {{ item().species }}</span>
          <h3 class="text-heading-4">{{ item().listing.title }}</h3>
          <span class="flex items-center gap-1.5 text-small text-ink-muted">
            <bt-icon name="pin" [size]="14" />
            {{ item().farmName }} · {{ item().farmLocation }}
          </span>
          <bt-rating [value]="item().reviewAverage" [showValue]="true" [count]="item().reviewCount" [size]="14" />
          <bt-capacity-bar [capacity]="item().capacity" [availableQuantity]="item().availableQuantity" />
          <div class="mt-auto flex items-baseline justify-between pt-1">
            <span class="text-amount">{{ item().listing.unitPrice | fcfa }}</span>
            @if (item().listing.unit; as unit) {
              <span class="text-small text-ink-muted">/ {{ unit }}</span>
            }
          </div>
        </div>
      </a>
    }
  `
})
export class ListingCardComponent {
  readonly item = input.required<MarketplaceListing>();
  readonly layout = input<ListingCardLayout>('tile');
}
