import { DecimalPipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, booleanAttribute, computed, input, output } from '@angular/core';
import { IconComponent } from '../icon/icon.component';

const STARS = [1, 2, 3, 4, 5] as const;

/** 5 clay stars. Read-only by default (an average rating); set
 * `interactive` and listen to `changed` for a star picker (README:
 * `onChange` makes it a `radiogroup` with 44px targets). */
@Component({
  selector: 'bt-rating',
  imports: [IconComponent, DecimalPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="inline-flex items-center gap-1">
      @if (interactive()) {
        <div role="radiogroup" aria-label="Note" class="flex items-center">
          @for (star of stars; track star) {
            <button
              type="button"
              role="radio"
              [attr.aria-checked]="star <= value()"
              [attr.aria-label]="star + ' sur 5'"
              (click)="changed.emit(star)"
              class="flex items-center justify-center"
              [style.width.px]="size() + 22"
              [style.height.px]="size() + 22"
            >
              <bt-icon name="star" [size]="size()" [class]="star <= value() ? 'text-star' : 'text-star-empty'" />
            </button>
          }
        </div>
      } @else {
        <div class="flex items-center gap-0.5" [attr.aria-label]="value() + ' sur 5'">
          @for (star of stars; track star) {
            <bt-icon name="star" [size]="size()" [class]="star <= displayValue() ? 'text-star' : 'text-star-empty'" />
          }
        </div>
      }
      @if (showValue()) {
        <span class="ml-1 text-small text-ink-muted">{{ value() | number: '1.1-1' }}</span>
      }
      @if (count(); as reviewCount) {
        <span class="text-small text-ink-muted">({{ reviewCount }})</span>
      }
    </div>
  `
})
export class RatingComponent {
  readonly value = input.required<number>();
  readonly interactive = input(false, { transform: booleanAttribute });
  readonly showValue = input(false, { transform: booleanAttribute });
  readonly count = input<number>();
  readonly size = input(18);

  readonly changed = output<number>();

  protected readonly stars = STARS;
  protected readonly displayValue = computed(() => Math.round(this.value()));
}
