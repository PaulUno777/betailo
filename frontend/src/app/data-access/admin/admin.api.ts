import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { FarmDecision, FarmQueue, OverviewPeriod, RejectionCode, ResolveAction, StaffRole } from '../../domain/enums';
import { AdminUser, Farm, Reservation, Report, StaffInvitation } from '../../domain/models';
import { ApiClient, QueryParams } from '../../core/http/api-client';

@Injectable({ providedIn: 'root' })
export class AdminApi {
  private readonly api = inject(ApiClient);

  overviewUrl(period: OverviewPeriod): string {
    return this.api.url('/admin/overview', { period } as QueryParams);
  }

  navCountsUrl(): string {
    return this.api.url('/admin/nav-counts');
  }

  usersUrl(): string {
    return this.api.url('/admin/users');
  }

  /** An exception to the platform's farms-per-account, for one account.
   * 0 ends it; above 0 the API asks for a reason. */
  setFarmLimit(id: string, limit: number, reason: string): Observable<AdminUser> {
    return this.api.post<AdminUser>(`/admin/users/${id}/farm-limit`, { limit, reason });
  }

  /** The same, naming the account by its email. */
  setFarmLimitByEmail(email: string, limit: number, reason: string): Observable<AdminUser> {
    return this.api.post<AdminUser>('/admin/farm-limits', { email, limit, reason });
  }

  farmLimitsUrl(): string {
    return this.api.url('/admin/farm-limits');
  }

  farmsUrl(queue: FarmQueue): string {
    return this.api.url('/admin/farms', { queue } as QueryParams);
  }

  farmDetailUrl(id: string): string {
    return this.api.url(`/admin/farms/${id}`);
  }

  /** `checks` are the review points ticked; the API keeps them. */
  reviewFarm(id: string, decision: FarmDecision, reason?: string, reasonCode?: RejectionCode, checks?: readonly string[]): Observable<Farm> {
    return this.api.post<Farm>(`/admin/farms/${id}/review`, { decision, reason, reasonCode, checks });
  }

  /** The accounting export: every escrow transaction as CSV. */
  exportEscrow(): Observable<Blob> {
    return this.api.download('/admin/escrow/export.csv');
  }

  saveReportNote(id: string, note: string): Observable<Report> {
    return this.api.put<Report>(`/admin/reports/${id}/note`, { note });
  }

  suspendUser(id: string): Observable<AdminUser> {
    return this.api.post<AdminUser>(`/admin/users/${id}/suspend`);
  }

  reactivateUser(id: string): Observable<AdminUser> {
    return this.api.post<AdminUser>(`/admin/users/${id}/reactivate`);
  }

  staffUrl(): string {
    return this.api.url('/admin/staff');
  }

  /** Gives a staff role to an existing account, named by its email. */
  grantStaffRole(email: string, role: StaffRole): Observable<AdminUser> {
    return this.api.post<AdminUser>('/admin/staff', { email, role });
  }

  /** Invites an address onto the staff; the answer carries the link once. */
  inviteStaff(email: string, role: StaffRole): Observable<StaffInvitation> {
    return this.api.post<StaffInvitation>('/admin/staff/invitations', { email, role });
  }

  resendInvitation(id: string): Observable<StaffInvitation> {
    return this.api.post<StaffInvitation>(`/admin/staff/invitations/${id}/resend`);
  }

  cancelInvitation(id: string): Observable<void> {
    return this.api.delete<void>(`/admin/staff/invitations/${id}`);
  }

  /** Changes a member's role; an empty role takes them off the team. */
  setStaffRole(id: string, role: StaffRole | ''): Observable<AdminUser> {
    return this.api.post<AdminUser>(`/admin/staff/${id}/role`, { role });
  }

  auditUrl(): string {
    return this.api.url('/admin/audit');
  }

  cyclesUrl(): string {
    return this.api.url('/admin/cycles');
  }

  listingsUrl(): string {
    return this.api.url('/admin/listings');
  }

  escrowUrl(): string {
    return this.api.url('/admin/escrow');
  }

  releaseEscrow(reservationId: string): Observable<Reservation> {
    return this.api.post<Reservation>(`/admin/escrow/${reservationId}/release`);
  }

  refundEscrow(reservationId: string): Observable<Reservation> {
    return this.api.post<Reservation>(`/admin/escrow/${reservationId}/refund`);
  }

  reportsUrl(status?: string): string {
    return this.api.url('/admin/reports', { status } as QueryParams);
  }

  reportDetailUrl(id: string): string {
    return this.api.url(`/admin/reports/${id}`);
  }

  /** The decision closes every report waiting on the same target. */
  resolveReport(id: string, action: ResolveAction, note?: string): Observable<Report> {
    return this.api.post<Report>(`/admin/reports/${id}/resolve`, { action, note });
  }
}
