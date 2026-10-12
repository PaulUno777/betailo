import { ChangeDetectionStrategy, Component, ElementRef, effect, input, output, viewChild } from '@angular/core';
import { ButtonComponent, ButtonVariant } from '../button/button.component';
import { IconComponent } from '../icon/icon.component';
import { IconName } from '../icon/icon-paths';

export type DialogTone = 'forest' | 'clay' | 'alert' | 'solid';

const TONE_ICON: Readonly<Record<DialogTone, IconName>> = {
  forest: 'check',
  clay: 'cycle',
  alert: 'alert',
  solid: 'shield'
};

const TONE_CLASSES: Readonly<Record<DialogTone, string>> = {
  forest: 'bg-forest-soft text-forest',
  clay: 'bg-clay-soft text-clay-ink',
  alert: 'bg-alert-soft text-alert',
  solid: 'bg-forest text-on-forest'
};

/** A modal for confirmations and irreversible actions, built on the
 * native `<dialog>` element for its built-in focus trap and Escape
 * handling. The safe action is always first and `outline`, the
 * committing one second — never the other way round. */
@Component({
  selector: 'bt-dialog',
  imports: [ButtonComponent, IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <dialog
      #dialogEl
      [attr.aria-labelledby]="titleId"
      [attr.role]="tone() === 'alert' ? 'alertdialog' : 'dialog'"
      (close)="closed.emit()"
      class="fixed inset-0 m-auto w-full max-w-md rounded-lg bg-surface-card p-6 backdrop:bg-ink/40"
      style="box-shadow: var(--shadow-dialog)"
    >
      <div class="flex flex-col gap-4">
        <span [class]="'flex h-11 w-11 items-center justify-center rounded-full ' + toneClass()">
          <bt-icon [name]="toneIcon()" [size]="22" />
        </span>
        <div class="flex flex-col gap-1">
          <h2 [id]="titleId" class="text-heading-3">{{ title() }}</h2>
          <div class="text-body text-ink-muted"><ng-content /></div>
        </div>
        <div class="mt-2 flex gap-3">
          <bt-button variant="outline" block (click)="secondaryClick.emit()">{{ secondaryLabel() }}</bt-button>
          <bt-button [variant]="primaryVariant()" block (click)="primaryClick.emit()">{{ primaryLabel() }}</bt-button>
        </div>
      </div>
    </dialog>
  `
})
export class DialogComponent {
  private static nextId = 0;
  protected readonly titleId = `bt-dialog-title-${++DialogComponent.nextId}`;

  readonly open = input.required<boolean>();
  readonly tone = input<DialogTone>('forest');
  readonly title = input.required<string>();
  readonly primaryLabel = input.required<string>();
  readonly primaryVariant = input<ButtonVariant>('primary');
  readonly secondaryLabel = input.required<string>();

  readonly primaryClick = output<void>();
  readonly secondaryClick = output<void>();
  readonly closed = output<void>();

  private readonly dialogRef = viewChild.required<ElementRef<HTMLDialogElement>>('dialogEl');

  constructor() {
    effect(() => {
      const element = this.dialogRef().nativeElement;
      if (this.open() && !element.open) {
        element.showModal();
      } else if (!this.open() && element.open) {
        element.close();
      }
    });
  }

  protected toneIcon(): IconName {
    return TONE_ICON[this.tone()];
  }

  protected toneClass(): string {
    return TONE_CLASSES[this.tone()];
  }
}
