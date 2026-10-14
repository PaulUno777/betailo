/* String-literal unions mirroring the backend's exact enum values
 * (backend/domain — verified from source, not inferred). Kept as one
 * module: these are all narrow, related vocabularies with no behaviour
 * of their own. */

/** What a member of the platform's team may do in the back office.
 * Everyone else has none: there is one kind of account. */
export type StaffRole = 'super_admin' | 'moderator' | 'finance' | 'support';

/** What a staff role may do in the back office. The API holds the
 * table (`GET /admin/staff`) and returns the caller's own list on
 * `GET /me`; these are only the codes. */
export type Permission =
  | 'overview.read'
  | 'accounts.read'
  | 'accounts.suspend'
  | 'accounts.farm_limit'
  | 'farms.read'
  | 'farms.review'
  | 'catalog.read'
  | 'reports.read'
  | 'reports.resolve'
  | 'escrow.read'
  | 'escrow.settle'
  | 'staff.manage'
  | 'audit.read';

export type IdDocumentType = 'national_id' | 'passport' | 'driving_licence';

export type RejectionCode = 'id_unreadable' | 'name_mismatch' | 'photos_not_recent' | 'farm_not_found' | 'other';

export type FarmQueue = 'todo' | 'resubmitted' | 'done';

export type OverviewPeriod = '7d' | '30d' | '12m';

/** A farm's review state. Only `approved` sells. */
export type FarmStatus = 'draft' | 'pending' | 'approved' | 'rejected' | 'suspended';

export type FarmDecision = 'approve' | 'reject' | 'suspend';

export type CycleStatus = 'planned' | 'in_progress' | 'done' | 'cancelled';

export type ListingStatus = 'draft' | 'published' | 'unpublished';

export type ReservationStatus = 'pending' | 'confirmed' | 'delivered' | 'cancelled';

export type PaymentStatus = 'pending' | 'held' | 'released' | 'failed' | 'refunded';

export type PaymentMethod = 'mtn_momo' | 'orange_money' | 'card';

export type ReportTargetType = 'listing' | 'profile' | 'forum_post' | 'reservation';

/** How urgent a report looks in the queue; the API decides. */
export type ReportLevel = 'high' | 'medium' | 'low';

export type ReportStatus = 'open' | 'under_review' | 'handled' | 'rejected';

export type ResolveAction = 'dismiss' | 'suspend';
