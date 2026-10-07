import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { FcfaCurrencyPipe } from '../../pipes/fcfa-currency.pipe';

export interface PriceRangeValue {
  readonly min: number | null;
  readonly max: number | null;
}

/** "Prix unitaire" from the design: two thumbs on one track, bounded by
 * the cheapest and dearest lot the other filters leave. `null` means
 * "no limit on this side", so dragging a thumb back to its end removes
 * that half of the filter. Two native range inputs stacked on one track
 * keep it keyboard- and screen-reader-friendly. */
@Component({
  selector: 'bt-price-range',
  imports: [TranslatePipe, FcfaCurrencyPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-3">
      <div class="relative h-6">
        <div class="absolute inset-x-0 top-1/2 h-1 -translate-y-1/2 rounded-full bg-line"></div>
        <div
          class="absolute top-1/2 h-1 -translate-y-1/2 rounded-full bg-forest"
          [style.left.%]="pct(low())"
          [style.right.%]="100 - pct(high())"
        ></div>
        <input
          type="range"
          [min]="floor()"
          [max]="ceiling()"
          [step]="step()"
          [value]="low()"
          [attr.aria-label]="'priceRange.min' | t"
          (change)="setLow($any($event.target).valueAsNumber)"
          class="range-thumb pointer-events-none absolute inset-0 h-6 w-full appearance-none bg-transparent"
        />
        <input
          type="range"
          [min]="floor()"
          [max]="ceiling()"
          [step]="step()"
          [value]="high()"
          [attr.aria-label]="'priceRange.max' | t"
          (change)="setHigh($any($event.target).valueAsNumber)"
          class="range-thumb pointer-events-none absolute inset-0 h-6 w-full appearance-none bg-transparent"
        />
      </div>
      <div class="flex justify-between font-mono text-small text-ink-muted">
        <span>{{ low() | fcfa }}</span>
        <span>{{ high() | fcfa }}</span>
      </div>
    </div>
  `,
  styles: `
    .range-thumb::-webkit-slider-thumb {
      pointer-events: auto;
      appearance: none;
      height: 22px;
      width: 22px;
      border-radius: 9999px;
      background: var(--color-surface-card);
      border: 2px solid var(--color-forest);
      cursor: pointer;
    }
    .range-thumb::-moz-range-thumb {
      pointer-events: auto;
      height: 18px;
      width: 18px;
      border-radius: 9999px;
      background: var(--color-surface-card);
      border: 2px solid var(--color-forest);
      cursor: pointer;
    }
  `
})
export class PriceRangeComponent {
  /** The cheapest and dearest unit prices available to pick between. */
  readonly bounds = input.required<{ min: number; max: number }>();
  readonly value = input.required<PriceRangeValue>();
  readonly valueChange = output<PriceRangeValue>();

  /** A round step (1, 2 or 5 × 10ⁿ) giving about a hundred stops, with
   * both ends on it, so every thumb position is a round price and both
   * ends are reachable (an end is how a buyer removes that limit). */
  protected readonly step = computed(() => {
    const raw = Math.max((this.bounds().max - this.bounds().min) / 100, 1);
    const magnitude = 10 ** Math.floor(Math.log10(raw));
    return [1, 2, 5, 10].map((m) => m * magnitude).find((s) => s >= raw) ?? 10 * magnitude;
  });
  protected readonly floor = computed(() => Math.floor(this.bounds().min / this.step()) * this.step());
  protected readonly ceiling = computed(() =>
    Math.max(Math.ceil(this.bounds().max / this.step()) * this.step(), this.floor() + this.step())
  );
  protected readonly low = computed(() => this.clamp(this.value().min ?? this.floor()));
  protected readonly high = computed(() => this.clamp(this.value().max ?? this.ceiling()));

  protected pct(v: number): number {
    return ((v - this.floor()) / (this.ceiling() - this.floor())) * 100;
  }

  protected setLow(v: number): void {
    const low = Math.min(v, this.high());
    this.valueChange.emit({ min: low <= this.floor() ? null : low, max: this.value().max });
  }

  protected setHigh(v: number): void {
    const high = Math.max(v, this.low());
    this.valueChange.emit({ min: this.value().min, max: high >= this.ceiling() ? null : high });
  }

  private clamp(v: number): number {
    return Math.min(this.ceiling(), Math.max(this.floor(), v));
  }
}
