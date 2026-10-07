import {
  CycleStatus,
  ListingStatus,
  PaymentMethod,
  PaymentStatus,
  ReportStatus,
  ReportTargetType,
  ReservationStatus,
  ResolveAction,
  Role
} from './enums';

/* Interfaces mirror the backend's JSON responses field-for-field
 * (verified from backend/interfaces/httpserver source). IDs are
 * 24-char Mongo ObjectID hex strings; timestamps are RFC3339 strings —
 * kept as `string` here rather than `Date`, converted only at the
 * presentation edge (pipes), so a model is exactly what the wire sends. */

export interface User {
  id: string;
  email: string;
  role: Role;
  displayName?: string;
}

/** `GET /me`: the signed-in user's own account. */
export interface Profile extends User {
  phone?: string;
  city?: string;
  verified: boolean;
  trustScore: number;
  memberSince: string;
}

export interface Farm {
  id: string;
  ownerId: string;
  name: string;
  location: string;
  species?: string[];
  description?: string;
  /** E.164. Only present for the owner and signed-in viewers. */
  contactPhone?: string;
}

/** `GET /farms/{id}`: a farm plus its server-computed trust signals. */
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
  farmVerified: boolean;
  farmTrustScore: number;
  reviewAverage: number;
  reviewCount: number;
  favorited: boolean;
}

export interface ListingSearchResult {
  items: MarketplaceListing[];
  total: number;
}

export interface ListingSearchParams {
  species?: string;
  cycleType?: string;
  region?: string;
  minPrice?: number;
  maxPrice?: number;
  verifiedOnly?: boolean;
  q?: string;
  sort?: string;
  page?: number;
  pageSize?: number;
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

export interface SellerDashboard {
  trustScore: number;
  reservationsThisPeriod: number;
  escrowHeldTotal: number;
  releasedThisPeriod: number;
  weeklyReservationCounts: number[];
  activeCycles: CycleSummary[];
  latestReservations: Reservation[];
  todos: DashboardTodo[];
}

export interface AdminOverview {
  volumeReserved: number;
  escrowHeld: number;
  activeCyclesPlanned: number;
  activeCyclesInProgress: number;
  openReports: number;
  reservationsPerDay: number[];
}

export interface AdminUser {
  id: string;
  email: string;
  role: Role;
  displayName?: string;
  isVerified: boolean;
  suspended: boolean;
  trustScore: number;
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
