import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';
import { Review } from '../../../domain/models';
import { RatingComponent } from '../rating/rating.component';
import { AvatarComponent } from '../avatar/avatar.component';
import { IconComponent } from '../icon/icon.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { LocalDatePipe } from '../../pipes/local-date.pipe';

/** A verified review, per the design: who (public name only) and when,
 * stars, comment, photos, what they liked, the "Avis vérifié · RSV-xxxx"
 * badge, an "Utile · N" button, and the farmer's reply. `canVote` and
 * `canReply` come from the page (signed in, the farm's owner); the API
 * enforces both regardless. */
@Component({
  selector: 'bt-review-card',
  imports: [RatingComponent, AvatarComponent, IconComponent, TranslatePipe, LocalDatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <article class="flex flex-col gap-3 rounded-xl border border-line bg-surface-card p-4 sm:p-5">
      <header class="flex items-start gap-3">
        <bt-avatar [name]="name()" tone="sky" [size]="40" />
        <div class="flex min-w-0 flex-1 flex-col">
          <span class="truncate text-body-strong">{{ name() }}</span>
          <span class="text-caption text-ink-muted">{{ review().createdAt | localDate: 'monthYear' }}</span>
        </div>
        <bt-rating [value]="review().rating" [size]="15" />
      </header>

      @if (review().comment; as text) {
        <p class="whitespace-pre-line text-body text-ink">{{ text }}</p>
      }

      @if (review().photos.length) {
        <div class="no-scrollbar -mx-1 flex gap-2 overflow-x-auto px-1">
          @for (photo of review().photos; track photo) {
            <a [href]="photo" target="_blank" rel="noopener" class="shrink-0">
              <img [src]="photo" [alt]="'review.photoAlt' | t" loading="lazy" class="h-20 w-20 rounded-md object-cover sm:h-24 sm:w-24" />
            </a>
          }
        </div>
      }

      @if (review().tags.length) {
        <div class="flex flex-wrap gap-1.5">
          @for (tag of review().tags; track tag) {
            <span class="rounded-full bg-forest-soft px-3 py-1 text-caption text-forest">{{ 'reviewTags.' + tag | t }}</span>
          }
        </div>
      }

      @if (review().reply; as reply) {
        <div class="rounded-lg bg-surface-sunken p-3">
          <p class="text-small font-semibold text-forest">{{ 'review.farmerReply' | t }}</p>
          <p class="text-body">{{ reply.text }}</p>
        </div>
      }

      <footer class="flex flex-wrap items-center justify-between gap-2">
        @if (review().reservationRef; as ref) {
          <span class="rounded-full bg-clay-soft px-3 py-1 text-caption text-clay-ink">{{ 'review.verified' | t: { id: ref } }}</span>
        } @else {
          <span></span>
        }
        <div class="flex items-center gap-2">
          @if (canReply() && !review().reply) {
            <button type="button" (click)="reply.emit()" class="h-10 rounded-full px-3 text-small font-semibold text-forest">
              {{ 'review.replyAction' | t }}
            </button>
          }
          <button
            type="button"
            [disabled]="!canVote() || review().votedByMe || review().mine"
            [attr.aria-pressed]="review().votedByMe"
            (click)="helpful.emit()"
            class="flex h-10 items-center gap-1.5 rounded-full border px-4 text-small font-semibold transition disabled:cursor-default"
            [class]="review().votedByMe ? 'border-forest bg-forest-soft text-forest' : 'border-line-strong text-ink'"
          >
            <bt-icon name="thumb" [size]="16" />
            {{ 'review.helpful' | t: { count: review().helpfulCount } }}
          </button>
        </div>
      </footer>
    </article>
  `
})
export class ReviewCardComponent {
  readonly review = input.required<Review>();
  readonly canVote = input(false);
  readonly canReply = input(false);
  /** Shown when the author set no display name. Already translated. */
  readonly anonymousName = input('');

  readonly helpful = output<void>();
  readonly reply = output<void>();

  protected readonly name = computed(() => this.review().authorName || this.anonymousName());
}
