import {
  CycleStatus,
  FarmStatus,
  IdDocumentType,
  ListingStatus,
  OverviewPeriod,
  PaymentMethod,
  PaymentStatus,
  Permission,
  ProofPart,
  RejectionCode,
  ReportLevel,
  ReportStatus,
  ReportTargetType,
  ReservationStatus,
  ResolveAction,
  StaffRole
} from './enums';

/* Interfaces mirror the backend's JSON responses field-for-field
 * (verified from backend/interfaces/httpserver source). IDs are
 * 24-char Mongo ObjectID hex strings; timestamps are RFC3339 strings —
 * kept as `string` here rather than `Date`, converted only at the
 * presentation edge (pipes), so a model is exactly what the wire sends. */

export interface User {
  id: string;
  email: string;
  /** Absent for everyone but the platform's team. */
  staffRole?: StaffRole;
  displayName?: string;
}

/** The signed-in user's own profile (`PATCH /me`). */
export interface Profile extends User {
  phone?: string;
  city?: string;
  trustScore: number;
  memberSince: string;
}

/** `GET /me`: the profile plus what the account can do as a seller,
 * decided by the API: the farms it owns with their review state, how
 * many it may own, and whether it may open another. */
export interface Account extends Profile {
  farms: Farm[];
  farmLimit: number;
  canAddFarm: boolean;
  /** Set when staff gave this account its own farm limit. */
  farmLimitRaisedAt?: string;
  /** What the account's staff role allows; empty for everyone else. */
  permissions: Permission[];
  /** The platform's root account. */
  isRoot?: boolean;
}

export interface Farm {
  id: string;
  /** The code staff and owners quote ("FRM-0212"). */
  reference?: string;
  ownerId: string;
  name: string;
  location: string;
  /** ISO 3166-1 alpha-2. */
  country?: string;
  region?: string;
  species?: string[];
  description?: string;
  /** E.164. Only present for the owner and signed-in viewers. */
  contactPhone?: string;
  status: FarmStatus;
  /** The review, as its owner and staff see it. Never on a public profile. */
  review?: FarmReview;
}

export interface FarmReview {
  submittedAt?: string;
  /** How many times the farm was sent for review. */
  submissions: number;
  idDocumentType?: IdDocumentType;
  reviewedAt?: string;
  /** Let through by the platform's test switch, not by a person. */
  autoApproved?: boolean;
  /** The kind of refusal staff picked, and the message they wrote. */
  rejectionCode?: RejectionCode;
  rejectionReason?: string;
  /** A refusal split into what to send again and what was accepted. */
  toRedo?: ProofPart[];
  accepted?: ProofPart[];
  /** The role of the staff member who decided. */
  reviewerRole?: StaffRole;
  /** How many farm photos the last submission held. */
  photoCount: number;
  /** The farm is waiting on its owner to (re)submit proof. */
  canSubmit: boolean;
  /** It is in the review queue and can be taken back out. */
  canWithdraw: boolean;
}

/** `GET /admin/farms`: a farm in the validation queue, with its owner. */
export interface AdminFarm extends Farm {
  owner?: AdminUser;
}

/** `GET /admin/farms/{id}`: plus short-lived links to the proof. */
export interface AdminFarmDetail extends AdminFarm {
  idDocumentUrl?: string;
  idDocumentBackUrl?: string;
  photoUrls: string[];
  ownerFarmCount: number;
  ownerFarmLimit: number;
  ownerPurchases: number;
  ownerReports: number;
  reviewerName?: string;
  reviewerRole?: StaffRole;
  /** The refusal reasons staff pick from. */
  rejectionCodes: RejectionCode[];
}

/** `GET /admin/farms?queue=`: one tab of the validation page, plus the
 * sizes of the tabs still to work through. */
export interface AdminFarmList {
  farms: AdminFarm[];
  counts: { todo: number; resubmitted: number };
}

/** `GET /farms/{id}`: a farm plus its server-computed trust signals. */
/** One farm on its owner's "Mes fermes" page: the farm with a photo of
 * it and the figures that say how it is doing. */
export interface FarmCard extends Farm {
  /** Short-lived link to the first photo sent with the farm's proof. */
  coverUrl?: string;
  activeCycles: number;
  salesCount: number;
  trustScore: number;
  reviewAverage: number;
  reviewCount: number;
}

