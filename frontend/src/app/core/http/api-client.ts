import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';

export type QueryParams = Record<string, string | number | boolean | undefined | null>;

function buildParams(query?: QueryParams): HttpParams {
  let params = new HttpParams();
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== undefined && value !== null) {
      params = params.set(key, value);
    }
  }
  return params;
}

/** The one place that knows the API's base URL and joins it with a path —
 * every `data-access` adapter depends on this, never on `HttpClient` or a
 * hardcoded origin directly (DIP). `url()` is for the handful of GET reads
 * built with `httpResource()`, which needs a plain URL rather than an
 * Observable. */
@Injectable({ providedIn: 'root' })
export class ApiClient {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = inject(API_BASE_URL);

  url(path: string, query?: QueryParams): string {
    const params = buildParams(query);
    const qs = params.toString();
    return qs ? `${this.baseUrl}${path}?${qs}` : `${this.baseUrl}${path}`;
  }

  get<T>(path: string, query?: QueryParams): Observable<T> {
    return this.http.get<T>(`${this.baseUrl}${path}`, { params: buildParams(query) });
  }

  /** A file the API builds (a CSV export), as the session's own request. */
  download(path: string): Observable<Blob> {
    return this.http.get(this.url(path), { responseType: 'blob' });
  }

  post<T>(path: string, body?: unknown): Observable<T> {
    return this.http.post<T>(`${this.baseUrl}${path}`, body ?? {});
  }

  put<T>(path: string, body?: unknown): Observable<T> {
    return this.http.put<T>(`${this.baseUrl}${path}`, body ?? {});
  }

  patch<T>(path: string, body?: unknown): Observable<T> {
    return this.http.patch<T>(`${this.baseUrl}${path}`, body ?? {});
  }

  delete<T>(path: string): Observable<T> {
    return this.http.delete<T>(`${this.baseUrl}${path}`);
  }
}
