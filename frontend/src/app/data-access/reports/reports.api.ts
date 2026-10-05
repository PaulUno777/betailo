import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ReportTargetType } from '../../domain/enums';
import { Report } from '../../domain/models';
import { ApiClient } from '../../core/http/api-client';

export interface CreateReportInput {
  readonly targetId: string;
  readonly targetType: ReportTargetType;
  readonly reason: string;
}

@Injectable({ providedIn: 'root' })
export class ReportsApi {
  private readonly api = inject(ApiClient);

  create(input: CreateReportInput): Observable<Report> {
    return this.api.post<Report>('/reports', input);
  }
}
