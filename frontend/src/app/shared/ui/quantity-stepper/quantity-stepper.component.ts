import { ChangeDetectionStrategy, Component, effect, input, model, untracked } from '@angular/core';
import { IconComponent } from '../icon/icon.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

/** [−] [ 30 ] [+], full width with a typeable number in the middle (per
 * the design): 30 chicks shouldn't take 29 taps. The number pad opens on
 * phones, and whatever is typed is clamped to `[min, max]` when the
 * field loses focus. The bounds come from the API (the cycle's min/max
 * and what's left), never a client-side guess at capacity. */
@Component({
  selector: 'bt-quantity-stepper',
  imports: [IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-1">
      <div class="flex items-stretch gap-2">
        <button
          type="button"
          [attr.aria-label]="'quantityStepper.decrease' | t"
          (click)="set(value() - 1)"
          [disabled]="value() <= min()"
          class="flex h-14 w-14 shrink-0 items-center justify-center rounded-md border border-line-strong bg-surface-card transition active:scale-95 disabled:opacity-40"
        >
          <bt-icon name="minus" [size]="20" />
        </button>
        <input
          type="text"
          inputmode="numeric"
          pattern="[0-9]*"
          [attr.aria-label]="'quantityStepper.label' | t"
          [value]="value()"
          (input)="onType($event)"
          (blur)="onBlur($event)"
          (keydown.enter)="onBlur($event)"
          class="h-14 min-w-0 flex-1 rounded-md border border-line-strong bg-surface-card text-center text-heading-2 text-ink outline-none focus:border-forest"
        />
        <button
          type="button"
          [attr.aria-label]="'quantityStepper.increase' | t"
          (click)="set(value() + 1)"
          [disabled]="value() >= max()"
          class="flex h-14 w-14 shrink-0 items-center justify-center rounded-md border border-line-strong bg-surface-card transition active:scale-95 disabled:opacity-40"
        >
          <bt-icon name="plus" [size]="20" />
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

  constructor() {
    // Bounds can arrive or tighten after the value was set (a quote
    // loads, places sell out): keep the value inside them.
    effect(() => {
      const lo = this.min();
      const hi = Math.max(lo, this.max());
      const v = untracked(this.value);
      if (v < lo || v > hi) this.value.set(Math.min(hi, Math.max(lo, v)));
    });
  }

  protected set(n: number): void {
    this.value.set(this.clamp(n));
  }

  /** While typing, accept any whole number so the field doesn't fight
   * the keyboard; clamping waits for blur. */
  protected onType(event: Event): void {
    const n = parseInt((event.target as HTMLInputElement).value.replace(/\D/g, ''), 10);
    if (!Number.isNaN(n)) this.value.set(n);
  }

  protected onBlur(event: Event): void {
    const input = event.target as HTMLInputElement;
    const n = this.clamp(parseInt(input.value.replace(/\D/g, ''), 10) || this.min());
    this.value.set(n);
    input.value = String(n);
  }

  private clamp(n: number): number {
    return Math.min(Math.max(this.min(), this.max()), Math.max(this.min(), n));
  }
}
