import { ChangeDetectionStrategy, Component, booleanAttribute, input, output } from '@angular/core';
import { BadgeComponent } from '../badge/badge.component';

/** A large radio styled as a card — used standalone (e.g. the buyer/farmer
 * choice at sign-up) or in pairs (the pay-now/pay-later priority choice).
 * The parent owns the `radiogroup` wrapper and `name`. */
@Component({
  selector: 'bt-choice-card',
  imports: [BadgeComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <label
      [class]="
        'flex cursor-pointer items-start gap-3 rounded-md border p-4 transition has-[:focus-visible]:shadow-[var(--shadow-focus)] ' +
        (checked() ? 'border-forest bg-forest-soft' : 'border-line-strong bg-surface-card')
      "
    >
      <input type="radio" [name]="name()" [value]="value()" [checked]="checked()" (change)="selected.emit()" class="sr-only" />
      <div class="flex flex-1 flex-col gap-1">
        <span class="text-body-strong">{{ title() }}</span>
        @if (description(); as text) {
          <span class="text-small text-ink-muted">{{ text }}</span>
        }
      </div>
      @if (badge(); as text) {
        <bt-badge [tone]="tone() === 'amber' ? 'amber' : 'neutral'">{{ text }}</bt-badge>
      }
    </label>
  `
})
export class ChoiceCardComponent {
  readonly name = input.required<string>();
  readonly value = input.required<string>();
  readonly title = input.required<string>();
  readonly description = input<string>();
  readonly badge = input<string>();
  readonly tone = input<'default' | 'amber'>('default');
  readonly checked = input(false, { transform: booleanAttribute });

  readonly selected = output<void>();
}
