import { ChangeDetectionStrategy, Component, forwardRef, input } from '@angular/core';
import { NG_VALUE_ACCESSOR } from '@angular/forms';
import { IconComponent } from '../icon/icon.component';
import { ValueAccessorBase } from '../control-value-accessor.base';

export interface SelectOption {
  readonly value: string;
  readonly label: string;
}

let nextId = 0;

/** A native `<select>` with the field frame. For 4+ options — below that,
 * prefer `ChoiceCard` or a segmented control. */
@Component({
  selector: 'bt-select-field',
  imports: [IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [{ provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => SelectFieldComponent), multi: true }],
  template: `
    <div class="flex flex-col gap-1.5">
      <label [for]="id" class="text-small text-ink-muted">{{ label() }}</label>
      <div class="relative flex items-center rounded-sm border border-line-strong bg-surface-card h-control-lg transition focus-within:border-forest focus-within:shadow-[var(--shadow-focus)]">
        <select
          [id]="id"
          [disabled]="disabled()"
          [value]="value()"
          (change)="handleChange($event)"
          (blur)="touch()"
          class="w-full appearance-none bg-transparent px-4 pr-10 text-body text-ink outline-none"
        >
          @if (placeholder(); as text) {
            <option value="">{{ text }}</option>
          }
          @for (option of options(); track option.value) {
            <option [value]="option.value">{{ option.label }}</option>
          }
        </select>
        <bt-icon name="chevron-down" [size]="18" class="pointer-events-none absolute right-3 text-ink-muted" />
      </div>
    </div>
  `
})
export class SelectFieldComponent extends ValueAccessorBase<string> {
  readonly id = `bt-select-field-${++nextId}`;

  readonly label = input.required<string>();
  readonly options = input.required<readonly SelectOption[]>();
  readonly placeholder = input<string>();

  protected initialValue(): string {
    return '';
  }

  protected handleChange(event: Event): void {
    this.emit((event.target as HTMLSelectElement).value);
  }
}
