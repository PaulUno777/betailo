import { ChangeDetectionStrategy, Component, booleanAttribute, forwardRef, input } from '@angular/core';
import { NG_VALUE_ACCESSOR } from '@angular/forms';
import { ValueAccessorBase } from '../control-value-accessor.base';

export type TextFieldType = 'text' | 'email' | 'password' | 'tel' | 'number';

let nextId = 0;

/** Label above, 52px input, hint or error below. Never placeholder-as-label
 * (README) — `label` is always a real `<label for>`. */
@Component({
  selector: 'bt-text-field',
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [{ provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => TextFieldComponent), multi: true }],
  template: `
    <div class="flex flex-col gap-1.5">
      <label [for]="id" class="text-small text-ink-muted">{{ label() }}</label>
      <div
        [class]="
          'flex items-center gap-2 rounded-sm border bg-surface-card px-4 transition focus-within:border-forest focus-within:shadow-[var(--shadow-focus)] ' +
          (multiline() ? '' : 'h-control-lg') +
          ' ' +
          (error() ? 'border-alert' : 'border-line-strong')
        "
      >
        @if (prefix(); as prefixText) {
          <span class="text-body text-ink-muted">{{ prefixText }}</span>
        }
        @if (multiline()) {
          <textarea
            [id]="id"
            [attr.autocomplete]="autocomplete()"
            [disabled]="disabled()"
            [value]="value()"
            (input)="handleInput($event)"
            (blur)="handleBlur()"
            rows="4"
            class="min-w-0 flex-1 resize-none bg-transparent py-3 text-body text-ink outline-none placeholder:text-ink-muted"
          ></textarea>
        } @else {
          <input
            [id]="id"
            [type]="type()"
            [attr.autocomplete]="autocomplete()"
            [attr.placeholder]="placeholder()"
            [disabled]="disabled()"
            [value]="value()"
            (input)="handleInput($event)"
            (blur)="handleBlur()"
            class="min-w-0 flex-1 bg-transparent text-body text-ink outline-none placeholder:text-ink-muted"
          />
        }
        @if (suffix(); as suffixText) {
          <span class="text-body text-ink-muted">{{ suffixText }}</span>
        }
      </div>
      @if (error(); as message) {
        <span class="text-small text-alert">{{ message }}</span>
      } @else if (hint(); as message) {
        <span class="text-small text-ink-muted">{{ message }}</span>
      }
    </div>
  `
})
export class TextFieldComponent extends ValueAccessorBase<string> {
  readonly id = `bt-text-field-${++nextId}`;

  readonly label = input.required<string>();
  readonly type = input<TextFieldType>('text');
  readonly placeholder = input<string>();
  readonly prefix = input<string>();
  readonly suffix = input<string>();
  readonly multiline = input(false, { transform: booleanAttribute });
  readonly hint = input<string>();
  readonly error = input<string>();
  readonly autocomplete = input<string>();

  protected initialValue(): string {
    return '';
  }

  protected handleInput(event: Event): void {
    this.emit((event.target as HTMLInputElement | HTMLTextAreaElement).value);
  }

  protected handleBlur(): void {
    this.touch();
  }
}
