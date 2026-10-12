import { ChangeDetectionStrategy, Component } from '@angular/core';

/** The rounded pill track for a small set of mutually-exclusive choices
 * (Auth's Connexion/Inscription tabs, a unit toggle). Each item stays a
 * plain `<a>` or `<button>` styled by its own consumer — active state can
 * come from `routerLinkActive`, a signal comparison, or anything else —
 * this component only supplies the shared track chrome. */
@Component({
  selector: 'bt-segmented-control',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div role="tablist" class="inline-flex w-full gap-1 rounded-full bg-surface-sunken p-1">
      <ng-content />
    </div>
  `
})
export class SegmentedControlComponent {}
