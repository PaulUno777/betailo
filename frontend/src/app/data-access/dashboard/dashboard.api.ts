import { Injectable, inject } from '@angular/core';
import { ApiClient } from '../../core/http/api-client';

@Injectable({ providedIn: 'root' })
export class DashboardApi {
  private readonly api = inject(ApiClient);

  sellerDashboardUrl(farmId: string): string {
    return this.api.url(`/farms/${farmId}/dashboard`);
  }
}
