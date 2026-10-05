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

  delete(id: string): Observable<void> {
    return this.api.delete<void>(`/cycles/${id}`);
  }
}
