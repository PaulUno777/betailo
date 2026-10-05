import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiClient } from '../../core/http/api-client';

@Injectable({ providedIn: 'root' })
export class NotificationsApi {
  private readonly api = inject(ApiClient);

  /** A `NotificationList`: the latest notifications plus the unread count. */
  listUrl(): string {
    return this.api.url('/notifications');
  }

  unreadCount(): Observable<{ unread: number }> {
    return this.api.get<{ unread: number }>('/notifications/unread-count');
  }

  markRead(id: string): Observable<void> {
    return this.api.post<void>(`/notifications/${id}/read`);
  }

  markAllRead(): Observable<void> {
    return this.api.post<void>('/notifications/read-all');
  }
}
