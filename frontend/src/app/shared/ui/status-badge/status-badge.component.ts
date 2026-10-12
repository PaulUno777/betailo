import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { CycleStatus, ListingStatus, PaymentStatus, ReservationStatus } from '../../../domain/enums';
import {
  BadgeTone,
  cycleStatusEntry,
  escrowStatusEntry,
  listingStatusEntry,
  priorityStatusEntry,
  reservationStatusEntry,
  trustScoreTone
} from '../../../domain/rules/status-registry';
import { LocaleService } from '../../../core/i18n/locale.service';
import { BadgeComponent } from '../badge/badge.component';

export type StatusBadgeKind = 'cycle' | 'listing' | 'reservation' | 'escrow' | 'payment' | 'trust';

interface ToneEntry {
  readonly tone: BadgeTone;
  readonly labelKey?: string;
}

/** The canonical label and colour for every domain status — extend
 * `domain/rules/status-registry.ts` to add one, never a colour here.
 * `trust` is the one kind with no fixed vocabulary (it's a raw score),
 * so it renders the number itself rather than a translated label. */
@Component({
  selector: 'bt-status-badge',
  imports: [BadgeComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <bt-badge [tone]="entry().tone">{{ label() }}</bt-badge>
  `
})
export class StatusBadgeComponent {
  private readonly locale = inject(LocaleService);

  readonly kind = input.required<StatusBadgeKind>();
  readonly status = input.required<string | boolean | number>();
  readonly suffix = input<string>();

  protected readonly entry = computed<ToneEntry>(() => {
    const status = this.status();
    switch (this.kind()) {
      case 'cycle':
        return cycleStatusEntry(status as CycleStatus);
      case 'listing':
        return listingStatusEntry(status as ListingStatus);
      case 'reservation':
        return reservationStatusEntry(status as ReservationStatus);
      case 'escrow':
        return escrowStatusEntry(status as PaymentStatus);
      case 'payment':
        return priorityStatusEntry(status as boolean);
      case 'trust':
        return { tone: trustScoreTone(status as number) };
    }
  });

  protected readonly label = computed(() => {
    const { labelKey } = this.entry();
    const base = labelKey ? this.locale.t(labelKey) : `${this.status()}`;
    const suffix = this.suffix();
    return suffix ? `${base} · ${suffix}` : base;
  });
}
