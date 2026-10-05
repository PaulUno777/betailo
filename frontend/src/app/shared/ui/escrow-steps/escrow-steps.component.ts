import { ChangeDetectionStrategy, Component, input } from '@angular/core';

export type EscrowStepState = 'done' | 'current' | 'pending';

export interface EscrowStep {
  readonly label: string;
  readonly detail?: string;
  readonly state: EscrowStepState;
}

/** The horizontal delivery/escrow progress line on a reservation card
 * (`BuyerPortal.dc.html`'s centerpiece) — a fixed sequence of named
 * steps, each rendered from real reservation/payment state, never
 * invented. A step this session's data can't back (e.g. a per-cycle
 * "hatched" event) is simply left out of the `steps` input rather than
 * faked here. */
@Component({
  selector: 'bt-escrow-steps',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex items-start">
      @for (step of steps(); track step.label; let last = $last) {
        <div class="flex flex-1 flex-col items-center gap-1.5 text-center">
          <div class="flex w-full items-center">
            <span class="h-px flex-1" [class]="$index === 0 ? 'bg-transparent' : lineClass(step.state)"></span>
            <span [class]="'flex h-4.5 w-4.5 shrink-0 items-center justify-center rounded-full box-border ' + dotClass(step.state)"></span>
            <span class="h-px flex-1" [class]="last ? 'bg-transparent' : lineClass(step.state)"></span>
          </div>
          <span class="text-caption font-medium" [class]="step.state === 'pending' ? 'text-ink-muted' : 'text-ink'">{{ step.label }}</span>
          @if (step.detail; as detail) {
            <span class="text-caption" [class]="step.state === 'current' ? 'text-clay-ink' : 'text-ink-muted'">{{ detail }}</span>
          }
        </div>
      }
    </div>
  `
})
export class EscrowStepsComponent {
  readonly steps = input.required<readonly EscrowStep[]>();

  protected dotClass(state: EscrowStepState): string {
    switch (state) {
      case 'done':
        return 'bg-forest';
      case 'current':
        return 'border-2 border-clay bg-surface-card';
      case 'pending':
        return 'border-2 border-line-strong bg-surface-card';
    }
  }

  protected lineClass(state: EscrowStepState): string {
    return state === 'pending' ? 'bg-line' : 'bg-forest';
  }
}
