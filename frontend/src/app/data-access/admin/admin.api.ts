import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { FarmDecision, FarmStatus, ResolveAction } from '../../domain/enums';
import { AdminUser, Farm, Reservation, Report } from '../../domain/models';
import { ApiClient, QueryParams } from '../../core/http/api-client';

@Injectable({ providedIn: 'root' })
export class AdminApi {
  private readonly api = inject(ApiClient);

  overviewUrl(): string {
    return this.api.url('/admin/overview');
  }

  usersUrl(): string {
    return this.api.url('/admin/users');
  }

  /** How many farms this account may own; 0 puts it back on the
   * platform default. */
  setFarmLimit(id: string, limit: number): Observable<AdminUser> {
    return this.api.post<AdminUser>(`/admin/users/${id}/farm-limit`, { limit });
  }

  farmsUrl(status?: FarmStatus): string {
    return this.api.url('/admin/farms', { status } as QueryParams);
  }

  farmDetailUrl(id: string): string {
    return this.api.url(`/admin/farms/${id}`);
  }

  reviewFarm(id: string, decision: FarmDecision, reason?: string): Observable<Farm> {
    return this.api.post<Farm>(`/admin/farms/${id}/review`, { decision, reason });
  }

  suspendUser(id: string): Observable<AdminUser> {
    return this.api.post<AdminUser>(`/admin/users/${id}/suspend`);
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

  resolveReport(id: string, action: ResolveAction): Observable<Report> {
    return this.api.post<Report>(`/admin/reports/${id}/resolve`, { action });
  }
}
