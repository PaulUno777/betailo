import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { DomSanitizer } from '@angular/platform-browser';
import { inject } from '@angular/core';
import { ICON_PATHS, IconName } from './icon-paths';

/** 24px stroke icon, `currentColor` — only `star` renders filled. Sizes
 * per the README: 14 in chips, 18 in buttons, 20-22 in navigation. */
@Component({
  selector: 'bt-icon',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <svg
      [attr.width]="size()"
      [attr.height]="size()"
      viewBox="0 0 24 24"
      [attr.fill]="isStar() ? 'currentColor' : 'none'"
      [attr.stroke]="isStar() ? 'none' : 'currentColor'"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      [attr.role]="label() ? 'img' : null"
      [attr.aria-label]="label() ?? null"
      [attr.aria-hidden]="label() ? null : 'true'"
      [innerHTML]="markup()"
    ></svg>
  `,
  host: { class: 'inline-flex shrink-0' }
})
export class IconComponent {
  private readonly sanitizer = inject(DomSanitizer);

  readonly name = input.required<IconName>();
  readonly size = input(20);
  readonly label = input<string>();

  protected readonly isStar = computed(() => this.name() === 'star');
  protected readonly markup = computed(() => this.sanitizer.bypassSecurityTrustHtml(ICON_PATHS[this.name()]));
}
