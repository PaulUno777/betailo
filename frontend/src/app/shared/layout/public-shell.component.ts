import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { filter, map, startWith } from 'rxjs';
import { AuthService } from '../../core/auth/auth.service';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { ThemeService } from '../../core/theme/theme.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { BottomNavCenter, BottomNavComponent, BottomNavItem } from '../ui/bottom-nav/bottom-nav.component';
import { HeaderSearchComponent } from '../ui/header-search/header-search.component';
import { IconComponent } from '../ui/icon/icon.component';
import { ModeSwitchComponent } from '../ui/mode-switch/mode-switch.component';
import { NavCountsService } from '../../core/navigation/nav-counts.service';
import { SPECIES_LABEL_KEYS, SPECIES_VALUES } from '../../domain/reference/species';
import { ButtonComponent } from '../ui/button/button.component';
import { IconButtonComponent } from '../ui/icon-button/icon-button.component';
import { LogoComponent } from '../ui/logo/logo.component';
import { LocaleToggleComponent } from '../ui/locale-toggle/locale-toggle.component';
import { UserMenuComponent } from '../ui/user-menu/user-menu.component';
import { NotificationBellComponent } from '../ui/notification-bell/notification-bell.component';
import { ToastHostComponent } from '../ui/toast/toast-host.component';
import { SiteFooterComponent } from './site-footer.component';

/** Desktop's second header row: one chip per species, like the
 * category bar under every marketplace's search. */
const CATEGORY_CHIPS = SPECIES_VALUES.map((value) => ({ value, labelKey: SPECIES_LABEL_KEYS[value] }));

/** The shell for every public-site route (home, marketplace, listing,
 * auth, forum, account), laid out like the marketplaces our users
 * already know (Jumia, Leboncoin):
 *
 * - Desktop (`lg` and up): logo, a wide search field, a "Vendre" button,
 *   then sign-in, or the bell, the Acheter/Vendre switch (farmers) and
 *   the account menu. A second row lists the species as category chips.
 * - Phone and tablet: a compact header (logo, then the bell or
 *   "Se connecter") and five labelled bottom tabs: Accueil, Acheter,
 *   the raised Vendre, Mes achats (badged when something waits on the
 *   buyer) and Compte, which holds everything else, so there is no
 *   avatar menu to discover. Home and the marketplace carry their own
 *   search field.
 *
 * Focus pages (`data: { focus: true }`: listing detail, checkout, the
 * sell wizard) drop the phone header and tabs for their own action bar. */