export interface FarmProfile extends Farm {
  trustScore: number;
  verified: boolean;
  salesCount: number;
  memberSince: string;
  reviewAverage: number;
  reviewCount: number;
}

export interface Cycle {
  id: string;
  reference?: string;
  farmId: string;
  species: string;
  type: string;
  status: CycleStatus;
  capacity: number;
  availableQuantity: number;
  startsAt: string;
  endsAt?: string;
  minPerReservation?: number;
  maxPerReservation?: number;
}

export type TimelineStepKind = 'start' | 'health' | 'end';
export type TimelineStepState = 'done' | 'current' | 'upcoming';

/** One step of a cycle's server-built timeline. `title`/`detail` only
 * exist on health entries (the farmer's own words); start and end steps
 * are labelled client-side. */
export interface TimelineStep {
  kind: TimelineStepKind;
  date: string;
  title?: string;
  detail?: string;
  state: TimelineStepState;
}

export interface CycleDetail extends Cycle {
  timeline: TimelineStep[];
  /** What the owner may move the cycle to next, decided server-side. */
  nextStatuses: CycleStatus[];
}

export interface Listing {
  id: string;
  cycleId?: string;
  title: string;
  description?: string;
  unitPrice: number;
  unit?: string;
  status: ListingStatus;
  photos?: string[];
  breed?: string;
  ageWeeks?: number;
  deliveryAvailable?: boolean;
}

/** `GET /farms/{farmId}/listings`: the owner's view, every status. */
export interface FarmListing {
  listing: Listing;
  cycle: Cycle;
  activeReservations: number;
  /** Storage keys behind `listing.photos`, same order. */
  photoKeys: string[];
}

/** `GET /listings/price-hint`: what comparable published listings ask
 * per unit (quartiles). Absent (204) below the API's sample minimum. */
export interface PriceHint {
  count: number;
  low: number;
  median: number;
  high: number;
  /** True when the range is national because the region had too few. */
  regional: boolean;
}

/** The search-result shape returned by `GET /listings` — a listing joined
 * server-side with its cycle and farm summary for the marketplace grid. */
export interface MarketplaceListing {
  listing: Listing;
  species: string;
  cycleType: string;
  cycleStatus: CycleStatus;
  startsAt: string;
  endsAt?: string;
  capacity: number;
  availableQuantity: number;
  farmId: string;
  farmName: string;
  farmLocation: string;
  farmCountry?: string;
  farmRegion?: string;
  farmVerified: boolean;
  farmTrustScore: number;
  reviewAverage: number;
  reviewCount: number;
  favorited: boolean;
}

export interface FacetCount {
  value: string;
  count: number;
}

export interface RegionFacet {
  country: string;
  region: string;
  count: number;
}

/** What each filter would lead to, counted server-side: every
 * dimension with the other filters applied but not its own. */
export interface SearchFacets {
  species: FacetCount[];
  cycleTypes: FacetCount[];
  statuses: FacetCount[];
  regions: RegionFacet[];
  priceMin: number;
  priceMax: number;
}

export interface ListingSearchResult {
  items: MarketplaceListing[];
  total: number;
  facets?: SearchFacets;
}

/** Multi-value filters travel comma-separated (`species=a,b`). */
export interface ListingSearchParams {
  species?: string;
  cycleType?: string;
  status?: string;
  country?: string;
  region?: string;
  /** A month (`2026-10`) or a day: lots open at some point in it. */
  available?: string;
  minPrice?: number;
  maxPrice?: number;
  verifiedOnly?: boolean;
  q?: string;
  sort?: string;
  page?: number;
  pageSize?: number;
}

/** `GET /reference/geo`: the countries whose regions are a fixed list. */
export interface GeoReference {
  defaultCountry: string;
  countries: { code: string; regions: string[] }[];
  /** International prefix per country ("GH": "233"). */
  callingCodes: Record<string, string>;
}

export interface ReservationPayment {
  status: PaymentStatus;
  amount: number;
  heldAt?: string;
  releasedAt?: string;
  method?: PaymentMethod;
  payerPhoneMasked?: string;
}

