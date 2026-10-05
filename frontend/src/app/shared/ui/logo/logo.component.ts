import { ChangeDetectionStrategy, Component, input } from '@angular/core';

export type LogoTone = 'brand' | 'light' | 'mono';
export type LogoVariant = 'full' | 'mark';

/** The Betailo mark (two cycle arrows around a clay leaf) with its
 * wordmark. Never rotate the mark — the arrows show the cycle direction —
 * and never recolour the leaf other than `clay` or, in `mono`, the ink. */
@Component({
  selector: 'bt-logo',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <span class="inline-flex items-center gap-2" [class]="wordmarkColorClass()">
      <svg [attr.width]="markSize()" [attr.height]="markSize()" viewBox="0 0 32 32" fill="none">
        <path
          d="M27 16a11 11 0 0 1-18.7 7.8L5 20.5"
          [attr.stroke]="arrowColor()"
          stroke-width="2.4"
          stroke-linecap="round"
        />
        <path
          d="M5 16A11 11 0 0 1 23.7 8.2L27 11.5"
          [attr.stroke]="arrowColor()"
          stroke-width="2.4"
          stroke-linecap="round"
        />
        <path d="M16 10c3.5 0 6 2.7 6 6.2-3.6 0-6.6-1-6-6.2z" [attr.fill]="leafColor()" />
        <path d="M16 10c-3.5 0-6 2.7-6 6.2 3.6 0 6.6-1 6-6.2z" [attr.fill]="leafColor()" />
      </svg>
      @if (variant() === 'full') {
        <span class="text-heading-4">Betailo</span>
      }
    </span>
  `
})
export class LogoComponent {
  readonly tone = input<LogoTone>('brand');
  readonly variant = input<LogoVariant>('full');
  readonly markSize = input(28);

  protected arrowColor(): string {
    switch (this.tone()) {
      case 'light':
        return 'var(--color-on-forest-deep)';
      case 'mono':
        return 'currentColor';
      default:
        return 'var(--color-forest)';
    }
  }

  protected leafColor(): string {
    return this.tone() === 'mono' ? 'currentColor' : 'var(--color-clay)';
  }

  protected wordmarkColorClass(): string {
    switch (this.tone()) {
      case 'light':
        return 'text-on-forest-deep';
      case 'mono':
        return 'text-current';
      default:
        return 'text-ink';
    }
  }
}
