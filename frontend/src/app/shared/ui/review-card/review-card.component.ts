import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { RatingComponent } from '../rating/rating.component';
import { BadgeComponent } from '../badge/badge.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

/** A verified review. The design shows author, photos, tags and a farmer
 * reply alongside a mandatory "Avis vérifié · RSV-xxxx" badge — today's
 * `Review` API only returns `{id, rating, comment}`, so every field below
 * beyond `rating`/`comment` is optional and simply doesn't render until
 * the backend exposes it. */
@Component({
  selector: 'bt-review-card',
  imports: [RatingComponent, BadgeComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-3 rounded-lg border border-line bg-surface-card p-5">
      <div class="flex items-center justify-between">
        <bt-rating [value]="rating()" [size]="16" />
        @if (verifiedReservationId(); as reservationId) {
          <bt-badge tone="forest">{{ 'review.verified' | t: { id: reservationId } }}</bt-badge>
        }
      </div>
      @if (authorName(); as name) {
        <span class="text-body-strong">{{ name }}</span>
      }
      @if (comment(); as text) {
        <p class="text-body text-ink">{{ text }}</p>
      }
      @if (farmerReply(); as reply) {
        <div class="mt-1 rounded-md bg-surface-sunken p-3">
          <p class="text-small text-ink-muted">{{ 'review.farmerReply' | t }}</p>
          <p class="text-body">{{ reply }}</p>
        </div>
      }
    </div>
  `
})
export class ReviewCardComponent {
  readonly rating = input.required<number>();
  readonly comment = input<string>();
  readonly authorName = input<string>();
  readonly verifiedReservationId = input<string>();
  readonly farmerReply = input<string>();
}
