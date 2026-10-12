import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { IconComponent } from '../icon/icon.component';
import { IconName } from '../icon/icon-paths';
import { Toast, ToastTone } from '../../../core/notifications/toast.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

const TONE_ICON: Readonly<Record<ToastTone, IconName>> = {
  info: 'shield',
  success: 'check',
  warning: 'alert',
  danger: 'alert'
};

const TONE_ICON_CLASSES: Readonly<Record<ToastTone, string>> = {
  info: 'bg-sky-soft text-sky',
  success: 'bg-forest-soft text-forest',
  warning: 'bg-amber-soft text-amber',
  danger: 'bg-alert-soft text-alert'
};

/** A short confirmation after an action — title in the past participle,
 * one line of detail. Auto-dismiss is handled by `ToastService`, not here. */
@Component({
  selector: 'bt-toast',
  imports: [IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div
      [attr.role]="toast().tone === 'danger' ? 'alert' : 'status'"
      class="flex w-full max-w-sm items-start gap-3 rounded-md bg-surface-card p-4"
      style="box-shadow: var(--shadow-dialog)"
    >
      <span [class]="'flex h-8 w-8 shrink-0 items-center justify-center rounded-full ' + iconClass()">
        <bt-icon [name]="icon()" [size]="16" />
      </span>
      <div class="flex-1">
        <p class="text-body-strong">{{ toast().title | t }}</p>
        @if (toast().detail; as detail) {
          <p class="text-small text-ink-muted">{{ detail | t }}</p>
        }
        @if (toast().action; as action) {
          <button type="button" (click)="action.onClick()" class="mt-1 text-small text-forest underline">
            {{ action.label | t }}
          </button>
        }
      </div>
      <button type="button" [attr.aria-label]="'common.close' | t" (click)="dismissed.emit()" class="-m-2 flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-ink-muted hover:bg-surface-sunken">
        <bt-icon name="x" [size]="16" />
      </button>
    </div>
  `
})
export class ToastComponent {
  readonly toast = input.required<Toast>();
  readonly dismissed = output<void>();

  protected icon(): IconName {
    return TONE_ICON[this.toast().tone];
  }

  protected iconClass(): string {
    return TONE_ICON_CLASSES[this.toast().tone];
  }
}
