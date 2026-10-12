import { DOCUMENT } from '@angular/common';
import { DestroyRef, Injectable, effect, inject, signal } from '@angular/core';
import { AuthService } from '../auth/auth.service';
import { NotificationsApi } from '../../data-access/notifications/notifications.api';

const POLL_MS = 60_000;

/** The unread count on the bell. Polled once a minute while signed in,
 * and straight away when the tab comes back to the foreground (a phone
 * returning from WhatsApp, say), never while the tab is hidden. */
@Injectable({ providedIn: 'root' })
export class NotificationBadgeService {
  private readonly api = inject(NotificationsApi);
  private readonly auth = inject(AuthService);
  private readonly document = inject(DOCUMENT);

  private readonly count = signal(0);
  readonly unread = this.count.asReadonly();

  constructor() {
    const timer = setInterval(() => this.refresh(), POLL_MS);
    const onVisible = () => {
      if (this.document.visibilityState === 'visible') this.refresh();
    };
    this.document.addEventListener('visibilitychange', onVisible);
    inject(DestroyRef).onDestroy(() => {
      clearInterval(timer);
      this.document.removeEventListener('visibilitychange', onVisible);
    });
    // Signing in fetches at once; signing out clears the badge.
    effect(() => {
      if (this.auth.isAuthenticated()) {
        this.refresh();
      } else {
        this.count.set(0);
      }
    });
  }

  /** Re-read the count now (after opening or reading notifications). */
  refresh(): void {
    if (!this.auth.isAuthenticated() || this.document.visibilityState === 'hidden') return;
    this.api.unreadCount().subscribe({
      next: ({ unread }) => this.count.set(unread),
      error: () => {
        // A missed poll is harmless; the next one catches up.
      }
    });
  }

  set(unread: number): void {
    this.count.set(unread);
  }
}
