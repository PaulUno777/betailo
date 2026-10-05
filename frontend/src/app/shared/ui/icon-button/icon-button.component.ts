import { ChangeDetectionStrategy, Component, booleanAttribute, input } from '@angular/core';
import { IconComponent } from '../icon/icon.component';
import { IconName } from '../icon/icon-paths';

/** Square 44px icon-only control. `label` is required — it becomes the
 * button's `aria-label` since there is no visible text. */
@Component({
  selector: 'bt-icon-button',
  imports: [IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <button
      type="button"
      [attr.aria-label]="label()"
      [disabled]="disabled()"
      class="relative inline-flex h-control-md w-11 items-center justify-center rounded-full text-ink hover:bg-surface-sunken transition disabled:opacity-50 disabled:pointer-events-none"
    >
      <bt-icon [name]="icon()" [size]="22" />
      @if (badge()) {
        <span class="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-clay"></span>
      }
    </button>
  `
})
export class IconButtonComponent {
  readonly icon = input.required<IconName>();
  readonly label = input.required<string>();
  readonly badge = input(false, { transform: booleanAttribute });
  readonly disabled = input(false, { transform: booleanAttribute });
}
