import { ChangeDetectionStrategy, Component, booleanAttribute, input, output } from '@angular/core';
import { BadgeComponent } from '../badge/badge.component';

/** A large radio styled as a card — used standalone (e.g. the buyer/farmer
 * choice at sign-up), in pairs (the pay-now/pay-later priority choice) or
 * as a list (payment methods). A visible radio dot leads; projected
 * `[leading]` content (a payment brand mark) sits between it and the
 * title. The parent owns the `radiogroup` wrapper and `name`. */
@Component({
  selector: 'bt-choice-card',
  imports: [BadgeComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <label
      [class]="
        'flex min-h-control-xl cursor-pointer items-center gap-3 rounded-md border-2 p-4 transition has-[:focus-visible]:shadow-[var(--shadow-focus)] ' +
        (checked() ? 'border-forest bg-forest-soft' : 'border-line bg-surface-card')
      "
    >
      <input type="radio" [name]="name()" [value]="value()" [checked]="checked()" (change)="selected.emit()" class="sr-only" />
      <span
        aria-hidden="true"
        class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full border-2"
        [class]="checked() ? 'border-forest' : 'border-line-strong'"
      >
        @if (checked()) {
          <span class="h-2.5 w-2.5 rounded-full bg-forest"></span>
        }
      </span>
      <ng-content select="[leading]" />
      <div class="flex flex-1 flex-col gap-0.5">
        <span class="text-body-strong">{{ title() }}</span>
        @if (description(); as text) {
          <span class="text-small text-ink-muted">{{ text }}</span>
        }
      </div>
      @if (badge(); as text) {
        <bt-badge [tone]="tone() === 'amber' ? 'amber' : tone() === 'forest' ? 'forest-solid' : 'neutral'">{{ text }}</bt-badge>
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
  readonly tone = input<'default' | 'amber' | 'forest'>('default');
  readonly checked = input(false, { transform: booleanAttribute });

  readonly selected = output<void>();
}
