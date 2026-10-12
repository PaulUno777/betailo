import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { AuthService } from '../../core/auth/auth.service';
import { ThemeService } from '../../core/theme/theme.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { BottomNavComponent } from '../ui/bottom-nav/bottom-nav.component';
import { ButtonComponent } from '../ui/button/button.component';
import { IconButtonComponent } from '../ui/icon-button/icon-button.component';
import { LogoComponent } from '../ui/logo/logo.component';
import { LocaleToggleComponent } from '../ui/locale-toggle/locale-toggle.component';
import { UserMenuComponent } from '../ui/user-menu/user-menu.component';
import { ToastHostComponent } from '../ui/toast/toast-host.component';
import { SiteFooterComponent } from './site-footer.component';

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
 * buttons. */
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
      <header class="sticky top-0 z-30 border-b border-line bg-surface-raised">
        <div class="mx-auto flex h-16 max-w-content-max items-center justify-between px-5 sm:px-18">
          <a routerLink="/"><bt-logo /></a>
          <nav class="hidden items-center gap-6 text-body-strong text-ink sm:flex">
            <a routerLink="/marketplace" routerLinkActive="text-forest">{{ 'nav.marketplace' | t }}</a>
            <a routerLink="/forum" routerLinkActive="text-forest">{{ 'nav.forum' | t }}</a>
            @if (!auth.isAuthenticated()) {
              <a routerLink="/" fragment="how">{{ 'nav.howItWorks' | t }}</a>
              <a routerLink="/" fragment="eleveurs">{{ 'nav.farmers' | t }}</a>
            }
          </nav>
          <div class="flex items-center gap-3">
            <bt-locale-toggle />
            <bt-icon-button [icon]="theme.isDark() ? 'sun' : 'moon'" [label]="'nav.theme' | t" (click)="theme.toggle()" />
            @if (auth.isAuthenticated()) {
              <bt-icon-button icon="bell" [label]="'nav.notifications' | t" />
              <bt-user-menu />
            } @else {
              <bt-button variant="ghost" size="sm" href="/login">{{ 'nav.login' | t }}</bt-button>
              <bt-button variant="solid" size="sm" href="/register">{{ 'nav.register' | t }}</bt-button>
            }
          </div>
        </div>
      </header>
      <main class="flex-1 pb-24 sm:pb-0">
        <router-outlet />
      </main>
      <bt-site-footer />
      <bt-bottom-nav />
      <bt-toast-host />
    </div>
  `
})
export class PublicShellComponent {
  protected readonly auth = inject(AuthService);
  protected readonly theme = inject(ThemeService);
}
