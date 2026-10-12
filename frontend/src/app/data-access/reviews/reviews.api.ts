import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { Review, ReviewCriteria, ReviewFilter } from '../../domain/models';
import { ApiClient } from '../../core/http/api-client';
import { PresignedUpload } from '../../core/http/photo-upload.service';

export interface CreateReviewInput {
  readonly rating: number;
  readonly comment?: string;
  readonly criteria?: ReviewCriteria;
  readonly tags?: string[];
  readonly photoKeys?: string[];
}

/** The closed vocabulary the API accepts ("Ce qui vous a plu"). */
export const REVIEW_TAGS = ['healthy_animals', 'as_described', 'on_time', 'good_packaging', 'responsive_farmer', 'good_value'] as const;

@Injectable({ providedIn: 'root' })
export class ReviewsApi {
  private readonly api = inject(ApiClient);

  create(reservationId: string, input: CreateReviewInput): Observable<Review> {
    return this.api.post<Review>(`/reservations/${reservationId}/reviews`, input);
  }

  presignPhoto(reservationId: string): Observable<PresignedUpload> {
    return this.api.post<PresignedUpload>(`/reservations/${reservationId}/reviews/photos/presign`);
  }

  /** A `FarmReviewsPage`. */
  farmReviewsUrl(farmId: string, filter: ReviewFilter, page: number, pageSize = 10): string {
    return this.api.url(`/farms/${farmId}/reviews`, { filter: filter || undefined, page, pageSize });
  }

  markHelpful(reviewId: string): Observable<{ helpfulCount: number }> {
    return this.api.post<{ helpfulCount: number }>(`/reviews/${reviewId}/helpful`);
  }

  reply(reviewId: string, text: string): Observable<Review> {
    return this.api.post<Review>(`/reviews/${reviewId}/reply`, { text });
  }
}