export interface Reservation {
  id: string;
  reference: string;
  listingId: string;
  buyerId: string;
  quantity: number;
  unitPrice: number;
  serviceFee: number;
  totalAmount: number;
  status: ReservationStatus;
  priority: boolean;
  deliveredAt?: string;
  createdAt: string;
  payment?: ReservationPayment;
}

export type ReservationGroup = 'active' | 'done' | 'cancelled';
export type ReservationAction = 'pay' | 'cancel' | 'confirm' | 'dispute' | 'review' | 'contact' | 'mark_delivered';
export type EscrowStepKey = 'paid' | 'delivered' | 'confirmed' | 'released';

/** One reservation card, joined and decided by the API: what was
 * reserved, from whom, its tab, escrow progress and allowed actions. */
export interface ReservationView extends Reservation {
  listing: { id: string; title: string; unit?: string; photo?: string };
  farm: { id: string; name: string; location: string; contactPhone?: string };
  cycle: { id: string; reference: string; status: CycleStatus; startsAt: string; endsAt?: string };
  group: ReservationGroup;
  steps: { key: EscrowStepKey; state: TimelineStepState }[];
  autoReleaseAt?: string;
  disputeOpen: boolean;
  reviewed: boolean;
  actions: ReservationAction[];
}

export interface ReservationList {
  items: ReservationView[];
  counts: Record<ReservationGroup, number>;
  escrowTotal: number;
}

/** `POST /listings/{id}/quote`: the checkout's server-computed preview. */
export interface ReservationQuote {
  quantity: number;
  unitPrice: number;
  subtotal: number;
  serviceFee: number;
  total: number;
  feeRate: number;
  min: number;
  max: number;
  available: number;
}

export interface AppNotification {
  id: string;
  type: string;
  message: string;
  params?: Record<string, string>;
  link?: string;
  createdAt: string;
  read: boolean;
}

export interface NotificationList {
  items: AppNotification[];
  unread: number;
}

export interface ReviewCriteria {
  health?: number;
  conformity?: number;
  communication?: number;
  timeliness?: number;
}

export interface ReviewSummary {
  average: number;
  count: number;
  histogram: Partial<Record<string, number>>;
  /** Per-criterion averages over the reviews that rated it (0 = none). */
  criteria: Required<ReviewCriteria>;
}

export interface Review {
  id: string;
  rating: number;
  comment?: string;
  criteria: ReviewCriteria;
  tags: string[];
  /** Signed download URLs. */
  photos: string[];
  helpfulCount: number;
  votedByMe: boolean;
  /** The viewer wrote it, so they can't vote on it. */
  mine: boolean;
  reply?: { text: string; at: string };
  /** "Joël T.", never an email; absent when the author set no name. */
  authorName?: string;
  /** The verified purchase: "Avis vérifié · RSV-1877". */
  reservationRef?: string;
  createdAt: string;
}

/** `GET /listings/{id}/detail`: the whole listing page in one payload,
 * including viewer-specific decisions the client must not recompute. */
export interface ListingDetail {
  listing: Listing;
  cycle: CycleDetail;
  farm: FarmProfile;
  reviewSummary: ReviewSummary;
  topReviews: Review[];
  favorited: boolean;
  isOwner: boolean;
  canReserve: boolean;
}

export type ReviewFilter = '' | 'photos' | '5' | 'critical';

/** `GET /farms/{id}/reviews`: one filtered page plus the farm summary. */
export interface FarmReviewsPage {
  items: Review[];
  total: number;
  summary: ReviewSummary;
}

export interface Report {
  id: string;
  targetId: string;
  targetType: ReportTargetType;
  reason: string;
  status: ReportStatus;
  /** Staff queue only: what was reported by name, who reported it, how
   * many reports the target drew, and how urgent it looks. */
  targetLabel?: string;
  reporterName?: string;
  createdAt?: string;
  count: number;
  level: ReportLevel;
}

export interface ReportRiskSignal {
  name: string;
  detail: string;
  weight: number;
}

export interface ReportDetail {
  report: Report;
  history: Report[];
  riskScore: number;
  signals: ReportRiskSignal[];
  /** The account behind the report, when there is one to act on. */
  subject?: { id: string; name: string; city?: string; createdAt: string; suspended: boolean };
  /** The reported listing, previewed. */
  listing?: { id: string; title: string; unitPrice: number; unit?: string; status: ListingStatus; photoUrl?: string; farmName?: string };
}

