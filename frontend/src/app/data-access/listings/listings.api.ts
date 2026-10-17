import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { Cycle, Listing, ListingSearchParams } from '../../domain/models';
import { ApiClient, QueryParams } from '../../core/http/api-client';
import { PhotoSizes, PresignedUpload } from '../../core/http/photo-upload.service';

export interface ListingInput {
  readonly cycleId: string;
  readonly title: string;
  readonly description?: string;
  readonly unitPrice: number;
  readonly unit?: string;
}

export interface ListingEditInput {
  readonly title: string;
  readonly description?: string;
  readonly unitPrice: number;
  readonly unit?: string;
  readonly breed?: string;
  readonly ageDays?: number;
  readonly deliveryAvailable?: boolean;
}

/** The sell wizard's one call: the API creates the cycle and a draft
 * listing together, deriving the cycle type and unit from the species. */
export interface QuickListingInput {
  readonly species: string;
  readonly breed?: string;
  readonly ageDays?: number;
  readonly quantity: number;
  readonly unitPrice: number;
  readonly title: string;
  readonly description?: string;
  /** RFC3339; omitted means available now. */
  readonly availableFrom?: string;
  readonly deliveryAvailable: boolean;
}

export interface QuickListingResult {
  readonly cycle: Cycle;
  readonly listing: Listing;
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

  /** The listing page's single payload — see `ListingDetail`. */
  fullDetailUrl(id: string): string {
    return this.api.url(`/listings/${id}/detail`);
  }

  farmListingsUrl(farmId: string): string {
    return this.api.url(`/farms/${farmId}/listings`);
  }

  priceHintUrl(species: string, region?: string): string {
    return this.api.url('/listings/price-hint', { species, region });
  }

  quickCreate(farmId: string, input: QuickListingInput): Observable<QuickListingResult> {
    return this.api.post<QuickListingResult>(`/farms/${farmId}/quick-listings`, input);
  }

  /** Needs at least one photo and an open cycle; the API says which. */
  publish(id: string): Observable<Listing> {
    return this.api.post<Listing>(`/listings/${id}/publish`);
  }

  favoritesUrl(): string {
    return this.api.url('/me/favorites');
  }

  favorite(id: string): Observable<void> {
    return this.api.put<void>(`/listings/${id}/favorite`);
  }

  unfavorite(id: string): Observable<void> {
    return this.api.delete<void>(`/listings/${id}/favorite`);
  }

  create(input: ListingInput): Observable<Listing> {
    return this.api.post<Listing>('/listings', input);
  }

  update(id: string, input: ListingEditInput): Observable<Listing> {
    return this.api.put<Listing>(`/listings/${id}`, input);
  }

  unpublish(id: string): Observable<void> {
    return this.api.delete<void>(`/listings/${id}`);
  }

  presignPhoto(id: string, sizes: PhotoSizes): Observable<PresignedUpload> {
    return this.api.post<PresignedUpload>(`/listings/${id}/photos/presign`, sizes);
  }

  setPhotos(id: string, keys: string[]): Observable<Listing> {
    return this.api.put<Listing>(`/listings/${id}/photos`, { keys });
  }
}
