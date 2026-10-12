import { ChangeDetectionStrategy, Component, DestroyRef, computed, inject, input, signal } from '@angular/core';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

interface CountdownLabel { key: string; params: Record<string, string | number> }

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;

/** "dans 47 h", "dans 1 h 05", "dans 12 min": time left until a server
 * timestamp (an auto-release, a payment deadline). Ticks once a minute;
 * the deadline itself is the API's, this only counts down to it. */
@Component({
  selector: 'bt-countdown',
  imports: [TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<time [attr.datetime]="until()">{{ label().key | t: label().params }}</time>`
})
export class CountdownComponent {
  readonly until = input.required<string>();

  private readonly now = signal(Date.now());

  constructor() {
    const id = setInterval(() => this.now.set(Date.now()), MINUTE);
    inject(DestroyRef).onDestroy(() => clearInterval(id));
  }

  protected readonly label = computed((): CountdownLabel => {
    const left = new Date(this.until()).getTime() - this.now();
    if (left <= 0) return { key: 'countdown.now', params: {} };
    if (left >= 2 * HOUR) return { key: 'countdown.hours', params: { h: Math.floor(left / HOUR) } };
    if (left >= HOUR) {
      const m = Math.floor((left % HOUR) / MINUTE);
      return { key: 'countdown.hourMinutes', params: { h: 1, m: m.toString().padStart(2, '0') } };
    }
    return { key: 'countdown.minutes', params: { m: Math.max(1, Math.floor(left / MINUTE)) } };
  });
}
