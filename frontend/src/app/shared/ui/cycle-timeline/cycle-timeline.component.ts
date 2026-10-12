import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { CycleStatus } from '../../../domain/enums';
import { IconComponent } from '../icon/icon.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

type StepState = 'done' | 'current' | 'todo';

interface Step {
  readonly labelKey: string;
  readonly state: StepState;
}

const STEP_LABEL_KEYS = ['cycleTimeline.step1', 'cycleTimeline.step2', 'cycleTimeline.step3'] as const;
const STEP_INDEX: Readonly<Record<Exclude<CycleStatus, 'cancelled'>, number>> = {
  planned: 0,
  in_progress: 1,
  done: 2
};

/** The cycle's progress — responsive by itself (a row on wide screens, a
 * column on narrow ones) rather than needing a `direction` input, so one
 * component serves both the desktop and mobile boards. */
@Component({
  selector: 'bt-cycle-timeline',
  imports: [IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (status() === 'cancelled') {
      <div class="flex items-center gap-2 text-alert">
        <bt-icon name="x" [size]="18" />
        <span class="text-body-strong">{{ 'cycleTimeline.cancelled' | t }}</span>
      </div>
    } @else {
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:gap-0">
        @for (step of steps(); track step.labelKey; let last = $last) {
          <div class="flex items-center gap-3 sm:flex-1">
            <span
              [class]="
                'flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-caption ' +
                (step.state === 'done'
                  ? 'bg-forest text-on-forest'
                  : step.state === 'current'
                    ? 'bg-clay text-on-clay ring-4 ring-clay-soft'
                    : 'bg-surface-sunken text-ink-muted')
              "
            >
              @if (step.state === 'done') {
                <bt-icon name="check" [size]="14" />
              } @else {
                {{ $index + 1 }}
              }
            </span>
            <span [class]="step.state === 'todo' ? 'text-body text-ink-muted' : 'text-body-strong'">{{ step.labelKey | t }}</span>
            @if (!last) {
              <span class="hidden h-px flex-1 bg-line sm:block"></span>
            }
          </div>
        }
      </div>
    }
  `
})
export class CycleTimelineComponent {
  readonly status = input.required<CycleStatus>();

  protected readonly steps = computed<Step[]>(() => {
    const status = this.status();
    if (status === 'cancelled') return [];
    const currentIndex = STEP_INDEX[status];
    return STEP_LABEL_KEYS.map((labelKey, index) => ({
      labelKey,
      state: index < currentIndex ? 'done' : index === currentIndex ? 'current' : 'todo'
    }));
  });
}