export interface ForumReply {
  authorId: string;
  body: string;
  createdAt: string;
}

export interface ForumPost {
  id: string;
  authorId: string;
  category: string;
  title: string;
  body: string;
  listingId?: string;
  createdAt: string;
  replies?: ForumReply[];
  bestReplyIndex?: number;
}

export interface CycleSummary {
  cycle: Cycle;
  /** The farm the cycle runs on, for the combined view. */
  farmName?: string;
  fillRate: number;
  nextMilestone?: string;
  nextMilestoneDate?: string;
}

export type DashboardTodoKind = 'mark_delivered' | 'health_log' | 'awaiting_payment';

/** One "À faire aujourd'hui" item, decided server-side. */
export interface DashboardTodo {
  kind: DashboardTodoKind;
  count?: number;
  cycleId?: string;
  cycleRef?: string;
}

/** A reservation on the seller's dashboard, with who made it. */
export interface DashboardReservation extends Reservation {
  buyerName?: string;
  cycleRef?: string;
  species?: string;
}

export interface SellerDashboard {
  trustScore: number;
  reservationsThisPeriod: number;
  /** Last month's count, to compare with. */
  reservationsPreviousPeriod: number;
  escrowHeldTotal: number;
  /** How many reservations that escrow covers. */
  heldReservations: number;
  releasedThisPeriod: number;
  /** Reservations made in the last seven days. */
  newReservations: number;
  /** How many farms the figures cover: one, or all on the combined view. */
  farmCount: number;
  reviewAverage: number;
  reviewCount: number;
  weeklyReservationCounts: number[];
  activeCycles: CycleSummary[];
  latestReservations: DashboardReservation[];
  todos: DashboardTodo[];
}

/** One bar of the overview chart: a day, or a month on the 12-month view. */
export interface OverviewBucket {
  start: string;
  paid: number;
  waiting: number;
}

export interface AdminOverview {
  period: OverviewPeriod;
  volumeReserved: number;
  volumePrevious: number;
  escrowHeld: number;
  heldReservations: number;
  activeCyclesPlanned: number;
  activeCyclesInProgress: number;
  openReports: number;
  openDisputes: number;
  pendingFarms: number;
  oldestPendingSince?: string;
  series: OverviewBucket[];
  /** The deployment's switches, read-only. */
  autoApproveFarms: boolean;
  maxFarmsPerAccount: number;
}

/** `GET /admin/nav-counts`: work waiting for staff. */
export interface AdminNavCounts {
  pendingFarms: number;
  openReports: number;
  openDisputes: number;
}

export interface FarmLimitNote {
  reason: string;
  setAt: string;
  setByName?: string;
  setByRole?: StaffRole;
}

export interface AdminUser {
  id: string;
  email: string;
  staffRole?: StaffRole;
  isRoot?: boolean;
  displayName?: string;
  /** What staff set for this account; 0 = the platform default. */
  farmLimit: number;
  farmLimitNote?: FarmLimitNote;
  farmCount: number;
  suspended: boolean;
  trustScore: number;
  memberSince: string;
}

/** `GET /admin/farm-limits`: the platform default (set by the
 * deployment) and every account staff made an exception for. */
export interface FarmLimits {
  default: number;
  max: number;
  exceptions: AdminUser[];
}

/** `GET /admin/staff`: the team, and the permission table as the API
 * enforces it. */
export interface StaffOverview {
  members: AdminUser[];
  roles: { role: StaffRole; permissions: Permission[] }[];
  permissions: Permission[];
}

/** One line of the back office's journal. */
export interface AdminAction {
  id: string;
  actorName?: string;
  actorRole?: StaffRole;
  action: string;
  targetType?: string;
  targetId?: string;
  targetLabel?: string;
  detail?: string;
  createdAt: string;
}

/** A reservation in the "Séquestre" table, with both parties named. */
export interface AdminEscrowItem extends Reservation {
  buyerName?: string;
  farmName?: string;
}

export interface ReservationRequest {
  quantity: number;
  payNow: boolean;
  paymentMethod?: PaymentMethod;
  /** Mobile Money number as typed; the API normalizes and validates it. */
  payerPhone?: string;
}

export interface ReservationResult {
  reservation: Reservation;
  paymentCaptured: boolean;
}

export type { ResolveAction };
