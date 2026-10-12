import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { CycleStatus } from '../../domain/enums';
import { Cycle } from '../../domain/models';
import { ApiClient } from '../../core/http/api-client';

export interface CreateCycleInput {
  readonly farmId: string;
  readonly species: string;
  readonly type: string;
  readonly capacity: number;
  readonly startsAt: string;
  readonly endsAt?: string;
}

export interface UpdateCycleInput {
  readonly startsAt: string;
  readonly endsAt?: string;
  readonly capacity: number;
  readonly minPerReservation: number;
  readonly maxPerReservation: number;
}

/** A carnet de santé entry: "Vaccin Newcastle", with optional detail. */
export interface HealthEventInput {
  readonly title: string;
  readonly detail?: string;
  readonly date: string;
}

@Injectable({ providedIn: 'root' })
export class CyclesApi {
  private readonly api = inject(ApiClient);

  detailUrl(id: string): string {
    return this.api.url(`/cycles/${id}`);
  }

  byFarmUrl(farmId: string): string {
    return this.api.url(`/farms/${farmId}/cycles`);
  }

  create(input: CreateCycleInput): Observable<Cycle> {
    return this.api.post<Cycle>('/cycles', input);
  }

  updateStatus(id: string, status: CycleStatus): Observable<Cycle> {
    return this.api.patch<Cycle>(`/cycles/${id}/status`, { status });
  }

  /** Capacity can't drop below what's already reserved: the API checks. */
  update(id: string, input: UpdateCycleInput): Observable<Cycle> {
    return this.api.put<Cycle>(`/cycles/${id}`, input);
  }

  addHealthEvent(id: string, input: HealthEventInput): Observable<unknown> {
    return this.api.post<unknown>(`/cycles/${id}/health-events`, input);
  }

  delete(id: string): Observable<void> {
    return this.api.delete<void>(`/cycles/${id}`);
  }
}
