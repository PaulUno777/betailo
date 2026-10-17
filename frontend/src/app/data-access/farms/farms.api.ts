import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { IdDocumentType } from '../../domain/enums';
import { Farm, FarmContact } from '../../domain/models';
import { PhotoSizes, PresignedUpload } from '../../core/http/photo-upload.service';
import { ApiClient } from '../../core/http/api-client';

export interface FarmInput {
  readonly name: string;
  readonly location: string;
  readonly country?: string;
  readonly region?: string;
  readonly species?: string[];
  readonly description?: string;
  readonly contacts?: FarmContact[];
}

/** Which of a farm's two public images. */
export type FarmImage = 'logo' | 'banner';

/** What an owner sends for review: the identity document (its back
 * too, when it has one), photos of the farm, and their declaration. */
export interface FarmProof {
  idDocumentType: IdDocumentType;
  idDocumentKey: string;
  idDocumentBackKey: string;
  photoKeys: string[];
  certified: boolean;
}

@Injectable({ providedIn: 'root' })
export class FarmsApi {
  private readonly api = inject(ApiClient);

  detailUrl(id: string): string {
    return this.api.url(`/farms/${id}`);
  }

  mineUrl(): string {
    return this.api.url('/farms/mine');
  }

  /** The caller's farms with their photo and figures. */
  mineCardsUrl(): string {
    return this.api.url('/farms/mine/cards');
  }

  create(input: FarmInput): Observable<Farm> {
    return this.api.post<Farm>('/farms', input);
  }

  update(id: string, input: FarmInput): Observable<Farm> {
    return this.api.put<Farm>(`/farms/${id}`, input);
  }

  delete(id: string): Observable<void> {
    return this.api.delete<void>(`/farms/${id}`);
  }

  /** One upload slot for a proof file (identity document or a photo of
   * the farm). Private: only this farm's review ever reads it. */
  presignProof(id: string, sizes: PhotoSizes): Observable<PresignedUpload> {
    return this.api.post<PresignedUpload>(`/farms/${id}/verification/presign`, sizes);
  }

  /** Upload slots for a new logo or banner and its small copy. */
  presignImage(id: string, sizes: PhotoSizes): Observable<PresignedUpload> {
    return this.api.post<PresignedUpload>(`/farms/${id}/branding/presign`, sizes);
  }

  /** Makes an uploaded image the farm's logo or banner; an empty key
   * removes it. */
  setImage(id: string, image: FarmImage, key: string): Observable<Farm> {
    return this.api.put<Farm>(`/farms/${id}/${image}`, { key });
  }

  /** Takes a farm back out of the review queue. */
  withdrawSubmission(id: string): Observable<Farm> {
    return this.api.post<Farm>(`/farms/${id}/verification/withdraw`);
  }

  /** Hands the farm to staff for review. */
  submitForReview(id: string, proof: FarmProof): Observable<Farm> {
    return this.api.post<Farm>(`/farms/${id}/verification/submit`, proof);
  }
}
