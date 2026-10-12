import { ChangeDetectionStrategy, Component } from '@angular/core';

/** The thumb-zone action bar pinned to the bottom of a phone screen
 * ("2 500 FCFA / poussin · Réserver", "Payer 76 500 FCFA"). Pages that
 * use it are "focus" pages: their route sets `data: { focus: true }` so
 * the shell hides the bottom nav, and this bar takes its place, flush
 * with the screen edge plus the safe-area inset. From `lg` up it is
 * hidden: desktop pages put the same action in their side panel.
 *
 * Pages using it must leave room at the bottom of their content
 * (`pb-28 lg:pb-0`). */
@Component({
  selector: 'bt-sticky-action-bar',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'contents' },
  template: `
    <div
      class="fixed inset-x-0 bottom-0 z-30 border-t border-line bg-surface-raised/95 px-4 pt-3 backdrop-blur lg:hidden"
      style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom)); box-shadow: 0 -8px 24px rgba(18, 61, 44, 0.08)"
    >
      <div class="mx-auto flex max-w-content-max items-center gap-4">
        <ng-content />
      </div>
    </div>
  `
})
export class StickyActionBarComponent {}
