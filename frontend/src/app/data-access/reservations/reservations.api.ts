import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { PaymentMethod } from '../../domain/enums';
import { Reservation, ReservationRequest, ReservationResult } from '../../domain/models';
import { ApiClient } from '../../core/http/api-client';

@Injectable({ providedIn: 'root' })
export class ReservationsApi {
  private readonly api = inject(ApiClient);

  create(listingId: string, input: ReservationRequest): Observable<ReservationResult> {
    return this.api.post<ReservationResult>(`/listings/${listingId}/reservations`, input);
  }

  mineUrl(): string {
    return this.api.url('/reservations/mine');
  }

  pay(reservationId: string, paymentMethod?: PaymentMethod): Observable<Reservation> {
    return this.api.post<Reservation>(`/reservations/${reservationId}/pay`, { paymentMethod });
  }

  confirmDelivery(reservationId: string): Observable<Reservation> {
    return this.api.post<Reservation>(`/reservations/${reservationId}/confirm-delivery`);
  }

  dispute(reservationId: string, reason: string): Observable<void> {
    return this.api.post<void>(`/reservations/${reservationId}/dispute`, { reason });
  }

  byFarmUrl(farmId: string): string {
    return this.api.url(`/farms/${farmId}/reservations`);
  }

  markDelivered(reservationId: string): Observable<Reservation> {
    return this.api.post<Reservation>(`/reservations/${reservationId}/mark-delivered`);
  }
}
