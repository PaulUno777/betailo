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
}

export interface Farm {
  id: string;
  ownerId: string;
  name: string;
  location: string;
  species?: string[];
}

export interface Cycle {
  id: string;
  farmId: string;
  species: string;
  type: string;
  status: CycleStatus;
  capacity: number;
  availableQuantity: number;
  startsAt: string;
  endsAt?: string;
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
}

export interface Reservation {
  id: string;
  listingId: string;
  buyerId: string;
  quantity: number;
  unitPrice: number;
  serviceFee: number;
  totalAmount: number;
  status: ReservationStatus;
  priority: boolean;
  paymentDeadline?: string;
  deliveredAt?: string;
  createdAt: string;
  payment?: ReservationPayment;
}

export interface ReviewSummary {
  average: number;
  count: number;
  histogram: Record<string, number>;
}

export interface Review {
  id: string;
  rating: number;
  comment?: string;
}

export interface FarmReviews {
  reviews: Review[];
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

export interface SellerDashboard {
  trustScore: number;
  reservationsThisPeriod: number;
  escrowHeldTotal: number;
  releasedThisPeriod: number;
  weeklyReservationCounts: number[];
  activeCycles: CycleSummary[];
  latestReservations: Reservation[];
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
}

export interface ReservationResult {
  reservation: Reservation;
  paymentCaptured: boolean;
}

export type { ResolveAction };
