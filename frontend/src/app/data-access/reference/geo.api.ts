import { Injectable, inject } from '@angular/core';
import { ApiClient } from '../../core/http/api-client';

/** Where farms can be: the region lists the API validates against. */
@Injectable({ providedIn: 'root' })
export class GeoApi {
  private readonly api = inject(ApiClient);

  referenceUrl(): string {
    return this.api.url('/reference/geo');
  }
}
