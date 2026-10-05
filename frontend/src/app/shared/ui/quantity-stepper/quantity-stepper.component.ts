import { ChangeDetectionStrategy, Component, input, model } from '@angular/core';
import { IconComponent } from '../icon/icon.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

/** -/value/+, 44px buttons, clamped to `[min, max]` — `max` is always the
 * listing's `availableQuantity`, never a client-side guess at capacity. */
@Component({
  selector: 'bt-quantity-stepper',
  imports: [IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-1">
      <div class="flex items-center gap-3">
        <button
          type="button"
          [attr.aria-label]="'quantityStepper.decrease' | t"
          (click)="decrement()"
          [disabled]="value() <= min()"
          class="flex h-control-md w-control-md items-center justify-center rounded-sm border border-line-strong disabled:opacity-40"
        >
          <bt-icon name="minus" [size]="16" />
        </button>
        <span class="w-8 text-center text-body-strong" aria-live="polite">{{ value() }}</span>
        <button
          type="button"
          [attr.aria-label]="'quantityStepper.increase' | t"
          (click)="increment()"
          [disabled]="value() >= max()"
          class="flex h-control-md w-control-md items-center justify-center rounded-sm border border-line-strong disabled:opacity-40"
        >
          <bt-icon name="plus" [size]="16" />
        </button>
      </div>
      @if (value() >= max()) {
        <span class="text-small text-ink-muted">{{ 'quantityStepper.maxHint' | t: { count: max() } }}</span>
      }
    </div>
  `
})
export class QuantityStepperComponent {
  readonly value = model.required<number>();
  readonly min = input(1);
  readonly max = input(99);

  protected increment(): void {
    this.value.update((current) => Math.min(this.max(), current + 1));
  }

  protected decrement(): void {
    this.value.update((current) => Math.max(this.min(), current - 1));
  }
}
