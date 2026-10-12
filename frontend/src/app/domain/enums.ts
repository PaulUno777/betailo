/* String-literal unions mirroring the backend's exact enum values
 * (backend/domain — verified from source, not inferred). Kept as one
 * module: these are all narrow, related vocabularies with no behaviour
 * of their own. */

/** What a member of the platform's team may do in the back office.
 * Everyone else has none: there is one kind of account. */
export type StaffRole = 'super_admin';

/** A farm's review state. Only `approved` sells. */
export type FarmStatus = 'draft' | 'pending' | 'approved' | 'rejected' | 'suspended';

export type FarmDecision = 'approve' | 'reject' | 'suspend';

export type CycleStatus = 'planned' | 'in_progress' | 'done' | 'cancelled';

export type ListingStatus = 'draft' | 'published' | 'unpublished';

export type ReservationStatus = 'pending' | 'confirmed' | 'delivered' | 'cancelled';

export type PaymentStatus = 'pending' | 'held' | 'released' | 'failed' | 'refunded';

export type PaymentMethod = 'mtn_momo' | 'orange_money' | 'card';

export type ReportTargetType = 'listing' | 'profile' | 'forum_post' | 'reservation';

export type ReportStatus = 'open' | 'under_review' | 'handled' | 'rejected';

export type ResolveAction = 'dismiss' | 'suspend';
