import { ChangeDetectionStrategy, Component, inject, input, signal } from '@angular/core';
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
 *   by the caller); language, theme, notifications and sign out
 *   unfold from the button beside it.
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
          @if (utilitiesOpen()) {
            <div id="sidebar-utilities" class="flex flex-col gap-0.5 rounded-[14px] bg-white/6 p-1.5">
              <a routerLink="/" [class]="menuRowClass">
                <bt-icon name="search" [size]="17" />
                {{ 'nav.viewSite' | t }}
              </a>
              <a routerLink="/notifications" [class]="menuRowClass">
                <bt-icon name="bell" [size]="17" />
                <span class="flex-1">{{ 'nav.notifications' | t }}</span>
                @if (notifications.unread() > 0) {
                  <span class="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-clay px-1.5 text-[11px] font-bold text-on-clay">{{ notifications.unread() }}</span>
                }
              </a>
              <button type="button" [class]="menuRowClass" (click)="theme.toggle()">
                <bt-icon [name]="theme.isDark() ? 'sun' : 'moon'" [size]="17" />
                {{ 'nav.theme' | t }}
              </button>
              <div class="flex min-h-10 items-center justify-between gap-2 px-3">
                <span class="text-small font-medium text-on-forest-deep/80">{{ 'nav.languageShort' | t }}</span>
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
              </div>
              <button type="button" [class]="menuRowClass" (click)="signOut()">
                <bt-icon name="logout" [size]="17" />
                {{ 'nav.signOut' | t }}
              </button>
            </div>
          }
          <div class="flex items-center gap-1">
            <div class="min-w-0 flex-1"><ng-content select="[sidebarFooter]" /></div>
            <button
              type="button"
              [class]="utilityClass + ' relative shrink-0'"
              aria-controls="sidebar-utilities"
              [attr.aria-expanded]="utilitiesOpen()"
              [attr.aria-label]="'nav.accountOptions' | t"
              [attr.title]="'nav.accountOptions' | t"
              (click)="utilitiesOpen.set(!utilitiesOpen())"
            >
              <bt-icon name="more" [size]="18" />
              @if (!utilitiesOpen() && notifications.unread() > 0) {
                <span class="absolute right-2 top-2 h-2.5 w-2.5 rounded-full bg-clay ring-2" [class]="surface() === 'ink' ? 'ring-ink' : 'ring-forest-deep'" aria-hidden="true"></span>
              }
            </button>
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
  /** Language, theme, notifications and sign-out stay folded behind the
   * account card, so the sidebar's foot reads as the design draws it. */
  protected readonly utilitiesOpen = signal(false);
  protected readonly menuRowClass =
    'flex min-h-10 w-full items-center gap-3 rounded-[10px] px-3 text-left text-small font-medium text-on-forest-deep/80 transition hover:bg-white/8 hover:text-on-forest-deep';
  protected readonly locales = ['fr', 'en'] as const;
  protected readonly utilityClass =
    'inline-flex h-11 w-11 items-center justify-center rounded-full text-on-forest-deep/70 transition hover:bg-white/8 hover:text-on-forest-deep';

  protected signOut(): void {
    this.auth.signOut();
    this.router.navigateByUrl('/');
  }
}
