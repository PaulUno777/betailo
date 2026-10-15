import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { IconComponent } from '../../../shared/ui/icon/icon.component';

/** The quick actions on an entry: edit, (de)activate, move up/down. */
@Component({
  selector: 'bt-row-actions',
  imports: [IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'inline-flex items-center' },
  template: `
    <button type="button" class="flex h-8 w-8 items-center justify-center rounded-md text-ink-muted hover:bg-surface-sunken hover:text-ink disabled:opacity-30" [disabled]="first() || busy()" (click)="moved.emit(-1)" [attr.aria-label]="'adminReference.moveUp' | t">
      <bt-icon name="chevron-down" [size]="16" class="rotate-180" />
    </button>
    <button type="button" class="flex h-8 w-8 items-center justify-center rounded-md text-ink-muted hover:bg-surface-sunken hover:text-ink disabled:opacity-30" [disabled]="last() || busy()" (click)="moved.emit(1)" [attr.aria-label]="'adminReference.moveDown' | t">
      <bt-icon name="chevron-down" [size]="16" />
    </button>
    <button type="button" class="h-8 rounded-md px-2 text-caption font-semibold text-ink-muted hover:bg-surface-sunken hover:text-ink disabled:opacity-40" [disabled]="busy()" (click)="toggled.emit()">
      {{ (active() ? 'adminReference.deactivate' : 'adminReference.reactivate') | t }}
    </button>
    <button type="button" class="h-8 rounded-md px-2 text-caption font-semibold text-forest hover:bg-forest-soft" (click)="edit.emit()">
      {{ 'adminReference.edit' | t }}
    </button>
  `
})
export class RowActionsComponent {
  readonly active = input(true);
  readonly first = input(false);
  readonly last = input(false);
  readonly busy = input(false);
  readonly edit = output<void>();
  readonly toggled = output<void>();
  readonly moved = output<-1 | 1>();
}
