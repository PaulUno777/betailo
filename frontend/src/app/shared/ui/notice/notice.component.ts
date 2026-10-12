import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { IconComponent } from '../icon/icon.component';
import { IconName } from '../icon/icon-paths';

export type NoticeTone = 'info' | 'success' | 'warning' | 'danger';

const TONE_CLASSES: Readonly<Record<NoticeTone, string>> = {
  info: 'bg-sky-soft text-sky',
  success: 'bg-forest-soft text-forest',
  warning: 'bg-amber-soft text-amber',
  danger: 'bg-alert-soft text-alert'
};

const TONE_ICON: Readonly<Record<NoticeTone, IconName>> = {
  info: 'shield',
  success: 'check',
  warning: 'alert',
  danger: 'alert'
};

/** An inline message inside the page flow — escrow explanations, the
 * priority rule, anti-scam warnings. The body says what happens next. */
@Component({
  selector: 'bt-notice',
  imports: [IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div [class]="'flex gap-3 rounded-md p-4 ' + toneClass()">
      <bt-icon [name]="toneIcon()" [size]="20" class="mt-0.5 shrink-0" />
      <div class="flex flex-col gap-0.5">
        @if (title(); as heading) {
          <p class="text-body-strong">{{ heading }}</p>
        }
        <p class="text-body"><ng-content /></p>
      </div>
    </div>
  `
})
export class NoticeComponent {
  readonly tone = input<NoticeTone>('info');
  readonly title = input<string>();

  protected toneClass(): string {
    return TONE_CLASSES[this.tone()];
  }

  protected toneIcon(): IconName {
    return TONE_ICON[this.tone()];
  }
}
