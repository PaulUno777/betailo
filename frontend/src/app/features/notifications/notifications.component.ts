import { ChangeDetectionStrategy, Component, effect, inject } from '@angular/core';
import { Router } from '@angular/router';
import { AppNotification, NotificationList } from '../../domain/models';
import { NotificationsApi } from '../../data-access/notifications/notifications.api';
import { NotificationBadgeService } from '../../core/notifications/notification-badge.service';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { IconName } from '../../shared/ui/icon/icon-paths';
import { ButtonComponent } from '../../shared/ui/button/button.component';

const ICONS: Readonly<Record<string, { icon: IconName; tone: string }>> = {
  reservation_created: { icon: 'calendar', tone: 'bg-sky-soft text-sky' },
  reservation_paid: { icon: 'wallet', tone: 'bg-forest-soft text-forest' },
  reservation_cancelled: { icon: 'x', tone: 'bg-stone-soft text-stone' },
  reservation_expired: { icon: 'clock', tone: 'bg-stone-soft text-stone' },
  reservation_delivered: { icon: 'check', tone: 'bg-clay-soft text-clay-ink' },
  escrow_released: { icon: 'lock', tone: 'bg-forest-soft text-forest' },
  escrow_auto_released: { icon: 'lock', tone: 'bg-forest-soft text-forest' },
  escrow_refunded: { icon: 'wallet', tone: 'bg-sky-soft text-sky' },
  dispute_opened: { icon: 'alert', tone: 'bg-alert-soft text-alert' },
  review_received: { icon: 'star', tone: 'bg-amber-soft text-amber' },
  cycle_update: { icon: 'cycle', tone: 'bg-forest-soft text-forest' },
  forum_reply: { icon: 'message', tone: 'bg-plum-soft text-plum' },
  best_answer: { icon: 'check', tone: 'bg-plum-soft text-plum' }
};
const DEFAULT_ICON = { icon: 'bell' as IconName, tone: 'bg-surface-sunken text-ink-muted' };

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** The notification inbox (the bell's page): newest first, unread ones
 * marked, each opening where it's about. Text comes from a localized
 * template per type filled with the notification's params; types this
 * client doesn't know fall back to the server's own message. */
@Component({
  selector: 'bt-notifications',
  imports: [IconComponent, ButtonComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-2xl px-4 pb-8 pt-5 sm:px-8 lg:py-10">
      <header class="flex items-center justify-between gap-3">
        <h1 class="text-heading-1">{{ 'notifications.title' | t }}</h1>
        @if ((list.value()?.unread ?? 0) > 0) {
          <bt-button variant="ghost" size="sm" (click)="markAll()">{{ 'notifications.markAll' | t }}</bt-button>
        }
      </header>

      <div class="mt-5 flex flex-col">
        @if (list.value(); as l) {
          @for (n of l.items; track n.id) {
            <button
              type="button"
              (click)="open(n)"
              class="flex min-h-[72px] items-start gap-3 border-b border-line px-1 py-3 text-left transition active:bg-surface-sunken"
            >
              <span class="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-full" [class]="iconOf(n).tone">
                <bt-icon [name]="iconOf(n).icon" [size]="18" />
              </span>
              <span class="flex min-w-0 flex-1 flex-col gap-0.5">
                <span [class]="n.read ? 'text-body text-ink-muted' : 'text-body-strong text-ink'">{{ textOf(n) }}</span>
                <span class="text-caption text-ink-muted">{{ ago(n.createdAt) }}</span>
              </span>
              @if (!n.read) {
                <span class="mt-2 h-2.5 w-2.5 shrink-0 rounded-full bg-clay" [attr.aria-label]="'notifications.unread' | t"></span>
              }
            </button>
          } @empty {
            <div class="flex flex-col items-center gap-3 rounded-xl border border-dashed border-line-strong px-6 py-14 text-center">
              <bt-icon name="bell" [size]="30" class="text-ink-muted" />
              <p class="text-body text-ink-muted">{{ 'notifications.empty' | t }}</p>
            </div>
          }
        } @else if (list.isLoading()) {
          @for (i of [1, 2, 3, 4]; track i) {
            <div class="my-2 h-16 animate-pulse rounded-lg bg-surface-sunken"></div>
          }
        } @else if (list.error()) {
          <div class="flex flex-col items-center gap-3 py-10">
            <bt-button variant="outline" (click)="list.reload()">{{ 'common.retry' | t }}</bt-button>
          </div>
        }
      </div>
    </div>
  `
})
export class NotificationsComponent {
  private readonly api = inject(NotificationsApi);
  private readonly badge = inject(NotificationBadgeService);
  private readonly locale = inject(LocaleService);
  private readonly router = inject(Router);

  protected readonly list = apiResource<NotificationList>(() => this.api.listUrl());

  constructor() {
    // Keep the bell in sync with what this page just loaded.
    effect(() => {
      const unread = this.list.value()?.unread;
      if (unread !== undefined) this.badge.set(unread);
    });
  }

  protected iconOf(n: AppNotification) {
    return ICONS[n.type] ?? DEFAULT_ICON;
  }

  protected textOf(n: AppNotification): string {
    const key = `notifications.types.${n.type}`;
    const text = this.locale.t(key, n.params ?? {});
    return text === key ? n.message : text;
  }

  protected ago(iso: string): string {
    const diff = new Date(iso).getTime() - Date.now();
    const rtf = new Intl.RelativeTimeFormat(this.locale.locale() === 'fr' ? 'fr' : 'en', { numeric: 'auto' });
    if (Math.abs(diff) < HOUR) return rtf.format(Math.round(diff / MINUTE), 'minute');
    if (Math.abs(diff) < DAY) return rtf.format(Math.round(diff / HOUR), 'hour');
    return rtf.format(Math.round(diff / DAY), 'day');
  }

  protected open(n: AppNotification): void {
    if (!n.read) {
      this.api.markRead(n.id).subscribe({ next: () => this.badge.refresh(), error: () => undefined });
    }
    if (n.link) {
      this.router.navigateByUrl(n.link);
    } else {
      this.list.reload();
    }
  }

  protected markAll(): void {
    this.api.markAllRead().subscribe({
      next: () => {
        this.badge.set(0);
        this.list.reload();
      },
      error: () => undefined
    });
  }
}
