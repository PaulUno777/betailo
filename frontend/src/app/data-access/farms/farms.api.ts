import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { Farm } from '../../domain/models';
import { ApiClient } from '../../core/http/api-client';

export interface FarmInput {
  readonly name: string;
  readonly location: string;
  readonly country?: string;
  readonly region?: string;
  readonly species?: string[];
  readonly description?: string;
  readonly contactPhone?: string;
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

  create(input: FarmInput): Observable<Farm> {
    return this.api.post<Farm>('/farms', input);
  }

  update(id: string, input: FarmInput): Observable<Farm> {
    return this.api.put<Farm>(`/farms/${id}`, input);
  }

  delete(id: string): Observable<void> {
    return this.api.delete<void>(`/farms/${id}`);
  }
}
