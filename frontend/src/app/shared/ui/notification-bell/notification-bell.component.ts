import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { NotificationBadgeService } from '../../../core/notifications/notification-badge.service';
import { LocaleService } from '../../../core/i18n/locale.service';
import { IconComponent } from '../icon/icon.component';

/** The bell in every signed-in header: a link to the notifications
 * page, badged with the unread count. */
@Component({
  selector: 'bt-notification-bell',
  imports: [RouterLink, IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a
      routerLink="/notifications"
      [attr.aria-label]="label()"
      class="relative inline-flex h-11 w-11 items-center justify-center rounded-full text-ink transition hover:bg-surface-sunken"
    >
      <bt-icon name="bell" [size]="22" />
      @if (badge.unread() > 0) {
        <span
          class="absolute right-1 top-1 flex h-5 min-w-5 items-center justify-center rounded-full bg-clay px-1 text-[11px] font-bold text-on-clay ring-2 ring-surface-raised"
          aria-hidden="true"
        >
          {{ badge.unread() > 9 ? '9+' : badge.unread() }}
        </span>
      }
    </a>
  `
})
export class NotificationBellComponent {
  protected readonly badge = inject(NotificationBadgeService);
  private readonly locale = inject(LocaleService);

  protected readonly label = computed(() =>
    this.badge.unread() > 0
      ? this.locale.t('notifications.bellUnread', { count: this.badge.unread() })
      : this.locale.t('nav.notifications')
  );
}
