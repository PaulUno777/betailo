import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { MarketplaceListing } from '../../../domain/models';
import { FcfaCurrencyPipe } from '../../pipes/fcfa-currency.pipe';
import { StatusBadgeComponent } from '../status-badge/status-badge.component';
import { CapacityBarComponent } from '../capacity-bar/capacity-bar.component';
import { RatingComponent } from '../rating/rating.component';
import { IconComponent } from '../icon/icon.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { LocaleService } from '../../../core/i18n/locale.service';
import { speciesLabelKey } from '../../../domain/reference/species';
import { cycleTypeLabelKey } from '../../../domain/reference/cycle-types';
import { ThumbSrcDirective } from '../thumb-src/thumb-src.directive';

export type ListingCardLayout = 'tile' | 'row';

/** A marketplace result. `tile` is the grid card (photo on top, capacity
 * bar). `row` is the compact phone layout from the design: thumbnail on
 * the left, so a screen shows four or five results instead of one. */
@Component({
  selector: 'bt-listing-card',
  imports: [ThumbSrcDirective, RouterLink, FcfaCurrencyPipe, StatusBadgeComponent, CapacityBarComponent, RatingComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (layout() === 'row') {
      <a
        [routerLink]="['/marketplace', item().listing.id]"
        class="flex gap-3 rounded-lg border border-line bg-surface-card p-3 transition active:scale-[0.99] active:bg-surface-raised"
      >
        <div class="h-24 w-24 shrink-0 overflow-hidden rounded-md bg-surface-sunken">
          @if (item().listing.photos?.[0]; as photo) {
            <img [btThumbSrc]="item().listing.thumbs?.[0]" [photo]="photo" alt="" loading="lazy" decoding="async" class="h-full w-full object-cover" />
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
          <span class="truncate text-caption" [class]="item().availableQuantity === 0 ? 'text-alert' : 'text-ink-muted'">
            {{ eyebrow() }} ·
            @if (item().availableQuantity === 0) {
              {{ 'capacity.complete' | t }}
            } @else {
              {{ 'capacity.remaining' | t: { count: item().availableQuantity } }}
            }
          </span>
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
        <div class="relative aspect-[4/3] overflow-hidden bg-surface-sunken">
          @if (item().listing.photos?.[0]; as photo) {
            <!-- Absolute, so a portrait or square photo is cropped to 4:3
                 instead of stretching the card. -->
            <img
              [btThumbSrc]="item().listing.thumbs?.[0]"
              [photo]="photo"
              sizes="(min-width: 1024px) 30vw, (min-width: 640px) 50vw, 100vw"
              alt=""
              loading="lazy"
              decoding="async"
              class="absolute inset-0 h-full w-full object-cover"
            />
          } @else {
            <!-- No photo yet: say so, instead of an empty grey box. -->
            <div class="photo-pending absolute inset-0 flex items-end p-3">
              <span class="font-mono text-caption text-ink-muted">{{ 'listing.photoPending' | t: { species: (speciesKey() | t) } }}</span>
            </div>
          }
          <div class="absolute left-3 top-3">
            <bt-status-badge kind="cycle" [status]="item().cycleStatus" />
          </div>
          <span
            class="absolute right-3 top-3 inline-flex h-7 items-center gap-1 rounded-full px-2.5 text-caption font-bold"
            [class]="item().farmVerified ? 'bg-forest text-on-forest' : 'bg-surface-raised/90 text-ink'"
            [attr.title]="'listing.trustScore' | t"
          >
            <bt-icon name="shield" [size]="13" />
            <span class="sr-only">{{ 'listing.trustScore' | t }}</span>
            {{ trust() }}
          </span>
          @if (item().favorited) {
            <span class="absolute bottom-3 right-3 flex h-8 w-8 items-center justify-center rounded-full bg-surface-raised/90">
              <bt-icon name="heart" [size]="16" class="text-clay" [filled]="true" [label]="'listing.saved' | t" />
            </span>
          }
        </div>
        <div class="flex flex-1 flex-col gap-2 p-4">
          <span class="text-eyebrow text-clay-ink">{{ eyebrow() }}</span>
          <h3 class="text-heading-4">{{ item().listing.title }}</h3>
          <span class="flex items-center gap-1.5 text-small text-ink-muted">
            <bt-icon name="pin" [size]="14" />
            {{ item().farmName }} · {{ item().farmLocation }}
          </span>
          <bt-rating [value]="item().reviewAverage" [showValue]="true" [count]="item().reviewCount" [size]="14" />
          <bt-capacity-bar [capacity]="item().capacity" [availableQuantity]="item().availableQuantity" />
          <div class="mt-auto flex items-baseline gap-1.5 pt-1">
            <span class="text-amount">{{ item().listing.unitPrice | fcfa }}</span>
            @if (item().listing.unit; as unit) {
              <span class="text-small text-ink-muted">/ {{ unit }}</span>
            }
          </div>
        </div>
      </a>
    }
  `,
  styles: `
    .photo-pending {
      background: repeating-linear-gradient(135deg, var(--color-stone-soft) 0 12px, var(--color-surface-sunken) 12px 24px);
    }
  `
})
export class ListingCardComponent {
  private readonly locale = inject(LocaleService);

  readonly item = input.required<MarketplaceListing>();
  readonly layout = input<ListingCardLayout>('tile');

  protected readonly speciesKey = computed(() => speciesLabelKey(this.item().species));
  protected readonly trust = computed(() => Math.round(this.item().farmTrustScore));

  /** "ENGRAISSEMENT · 10 OCT. → 25 NOV.", "COUVÉE · DÈS LE 05 OCT.":
   * what kind of lot and when, the design's line above the title. */
  protected readonly eyebrow = computed(() => {
    const it = this.item();
    const lang = this.locale.locale();
    const type = this.locale.t(cycleTypeLabelKey(it.cycleType));
    const day = (iso: string) => new Intl.DateTimeFormat(lang, { day: '2-digit', month: 'short' }).format(new Date(iso));
    const starts = new Date(it.startsAt);
    let when: string;
    if (starts.getTime() > Date.now()) {
      when = this.locale.t('listing.from', { date: day(it.startsAt) });
    } else if (it.endsAt) {
      when = `${day(it.startsAt)} → ${day(it.endsAt)}`;
    } else {
      when = this.locale.t('listing.availableNow');
    }
    return `${type} · ${when}`;
  });
}
