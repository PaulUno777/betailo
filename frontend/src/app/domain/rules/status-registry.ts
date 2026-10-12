import { CycleStatus, FarmStatus, ListingStatus, PaymentStatus, ReservationStatus } from '../enums';

/** A Badge tone, from the design system's fixed tinted-pill palette. */
export type BadgeTone = 'forest' | 'forest-solid' | 'clay' | 'sky' | 'amber' | 'alert' | 'plum' | 'stone' | 'ink' | 'neutral';

/** `labelKey` into the i18n dictionary (`core/i18n`) — this module stays
 * locale-agnostic on purpose; only `TranslatePipe`/`LocaleService` turn a
 * key into displayed text, so a status's color and its copy source live
 * in exactly one place each. */
export interface StatusEntry {
  readonly labelKey: string;
  readonly tone: BadgeTone;
}

/** `reservation.priority` rendered as a status, per the design's fixed
 * status vocabulary (paid = forest "prioritaire", unpaid = amber). This
 * is presentation only — the boolean itself always comes from the API. */
export type PriorityStatus = 'paid' | 'unpaid';

/** Canonical label-key + colour for every domain status `StatusBadge`
 * renders. Extend by adding an entry here (and the matching key in
 * `core/i18n/translations/*.ts`) — never by branching in a component, so
 * a status can never drift to a different colour in two places. */
const CYCLE_STATUS: Readonly<Record<CycleStatus, StatusEntry>> = {
  planned: { labelKey: 'status.cyclePlanned', tone: 'sky' },
  in_progress: { labelKey: 'status.cycleInProgress', tone: 'forest' },
  done: { labelKey: 'status.cycleDone', tone: 'stone' },
  cancelled: { labelKey: 'status.cycleCancelled', tone: 'alert' }
};

const LISTING_STATUS: Readonly<Record<ListingStatus, StatusEntry>> = {
  draft: { labelKey: 'status.listingDraft', tone: 'stone' },
  published: { labelKey: 'status.listingPublished', tone: 'forest' },
  unpublished: { labelKey: 'status.listingUnpublished', tone: 'stone' }
};

const RESERVATION_STATUS: Readonly<Record<ReservationStatus, StatusEntry>> = {
  pending: { labelKey: 'status.reservationPending', tone: 'amber' },
  confirmed: { labelKey: 'status.reservationConfirmed', tone: 'sky' },
  delivered: { labelKey: 'status.reservationDelivered', tone: 'forest' },
  cancelled: { labelKey: 'status.reservationCancelled', tone: 'alert' }
};

/** Escrow state — `reservation.payment.status`. */
const ESCROW_STATUS: Readonly<Record<PaymentStatus, StatusEntry>> = {
  pending: { labelKey: 'status.escrowPending', tone: 'amber' },
  held: { labelKey: 'status.escrowHeld', tone: 'sky' },
  released: { labelKey: 'status.escrowReleased', tone: 'forest' },
  failed: { labelKey: 'status.escrowFailed', tone: 'alert' },
  refunded: { labelKey: 'status.escrowRefunded', tone: 'plum' }
};

/** A farm's review by the platform's team. */
const FARM_STATUS: Readonly<Record<FarmStatus, StatusEntry>> = {
  draft: { labelKey: 'status.farmDraft', tone: 'stone' },
  pending: { labelKey: 'status.farmPending', tone: 'amber' },
  approved: { labelKey: 'status.farmApproved', tone: 'forest' },
  rejected: { labelKey: 'status.farmRejected', tone: 'alert' },
  suspended: { labelKey: 'status.farmSuspended', tone: 'alert' }
};

const PRIORITY_STATUS: Readonly<Record<PriorityStatus, StatusEntry>> = {
  paid: { labelKey: 'status.priorityPaid', tone: 'forest' },
  unpaid: { labelKey: 'status.priorityUnpaid', tone: 'amber' }
};

export function cycleStatusEntry(status: CycleStatus): StatusEntry {
  return CYCLE_STATUS[status];
}

export function listingStatusEntry(status: ListingStatus): StatusEntry {
  return LISTING_STATUS[status];
}

export function reservationStatusEntry(status: ReservationStatus): StatusEntry {
  return RESERVATION_STATUS[status];
}

export function escrowStatusEntry(status: PaymentStatus): StatusEntry {
  return ESCROW_STATUS[status];
}

export function farmStatusEntry(status: FarmStatus): StatusEntry {
  return FARM_STATUS[status];
}

export function priorityStatusEntry(priority: boolean): StatusEntry {
  return PRIORITY_STATUS[priority ? 'paid' : 'unpaid'];
}

/** Trust score is a continuous value, not a discrete status, so it gets
 * its own threshold rule rather than a lookup table (per TrustScore's
 * README: 80+ forest, 60-79 amber, below 60 alert). */
export function trustScoreTone(score: number): BadgeTone {
  if (score >= 80) return 'forest';
  if (score >= 60) return 'amber';
  return 'alert';
}
