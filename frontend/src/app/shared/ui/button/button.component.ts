import { NgTemplateOutlet } from '@angular/common';
import { ChangeDetectionStrategy, Component, booleanAttribute, computed, input } from '@angular/core';
import { RouterLink } from '@angular/router';

export type ButtonVariant = 'primary' | 'solid' | 'outline' | 'ghost' | 'danger';
export type ButtonSize = 'sm' | 'md' | 'lg';

const VARIANT_CLASSES: Readonly<Record<ButtonVariant, string>> = {
  primary: 'bg-clay text-on-clay hover:brightness-95',
  solid: 'bg-forest text-on-forest hover:brightness-110',
  outline: 'bg-transparent text-ink border border-line-strong hover:bg-surface-sunken',
  ghost: 'bg-transparent text-forest hover:bg-forest-soft',
  danger: 'bg-alert text-on-alert hover:brightness-110'
};

const SIZE_CLASSES: Readonly<Record<ButtonSize, string>> = {
  sm: 'h-control-sm px-4 text-small',
  md: 'h-control-md px-5 text-body-strong',
  lg: 'h-control-xl px-7 text-body-strong'
};

/** Pill-shaped action button. Per the design system: at most one `primary`
 * per view, `danger` always sits behind a confirmation `Dialog`. */
@Component({
  selector: 'bt-button',
  imports: [RouterLink, NgTemplateOutlet],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <ng-template #content><ng-content /></ng-template>
    @if (href(); as link) {
      <a
        [routerLink]="link"
        [class]="classes()"
        [attr.aria-disabled]="disabled() || null"
      >
        <ng-container [ngTemplateOutlet]="content" />
      </a>
    } @else {
      <button
        [type]="type()"
        [class]="classes()"
        [disabled]="disabled()"
      >
        <ng-container [ngTemplateOutlet]="content" />
      </button>
    }
  `
})
export class ButtonComponent {
  readonly variant = input<ButtonVariant>('primary');
  readonly size = input<ButtonSize>('md');
  readonly block = input(false, { transform: booleanAttribute });
  readonly disabled = input(false, { transform: booleanAttribute });
  readonly href = input<string>();
  readonly type = input<'button' | 'submit'>('button');

  protected readonly classes = computed(() =>
    [
      'inline-flex items-center justify-center gap-2 rounded-full transition disabled:opacity-50 disabled:pointer-events-none',
      VARIANT_CLASSES[this.variant()],
      SIZE_CLASSES[this.size()],
      this.block() ? 'w-full' : ''
    ]
      .filter(Boolean)
      .join(' ')
  );
}
