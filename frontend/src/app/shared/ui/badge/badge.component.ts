import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { BadgeTone } from '../../../domain/rules/status-registry';

const TONE_CLASSES: Readonly<Record<BadgeTone, string>> = {
  forest: 'bg-forest-soft text-forest',
  'forest-solid': 'bg-forest text-on-forest',
  clay: 'bg-clay-soft text-clay-ink',
  sky: 'bg-sky-soft text-sky',
  amber: 'bg-amber-soft text-amber',
  alert: 'bg-alert-soft text-alert',
  plum: 'bg-plum-soft text-plum',
  stone: 'bg-stone-soft text-stone',
  ink: 'bg-surface-sunken text-ink',
  neutral: 'bg-surface-sunken text-ink-muted'
};

/** One or two words on a tinted pill. Use `StatusBadge` for cycle, escrow
 * and payment states instead — this is for everything else (species,
 * cycle type, counts). */
@Component({
  selector: 'bt-badge',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <span [class]="'inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-caption ' + toneClass()">
      <ng-content />
    </span>
  `
})
export class BadgeComponent {
  readonly tone = input<BadgeTone>('neutral');

  protected toneClass(): string {
    return TONE_CLASSES[this.tone()];
  }
}
