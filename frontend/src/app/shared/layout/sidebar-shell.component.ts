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
import { MoreSheetComponent, MoreSheetLink } from './more-sheet.component';

/** The shell shared by the farmer portal and admin. From `sm` up: the
 * `SidebarNav` (on whichever surface the caller passes — that's what
 * makes admin visually distinct, not a different component) plus a slim
 * top bar for locale/theme/account. Below `sm`: a compact header and the
 * section's own `BottomNav`, with a "Plus" sheet for the destinations
 * that don't fit in five tabs — a farmer or admin on a phone keeps the
 * whole section in reach, never the buyer's tabs. */
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
    UserMenuComponent,
    NotificationBellComponent,
    ToastHostComponent,
    MoreSheetComponent,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex min-h-full">
      <div class="sticky top-0 hidden h-dvh shrink-0 sm:block">
        <bt-sidebar-nav [items]="items()" [surface]="surface()" />
      </div>
      <div class="flex min-w-0 flex-1 flex-col">
        <header
          class="sticky top-0 z-30 flex h-14 items-center justify-between gap-3 border-b border-line bg-surface-raised/95 px-4 backdrop-blur sm:h-16 sm:justify-end sm:px-8"
          style="padding-top: env(safe-area-inset-top)"
        >
          <a [routerLink]="homeRoute()" class="sm:hidden"><bt-logo /></a>
          <div class="flex items-center gap-2 sm:gap-3">
            <bt-locale-toggle class="hidden sm:block" />
            <bt-icon-button
              class="hidden sm:inline-flex"
              [icon]="theme.isDark() ? 'sun' : 'moon'"
              [label]="'nav.theme' | t"
              (click)="theme.toggle()"
            />
            <bt-notification-bell />
            <bt-user-menu />
          </div>
        </header>
        <main class="flex-1 bg-surface pb-[calc(var(--height-bottom-nav)+env(safe-area-inset-bottom)+16px)] sm:pb-0">
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
  readonly surface = input<SidebarNavSurface>('forest-deep');
  readonly homeRoute = input.required<string>();
  readonly bottomItems = input.required<readonly BottomNavItem[]>();
  readonly center = input<BottomNavCenter>();
  readonly moreLinks = input.required<readonly MoreSheetLink[]>();

  protected readonly moreOpen = signal(false);
}
