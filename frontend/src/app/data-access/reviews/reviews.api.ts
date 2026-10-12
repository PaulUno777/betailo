import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { Review } from '../../domain/models';
import { ApiClient } from '../../core/http/api-client';

export interface CreateReviewInput {
  readonly rating: number;
  readonly comment?: string;
}

@Injectable({ providedIn: 'root' })
export class ReviewsApi {
  private readonly api = inject(ApiClient);

  create(reservationId: string, input: CreateReviewInput): Observable<Review> {
    return this.api.post<Review>(`/reservations/${reservationId}/reviews`, input);
  }
}
