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

/** The marketplace grid tile: photo, status + verified badge, title,
 * farm + rating, capacity, price and places left. */
@Component({
  selector: 'bt-listing-card',
  imports: [RouterLink, FcfaCurrencyPipe, StatusBadgeComponent, BadgeComponent, CapacityBarComponent, RatingComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a
      [routerLink]="['/marketplace', item().listing.id]"
      class="flex flex-col overflow-hidden rounded-lg border border-line bg-surface-card transition hover:shadow-[var(--shadow-hover)]"
    >
      <div class="relative aspect-[4/3] bg-surface-sunken">
        @if (item().listing.photos?.[0]; as photo) {
          <img [src]="photo" [alt]="item().listing.title" class="h-full w-full object-cover" />
        }
        <div class="absolute left-3 top-3 flex gap-1.5">
          <bt-status-badge kind="cycle" [status]="item().cycleStatus" />
          @if (item().farmVerified) {
            <bt-badge tone="forest-solid">{{ 'listing.verifiedBadge' | t }}</bt-badge>
            <bt-status-badge kind="trust" [status]="item().farmTrustScore" />
          }
        </div>
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
        <div class="mt-1 flex items-baseline justify-between">
          <span class="text-amount">{{ item().listing.unitPrice | fcfa }}</span>
          @if (item().listing.unit; as unit) {
            <span class="text-small text-ink-muted">/ {{ unit }}</span>
          }
        </div>
      </div>
    </a>
  `
})
export class ListingCardComponent {
  readonly item = input.required<MarketplaceListing>();
}
