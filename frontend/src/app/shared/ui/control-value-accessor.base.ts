import { Directive, signal } from '@angular/core';
import { ControlValueAccessor } from '@angular/forms';

/** The `ControlValueAccessor` plumbing every form-field UI component needs
 * is identical; only what `T` is and how it renders differs. Extend this
 * instead of retyping `writeValue`/`registerOnChange`/`registerOnTouched`
 * in each one. */
@Directive()
export abstract class ValueAccessorBase<T> implements ControlValueAccessor {
  protected abstract initialValue(): T;

  protected readonly value = signal<T>(this.initialValue());
  protected readonly disabled = signal(false);

  // eslint-disable-next-line @typescript-eslint/no-empty-function -- overwritten by registerOnChange before Forms ever calls it
  private onChangeFn: (value: T) => void = () => {};
  // eslint-disable-next-line @typescript-eslint/no-empty-function -- overwritten by registerOnTouched before Forms ever calls it
  private onTouchedFn: () => void = () => {};

  writeValue(value: T | null): void {
    this.value.set(value ?? this.initialValue());
  }

  registerOnChange(fn: (value: T) => void): void {
    this.onChangeFn = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouchedFn = fn;
  }

  setDisabledState(isDisabled: boolean): void {
    this.disabled.set(isDisabled);
  }

  protected emit(value: T): void {
    this.value.set(value);
    this.onChangeFn(value);
  }

  protected touch(): void {
    this.onTouchedFn();
  }
}