@Component({
  selector: 'bt-public-shell',
  imports: [
    RouterLink,
    RouterLinkActive,
    RouterOutlet,
    BottomNavComponent,
    HeaderSearchComponent,
    IconComponent,
    ModeSwitchComponent,
    ButtonComponent,
    IconButtonComponent,
    LogoComponent,
    LocaleToggleComponent,
    UserMenuComponent,
    NotificationBellComponent,
    ToastHostComponent,
    SiteFooterComponent,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex min-h-full flex-col bg-surface">
      <header
        class="sticky top-0 z-30 border-b border-line bg-surface-raised/95 backdrop-blur"
        [class.max-lg:hidden]="focus()"
        style="padding-top: env(safe-area-inset-top)"
      >
        <div class="mx-auto flex h-14 max-w-content-max items-center gap-3 px-4 lg:h-[72px] lg:gap-6 lg:px-10">
          <a routerLink="/" class="shrink-0"><bt-logo /></a>
          <div class="hidden max-w-xl flex-1 lg:block">
            <bt-header-search />
          </div>
          <div class="ml-auto flex items-center gap-1 lg:gap-3">
            <bt-button class="hidden lg:inline-flex" variant="primary" size="sm" [href]="sellRoute">
              <bt-icon name="plus" [size]="16" />
              <!-- A seller already has the switch's "Vendre" beside it. -->
              {{ (seller.hasFarm() ? 'nav.newListing' : 'nav.sell') | t }}
            </bt-button>
            @if (auth.isAuthenticated()) {
              <bt-mode-switch class="hidden lg:block" mode="buy" />
              <bt-notification-bell />
              <bt-user-menu class="hidden lg:block" />
            } @else {
              <bt-locale-toggle class="lg:hidden" />
              <bt-button variant="ghost" size="sm" href="/login">{{ 'nav.login' | t }}</bt-button>
              <bt-button class="hidden lg:inline-flex" variant="solid" size="sm" href="/register">{{ 'nav.register' | t }}</bt-button>
            }
          </div>
        </div>
        <nav [attr.aria-label]="'nav.categories' | t" class="hidden border-t border-line lg:block">
          <div class="mx-auto flex h-11 max-w-content-max items-center gap-1 px-10 text-small">
            <a
              routerLink="/marketplace"
              [class]="chipClass(onMarket() && !activeSpecies())"
            >{{ 'nav.allAnimals' | t }}</a>
            @for (c of categories; track c.value) {
              <a routerLink="/marketplace" [queryParams]="{ species: c.value }" [class]="chipClass(activeSpecies() === c.value)">{{ c.labelKey | t }}</a>
            }
            <span class="ml-auto"></span>
            <a routerLink="/forum" routerLinkActive="text-forest" [class]="chipClass(false)">
              <bt-icon name="message" [size]="16" />
              {{ 'nav.forum' | t }}
            </a>
            <bt-locale-toggle />
            <bt-icon-button [icon]="theme.isDark() ? 'sun' : 'moon'" [label]="'nav.theme' | t" (click)="theme.toggle()" />
          </div>
        </nav>
      </header>
      <main class="flex-1">
        <router-outlet />
      </main>
      @if (!focus()) {
        <bt-site-footer class="pb-[calc(var(--height-bottom-nav)+env(safe-area-inset-bottom))] lg:pb-0" />
        <bt-bottom-nav [items]="tabs()" [center]="center" />
      } @else {
        <bt-site-footer class="hidden lg:block" />
      }
      <bt-toast-host />
    </div>
  `
})
export class PublicShellComponent {
  protected readonly auth = inject(AuthService);
  protected readonly seller = inject(SellerSpaceService);
  protected readonly theme = inject(ThemeService);
  private readonly navCounts = inject(NavCountsService);
  private readonly router = inject(Router);

  protected readonly categories = CATEGORY_CHIPS;
  /** Everyone starts selling at /sell: an account with a farm gets the
   * wizard, anyone else opens their farm first (or signs in, via its
   * guard). */
  protected readonly sellRoute = '/sell';
  protected readonly center: BottomNavCenter = { labelKey: 'bottomNav.sell', icon: 'plus', route: '/sell' };

  protected readonly tabs = computed<readonly BottomNavItem[]>(() => [
    { kind: 'link', labelKey: 'bottomNav.home', icon: 'home', route: '/', exact: true },
    { kind: 'link', labelKey: 'bottomNav.buy', icon: 'search', route: '/marketplace' },
    {
      kind: 'link',
      labelKey: 'bottomNav.purchases',
      icon: 'calendar',
      route: '/account/reservations',
      badge: this.navCounts.counts().buyer
    },
    { kind: 'link', labelKey: 'bottomNav.account', icon: 'user', route: '/account', exact: true }
  ]);

  private readonly url = toSignal(
    this.router.events.pipe(
      filter((e) => e instanceof NavigationEnd),
      startWith(null),
      map(() => this.router.parseUrl(this.router.url))
    ),
    { requireSync: true }
  );
  protected readonly onMarket = computed(() => this.url().root.children['primary']?.segments[0]?.path === 'marketplace');
  protected readonly activeSpecies = computed(() => (this.onMarket() ? this.url().queryParamMap.get('species') : null));

  /** Focus pages (`data: { focus: true }`: listing detail, checkout…)
   * replace the bottom nav with their own sticky action bar on phones,
   * per the design, so the primary action owns the thumb zone. */
  protected readonly focus = toSignal(
    this.router.events.pipe(
      filter((e) => e instanceof NavigationEnd),
      startWith(null),
      map(() => {
        let r = this.router.routerState.snapshot.root;
        while (r.firstChild) {
          r = r.firstChild;
        }
        return !!r.data['focus'];
      })
    ),
    { initialValue: false }
  );

  protected chipClass(active: boolean): string {
    return (
      'inline-flex h-8 items-center gap-1.5 rounded-full px-3 font-semibold transition ' +
      (active ? 'bg-forest-soft text-forest-deep' : 'text-ink hover:bg-surface-sunken')
    );
  }
}
