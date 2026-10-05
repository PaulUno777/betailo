import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { AuthService } from '../../core/auth/auth.service';
import { ThemeService } from '../../core/theme/theme.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { BottomNavCenter, BottomNavComponent, BottomNavItem } from '../ui/bottom-nav/bottom-nav.component';
import { ButtonComponent } from '../ui/button/button.component';
import { IconButtonComponent } from '../ui/icon-button/icon-button.component';
import { LogoComponent } from '../ui/logo/logo.component';
import { LocaleToggleComponent } from '../ui/locale-toggle/locale-toggle.component';
import { UserMenuComponent } from '../ui/user-menu/user-menu.component';
import { ToastHostComponent } from '../ui/toast/toast-host.component';
import { SiteFooterComponent } from './site-footer.component';

const BUYER_TABS: readonly BottomNavItem[] = [
  { kind: 'link', labelKey: 'bottomNav.home', icon: 'home', route: '/', exact: true },
  { kind: 'link', labelKey: 'bottomNav.market', icon: 'search', route: '/marketplace' },
  { kind: 'link', labelKey: 'bottomNav.reservations', icon: 'calendar', route: '/account/reservations' },
  { kind: 'link', labelKey: 'bottomNav.profile', icon: 'user', route: '/account' }
];

/** The shell for every public-site route (Home, Marketplace, Listing,
 * Auth, Forum): desktop header nav, the router outlet, and the mobile
 * `BottomNav` — one responsive page per route, not a separate mobile
 * route, per the design's `M_*` boards being viewport variants.
 *
 * The header's right side stays short regardless of who's looking: an
 * anonymous visitor gets the marketing anchors (how it works / farmers)
 * and sign-in actions; a signed-in user gets locale, theme, notifications
 * and one `UserMenu` trigger — their role destination, account and
 * sign-out all live inside that one dropdown rather than as separate
 * buttons.
 *
 * Below `sm` the header shrinks to what a phone needs (logo, then either
 * the bell and account menu or a single sign-in button); theme and
 * language move into the account menu, and the bottom nav takes over
 * primary navigation. */
@Component({
  selector: 'bt-public-shell',
  imports: [
    RouterLink,
    RouterLinkActive,
    RouterOutlet,
    BottomNavComponent,
    ButtonComponent,
    IconButtonComponent,
    LogoComponent,
    LocaleToggleComponent,
    UserMenuComponent,
    ToastHostComponent,
    SiteFooterComponent,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex min-h-full flex-col bg-surface">
      <header
        class="sticky top-0 z-30 border-b border-line bg-surface-raised/95 backdrop-blur"
        style="padding-top: env(safe-area-inset-top)"
      >
        <div class="mx-auto flex h-14 max-w-content-max items-center justify-between gap-3 px-4 sm:h-16 sm:px-18">
          <a routerLink="/" class="shrink-0"><bt-logo /></a>
          <nav class="hidden items-center gap-6 text-body-strong text-ink sm:flex">
            <a routerLink="/marketplace" routerLinkActive="text-forest">{{ 'nav.marketplace' | t }}</a>
            <a routerLink="/forum" routerLinkActive="text-forest">{{ 'nav.forum' | t }}</a>
            @if (!auth.isAuthenticated()) {
              <a routerLink="/" fragment="how">{{ 'nav.howItWorks' | t }}</a>
              <a routerLink="/" fragment="eleveurs">{{ 'nav.farmers' | t }}</a>
            }
          </nav>
          <div class="flex items-center gap-1 sm:gap-3">
            <bt-locale-toggle class="hidden sm:block" />
            <bt-icon-button
              class="hidden sm:inline-flex"
              [icon]="theme.isDark() ? 'sun' : 'moon'"
              [label]="'nav.theme' | t"
              (click)="theme.toggle()"
            />
            @if (auth.isAuthenticated()) {
              <bt-icon-button icon="bell" [label]="'nav.notifications' | t" />
              <bt-user-menu />
            } @else {
              <bt-locale-toggle class="sm:hidden" />
              <bt-button variant="ghost" size="sm" href="/login">{{ 'nav.login' | t }}</bt-button>
              <bt-button class="hidden sm:inline-flex" variant="solid" size="sm" href="/register">{{ 'nav.register' | t }}</bt-button>
            }
          </div>
        </div>
      </header>
      <main class="flex-1">
        <router-outlet />
      </main>
      <bt-site-footer class="pb-[calc(var(--height-bottom-nav)+env(safe-area-inset-bottom))] sm:pb-0" />
      <bt-bottom-nav [items]="tabs" [center]="center()" />
      <bt-toast-host />
    </div>
  `
})
export class PublicShellComponent {
  protected readonly auth = inject(AuthService);
  protected readonly theme = inject(ThemeService);

  protected readonly tabs = BUYER_TABS;

  /** "Publier": farmers go straight to the listing form; anyone else
   * lands on /sell, which walks them through becoming a seller (or to
   * sign-in first, via its auth guard). */
  protected readonly center = computed<BottomNavCenter>(() => ({
    labelKey: 'bottomNav.publish',
    icon: 'plus',
    route: this.auth.role() === 'farmer' ? '/portal/listings/new' : '/sell'
  }));
}
