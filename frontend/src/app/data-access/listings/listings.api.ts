import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { Listing, ListingSearchParams } from '../../domain/models';
import { ApiClient, QueryParams } from '../../core/http/api-client';

export interface ListingInput {
  readonly cycleId: string;
  readonly title: string;
  readonly description?: string;
  readonly unitPrice: number;
  readonly unit?: string;
}

export interface PresignPhotoResult {
  readonly key: string;
  readonly uploadUrl: string;
}

/** Exposes URLs rather than Observables for reads — these feed Angular's
 * `httpResource()` directly from `features/*`, which needs a URL (or a
 * function returning one) rather than a subscribed stream. `ApiClient`
 * stays the one place that knows the base URL and query-param encoding. */
@Injectable({ providedIn: 'root' })
export class ListingsApi {
  private readonly api = inject(ApiClient);

  searchUrl(params: ListingSearchParams): string {
    return this.api.url('/listings', params as QueryParams);
  }

  detailUrl(id: string): string {
    return this.api.url(`/listings/${id}`);
  }

  create(input: ListingInput): Observable<Listing> {
    return this.api.post<Listing>('/listings', input);
  }

  update(id: string, input: ListingInput): Observable<Listing> {
    return this.api.put<Listing>(`/listings/${id}`, input);
  }

  unpublish(id: string): Observable<void> {
    return this.api.delete<void>(`/listings/${id}`);
  }

  presignPhoto(id: string): Observable<PresignPhotoResult> {
    return this.api.post<PresignPhotoResult>(`/listings/${id}/photos/presign`);
  }

  setPhotos(id: string, keys: string[]): Observable<Listing> {
    return this.api.put<Listing>(`/listings/${id}/photos`, { keys });
  }
}
