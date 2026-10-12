import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { RatingComponent } from '../rating/rating.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

const STAR_LEVELS = [5, 4, 3, 2, 1] as const;

/** Average + the 5★→1★ distribution, from `GET /farms/{id}/reviews`'
 * `summary`. The design also shows four scored criteria (santé,
 * conformité, communication, délais) — the API doesn't return per-criteria
 * scores today, so this only renders what it actually has. */
@Component({
  selector: 'bt-rating-summary',
  imports: [RatingComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-4 rounded-lg bg-surface-card p-6">
      <div class="flex items-center gap-4">
        <span class="text-number-xl">{{ average().toFixed(1) }}</span>
        <div class="flex flex-col gap-1">
          <bt-rating [value]="average()" [size]="16" />
          <span class="text-small text-ink-muted">{{ 'review.count' | t: { count: count() } }}</span>
        </div>
      </div>
      <div class="flex flex-col gap-1.5">
        @for (level of levels; track level) {
          <div class="flex items-center gap-2">
            <span class="w-4 text-small text-ink-muted">{{ level }}</span>
            <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-surface-sunken">
              <div class="h-full rounded-full bg-clay" [style.width.%]="ratioFor(level) * 100"></div>
            </div>
          </div>
        }
      </div>
    </div>
  `
})
export class RatingSummaryComponent {
  readonly average = input.required<number>();
  readonly count = input.required<number>();
  readonly histogram = input.required<Partial<Record<string, number>>>();

  protected readonly levels = STAR_LEVELS;

  private readonly maxCount = computed(() => Math.max(1, ...Object.values(this.histogram()).map((n) => n ?? 0)));

  protected ratioFor(level: number): number {
    return (this.histogram()[String(level)] ?? 0) / this.maxCount();
  }
}
