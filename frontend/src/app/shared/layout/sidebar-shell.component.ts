import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { Router, RouterLink, RouterOutlet } from '@angular/router';
import { AuthService } from '../../core/auth/auth.service';
import { LocaleService } from '../../core/i18n/locale.service';
import { NotificationBadgeService } from '../../core/notifications/notification-badge.service';
import { ThemeService } from '../../core/theme/theme.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { SidebarNavComponent, SidebarNavItem, SidebarNavSurface } from '../ui/sidebar-nav/sidebar-nav.component';
import { BottomNavCenter, BottomNavComponent, BottomNavItem } from '../ui/bottom-nav/bottom-nav.component';
import { IconComponent } from '../ui/icon/icon.component';
import { LogoComponent } from '../ui/logo/logo.component';
import { NotificationBellComponent } from '../ui/notification-bell/notification-bell.component';
import { ToastHostComponent } from '../ui/toast/toast-host.component';
import { MoreSheetComponent, MoreSheetLink } from './more-sheet.component';

/** The shell shared by the seller space and the back office.
 *
 * - Desktop (`lg` and up): the design's sidebar, full height, and
 *   nothing above the page: each page opens with its own header
 *   (eyebrow, title, its one primary action). What a top bar would
 *   hold lives at the foot of the sidebar: who is signed in (projected
 *   by the caller), then language, theme, notifications and sign out.
 * - Phone and tablet: a compact header (logo, a chip naming the space
 *   so people always know which side they're on, the bell) and the
 *   section's own labelled bottom nav, with a "Plus" sheet for what
 *   doesn't fit in five tabs. */
@Component({
  selector: 'bt-sidebar-shell',
  imports: [
    RouterLink,
    RouterOutlet,
    SidebarNavComponent,
    BottomNavComponent,
    IconComponent,
    LogoComponent,
    NotificationBellComponent,
    ToastHostComponent,
    MoreSheetComponent,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex min-h-full">
      <div class="sticky top-0 hidden h-dvh shrink-0 lg:block">
        <bt-sidebar-nav [items]="items()" [surface]="surface()" [badgeLabel]="spaceLabelKey() | t" [label]="navLabelKey() | t">
          <ng-content select="[sidebarTop]" ngProjectAs="[navTop]" />
          <ng-content select="[sidebarFooter]" />
          <div class="flex items-center justify-between gap-1 px-1">
            <div role="group" [attr.aria-label]="'nav.language' | t" class="flex items-center gap-0.5 rounded-full bg-white/8 p-0.5">
              @for (l of locales; track l) {
                <button
                  type="button"
                  [attr.aria-pressed]="locale.locale() === l"
                  (click)="locale.setLocale(l)"
                  class="h-8 rounded-full px-2.5 text-caption font-semibold uppercase transition"
                  [class]="locale.locale() === l ? 'bg-on-forest-deep text-ink' : 'text-on-forest-deep/70 hover:text-on-forest-deep'"
                >
                  {{ l }}
                </button>
              }
            </div>
            <div class="flex items-center">
              <button type="button" [class]="utilityClass" [attr.aria-label]="'nav.theme' | t" [attr.title]="'nav.theme' | t" (click)="theme.toggle()">
                <bt-icon [name]="theme.isDark() ? 'sun' : 'moon'" [size]="18" />
              </button>
              <a routerLink="/notifications" [class]="utilityClass + ' relative'" [attr.aria-label]="bellLabel()" [attr.title]="'nav.notifications' | t">
                <bt-icon name="bell" [size]="18" />
                @if (notifications.unread() > 0) {
                  <span class="absolute right-2 top-2 h-2.5 w-2.5 rounded-full bg-clay ring-2" [class]="surface() === 'ink' ? 'ring-ink' : 'ring-forest-deep'" aria-hidden="true"></span>
                }
              </a>
              <button type="button" [class]="utilityClass" [attr.aria-label]="'nav.signOut' | t" [attr.title]="'nav.signOut' | t" (click)="signOut()">
                <bt-icon name="logout" [size]="18" />
              </button>
            </div>
          </div>
        </bt-sidebar-nav>
      </div>
      <div class="flex min-w-0 flex-1 flex-col">
        <header
          class="sticky top-0 z-30 flex h-14 items-center justify-between gap-3 border-b border-line bg-surface-raised/95 px-4 backdrop-blur lg:hidden"
          style="padding-top: env(safe-area-inset-top)"
        >
          <a [routerLink]="homeRoute()" class="flex items-center gap-2">
            <bt-logo />
            <span class="rounded-full bg-forest-soft px-2 py-0.5 text-caption font-semibold text-on-forest-soft">{{ spaceLabelKey() | t }}</span>
          </a>
          <bt-notification-bell />
        </header>
        <main [class]="'flex-1 pb-[calc(var(--height-bottom-nav)+env(safe-area-inset-bottom)+16px)] lg:pb-0 ' + (surface() === 'ink' ? 'bg-surface-backoffice' : 'bg-surface')">
          <ng-content select="[shellBanner]" />
          <router-outlet />
        </main>
      </div>
      <bt-bottom-nav [items]="bottomItems()" [center]="center()" (more)="moreOpen.set(true)" />
      <bt-more-sheet [open]="moreOpen()" [links]="moreLinks()" (closed)="moreOpen.set(false)" />
      <bt-toast-host />
    </div>
  `
})
export class SidebarShellComponent {
  protected readonly theme = inject(ThemeService);
  protected readonly locale = inject(LocaleService);
  protected readonly notifications = inject(NotificationBadgeService);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  readonly items = input.required<SidebarNavItem[]>();
  /** Names the space on the phone header chip and next to the sidebar
   * logo ("Vendeur", "Staff"). */
  readonly spaceLabelKey = input.required<string>();
  /** The sidebar navigation's accessible name. */
  readonly navLabelKey = input.required<string>();
  readonly surface = input<SidebarNavSurface>('forest-deep');
  readonly homeRoute = input.required<string>();
  readonly bottomItems = input.required<readonly BottomNavItem[]>();
  readonly center = input<BottomNavCenter>();
  readonly moreLinks = input.required<readonly MoreSheetLink[]>();

  protected readonly moreOpen = signal(false);
  protected readonly locales = ['fr', 'en'] as const;
  protected readonly utilityClass =
    'inline-flex h-11 w-11 items-center justify-center rounded-full text-on-forest-deep/70 transition hover:bg-white/8 hover:text-on-forest-deep';

  protected readonly bellLabel = computed(() =>
    this.notifications.unread() > 0
      ? this.locale.t('notifications.bellUnread', { count: this.notifications.unread() })
      : this.locale.t('nav.notifications')
  );

  protected signOut(): void {
    this.auth.signOut();
    this.router.navigateByUrl('/');
  }
}
