import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { TimelineStep } from '../../../domain/models';
import { IconComponent } from '../icon/icon.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { LocalDatePipe } from '../../pipes/local-date.pipe';

/** "Déroulé du cycle": the server-built timeline (start, every health-log
 * entry, end), rendered as a vertical rail. Which step is done, current
 * or upcoming is decided by the API (`TimelineStep.state`), never here.
 * Health entries show the farmer's own title and detail; start and end
 * steps are labelled from the dictionary. */
@Component({
  selector: 'bt-cycle-timeline',
  imports: [IconComponent, TranslatePipe, LocalDatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (cancelled()) {
      <div class="mb-4 flex items-center gap-2 rounded-md bg-alert-soft px-3 py-2 text-alert">
        <bt-icon name="x" [size]="16" />
        <span class="text-small">{{ 'cycleTimeline.cancelled' | t }}</span>
      </div>
    }
    <ol class="flex flex-col">
      @for (step of steps(); track $index; let last = $last) {
        <li class="relative flex gap-4 pb-5 last:pb-0">
          @if (!last) {
            <span
              class="absolute left-[11px] top-6 bottom-0 w-0.5"
              [class]="step.state === 'done' ? 'bg-forest' : 'bg-line'"
              aria-hidden="true"
            ></span>
          }
          <span
            class="relative z-10 mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full"
            [class]="dotClass(step)"
            aria-hidden="true"
          >
            @if (step.state === 'done') {
              <bt-icon name="check" [size]="13" />
            }
          </span>
          <div class="flex min-w-0 flex-col">
            <span [class]="step.state === 'upcoming' ? 'text-body text-ink-muted' : 'text-body-strong text-ink'">
              {{ step.kind === 'health' ? step.title : ((step.kind === 'start' ? 'cycleTimeline.start' : 'cycleTimeline.end') | t) }}
            </span>
            <span class="text-small text-ink-muted">
              {{ step.date | localDate: 'dayMonth' }}@if (step.detail) { · {{ step.detail }} }
            </span>
            <span class="sr-only">{{ 'cycleTimeline.state.' + step.state | t }}</span>
          </div>
        </li>
      }
    </ol>
  `
})
export class CycleTimelineComponent {
  readonly steps = input.required<readonly TimelineStep[]>();
  readonly cancelled = input(false);

  protected dotClass(step: TimelineStep): string {
    switch (step.state) {
      case 'done':
        return 'bg-forest text-on-forest';
      case 'current':
        return 'bg-clay ring-4 ring-clay-soft';
      default:
        return 'border-2 border-line-strong bg-surface-card';
    }
  }
}
