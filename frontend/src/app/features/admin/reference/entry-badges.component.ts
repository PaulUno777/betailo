import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { Mark } from './reference.types';

/** An entry's state at a glance: pending change, inactive, usage. */
@Component({
  selector: 'bt-entry-badges',
  imports: [TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'inline-flex flex-wrap items-center gap-1' },
  template: `
    @if (mark(); as m) {
      <span class="rounded-full px-2 py-0.5 text-[11px] font-semibold" [class]="markClass[m]">{{ 'adminReference.mark.' + m | t }}</span>
    }
    @if (!active()) {
      <span class="rounded-full bg-surface-sunken px-2 py-0.5 text-[11px] font-semibold text-ink-muted">{{ 'adminReference.inactive' | t }}</span>
    }
    @if (usage() !== null) {
      <span class="rounded-full border border-line px-2 py-0.5 font-mono text-[11px] text-ink-muted" [attr.title]="'adminReference.usedBy' | t: { count: usage()! }">{{ usage() }}</span>
    }
  `
})
export class EntryBadgesComponent {
  readonly mark = input<Mark | undefined>();
  readonly active = input(true);
  /** How many farms, cycles, posts… use it; null when not counted. */
  readonly usage = input<number | null>(null);

  protected readonly markClass: Record<Mark, string> = {
    added: 'bg-forest-soft text-on-forest-soft',
    edited: 'bg-clay-soft text-clay-ink',
    moved: 'bg-sky-soft text-sky'
  };
}
