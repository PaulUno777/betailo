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
      <!-- The brand kit's symbol (betailo-symbole), on its own 49.2×50.4 box. -->
      <svg [attr.width]="markSize()" [attr.height]="markSize()" viewBox="7.4 6.8 49.2 50.4" fill="none" aria-hidden="true">
        <g [attr.stroke]="arrowColor()" stroke-width="6" stroke-linecap="round">
          <path d="M10.6 27.5A22 22 0 0 1 49.5 19" />
          <path d="M53.4 36.5A22 22 0 0 1 14.5 45" />
        </g>
        <g [attr.fill]="arrowColor()">
          <path d="M54.8 26.4 55.4 15 44.2 23Z" />
          <path d="M9.2 37.6 8.6 49 19.8 41Z" />
        </g>
        <path d="M21 43C21 31 29 23 43 21.5 43 34.5 35 43 21 43Z" [attr.fill]="leafColor()" />
      </svg>
      @if (variant() === 'full') {
        <span class="text-heading-4 font-extrabold tracking-tight">Betailo</span>
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
