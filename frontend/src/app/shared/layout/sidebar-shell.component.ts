import { ChangeDetectionStrategy, Component, inject, input, signal } from '@angular/core';
import { RouterLink, RouterOutlet } from '@angular/router';
import { ThemeService } from '../../core/theme/theme.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { SidebarNavComponent, SidebarNavItem, SidebarNavSurface } from '../ui/sidebar-nav/sidebar-nav.component';
import { BottomNavCenter, BottomNavComponent, BottomNavItem } from '../ui/bottom-nav/bottom-nav.component';
import { IconButtonComponent } from '../ui/icon-button/icon-button.component';
import { LocaleToggleComponent } from '../ui/locale-toggle/locale-toggle.component';
import { LogoComponent } from '../ui/logo/logo.component';
import { UserMenuComponent } from '../ui/user-menu/user-menu.component';
import { NotificationBellComponent } from '../ui/notification-bell/notification-bell.component';
import { ToastHostComponent } from '../ui/toast/toast-host.component';
import { ModeSwitchComponent } from '../ui/mode-switch/mode-switch.component';
import { MoreSheetComponent, MoreSheetLink } from './more-sheet.component';

/** The shell shared by the seller space and admin.
 *
 * - Desktop (`lg` and up): the `SidebarNav` on the caller's surface
 *   (that's what makes admin look distinct, not a different component),
 *   with the way out (Acheter/Vendre switch, back to the market, account)
 *   pinned to its bottom, and a slim top bar for locale, theme,
 *   notifications and the account menu.
 * - Phone and tablet: a compact header (logo, a chip naming the space so
 *   the farmer always knows which side they're on, the bell) and the
 *   section's own labelled bottom nav, with a "Plus" sheet for what
 *   doesn't fit in five tabs, including the way back to buying. */
@Component({
  selector: 'bt-sidebar-shell',
  imports: [
    RouterLink,
    RouterOutlet,
    SidebarNavComponent,
    BottomNavComponent,
    IconButtonComponent,
    LocaleToggleComponent,
    LogoComponent,
    ModeSwitchComponent,
    UserMenuComponent,
    NotificationBellComponent,
    ToastHostComponent,
    MoreSheetComponent,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex min-h-full">
      <div class="sticky top-0 hidden h-dvh shrink-0 lg:block">
        <bt-sidebar-nav [items]="items()" [footer]="footer()" [surface]="surface()" [badgeLabel]="spaceLabelKey() | t">
          @if (modeSwitch()) {
            <bt-mode-switch class="mb-2 px-1" mode="sell" tone="dark" />
          }
        </bt-sidebar-nav>
      </div>
      <div class="flex min-w-0 flex-1 flex-col">
        <header
          class="sticky top-0 z-30 flex h-14 items-center justify-between gap-3 border-b border-line bg-surface-raised/95 px-4 backdrop-blur lg:h-16 lg:justify-end lg:px-8"
          style="padding-top: env(safe-area-inset-top)"
        >
          <a [routerLink]="homeRoute()" class="flex items-center gap-2 lg:hidden">
            <bt-logo />
            <span class="rounded-full bg-forest-soft px-2 py-0.5 text-caption font-semibold text-forest-deep">{{ spaceLabelKey() | t }}</span>
          </a>
          <div class="flex items-center gap-2 lg:gap-3">
            <bt-locale-toggle class="hidden lg:block" />
            <bt-icon-button
              class="hidden lg:inline-flex"
              [icon]="theme.isDark() ? 'sun' : 'moon'"
              [label]="'nav.theme' | t"
              (click)="theme.toggle()"
            />
            <bt-notification-bell />
            <bt-user-menu class="hidden lg:block" />
          </div>
        </header>
        <main class="flex-1 bg-surface pb-[calc(var(--height-bottom-nav)+env(safe-area-inset-bottom)+16px)] lg:pb-0">
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

  readonly items = input.required<SidebarNavItem[]>();
  readonly footer = input<SidebarNavItem[]>([]);
  /** Names the space on the phone header chip and next to the sidebar
   * logo ("Vendeur", "Admin"). */
  readonly spaceLabelKey = input.required<string>();
  /** Show the Acheter/Vendre switch above the sidebar footer. */
  readonly modeSwitch = input(false);
  readonly surface = input<SidebarNavSurface>('forest-deep');
  readonly homeRoute = input.required<string>();
  readonly bottomItems = input.required<readonly BottomNavItem[]>();
  readonly center = input<BottomNavCenter>();
  readonly moreLinks = input.required<readonly MoreSheetLink[]>();

  protected readonly moreOpen = signal(false);
}
