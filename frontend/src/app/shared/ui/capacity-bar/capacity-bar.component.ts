import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { capacityState } from '../../../domain/rules/capacity';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

const FILL_CLASSES = { forest: 'bg-forest', clay: 'bg-clay', complete: 'bg-line-strong' } as const;

/** How full a cycle is — forest fill, clay past 90% reserved, muted
 * "Complet" (in alert) at 100%. Shown on every listing and the booking box. */
@Component({
  selector: 'bt-capacity-bar',
  imports: [TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-1.5">
      <div class="h-2 w-full overflow-hidden rounded-full bg-surface-sunken">
        <div [class]="'h-full rounded-full ' + fillClass()" [style.width.%]="state().reservedRatio * 100"></div>
      </div>
      <span [class]="'text-small ' + (state().tone === 'complete' ? 'text-alert' : 'text-ink-muted')">{{
        state().labelKey | t: state().labelParams
      }}</span>
    </div>
  `
})
export class CapacityBarComponent {
  readonly capacity = input.required<number>();
  readonly availableQuantity = input.required<number>();

  protected readonly state = computed(() => capacityState(this.capacity(), this.availableQuantity()));

  protected fillClass(): string {
    return FILL_CLASSES[this.state().tone];
  }
}
