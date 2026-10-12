import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';
import { TranslatePipe } from '../../core/i18n/translate.pipe';

/** A named placeholder for routes scaffolded in the Phase 0 route map but
 * built out in a later phase (see the frontend build plan) — the route
 * table stays complete and reviewable without every page existing yet.
 * The route's `data.titleKey` is a translation key, not literal text. */
@Component({
  selector: 'bt-coming-soon',
  imports: [TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto flex max-w-content-max flex-col items-center gap-2 px-5 py-24 text-center sm:px-18">
      <h1 class="text-heading-2">{{ titleKey() | t }}</h1>
      <p class="text-body text-ink-muted">{{ 'comingSoon.body' | t }}</p>
    </div>
  `
})
export class ComingSoonComponent {
  private readonly route = inject(ActivatedRoute);
  protected readonly titleKey = toSignal(this.route.data.pipe(map((data) => (data['titleKey'] as string) ?? 'comingSoon.default')), {
    initialValue: 'comingSoon.default'
  });
}
