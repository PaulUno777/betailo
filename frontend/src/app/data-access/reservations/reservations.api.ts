import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { PaymentMethod } from '../../domain/enums';
import { Reservation, ReservationGroup, ReservationQuote, ReservationRequest, ReservationResult } from '../../domain/models';
import { ApiClient } from '../../core/http/api-client';

@Injectable({ providedIn: 'root' })
export class ReservationsApi {
  private readonly api = inject(ApiClient);

  quote(listingId: string, quantity: number): Observable<ReservationQuote> {
    return this.api.post<ReservationQuote>(`/listings/${listingId}/quote`, { quantity });
  }

  create(listingId: string, input: ReservationRequest): Observable<ReservationResult> {
    return this.api.post<ReservationResult>(`/listings/${listingId}/reservations`, input);
  }

  /** A `ReservationList` for one tab (or every tab without `group`). */
  mineUrl(group?: ReservationGroup): string {
    return this.api.url('/reservations/mine', group ? { group } : undefined);
  }

  pay(reservationId: string, paymentMethod: PaymentMethod, payerPhone?: string): Observable<Reservation> {
    return this.api.post<Reservation>(`/reservations/${reservationId}/pay`, { paymentMethod, payerPhone });
  }

  cancel(reservationId: string): Observable<Reservation> {
    return this.api.post<Reservation>(`/reservations/${reservationId}/cancel`);
  }

  confirmDelivery(reservationId: string): Observable<Reservation> {
    return this.api.post<Reservation>(`/reservations/${reservationId}/confirm-delivery`);
  }

  dispute(reservationId: string, reason: string): Observable<void> {
    return this.api.post<void>(`/reservations/${reservationId}/dispute`, { reason });
  }

  /** A `ReservationList` of the farm's reservations, farmer's view. */
  byFarmUrl(farmId: string, group?: ReservationGroup): string {
    return this.api.url(`/farms/${farmId}/reservations`, group ? { group } : undefined);
  }

  markDelivered(reservationId: string): Observable<Reservation> {
    return this.api.post<Reservation>(`/reservations/${reservationId}/mark-delivered`);
  }
}
