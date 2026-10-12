import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink } from '@angular/router';
import { filter, map } from 'rxjs';
import { FarmStatus } from '../../domain/enums';
import { IconName } from '../ui/icon/icon-paths';
import { FarmSwitcherComponent } from '../ui/farm-switcher/farm-switcher.component';
import { AvatarComponent } from '../ui/avatar/avatar.component';
import { NoticeComponent, NoticeTone } from '../ui/notice/notice.component';
import { NavCountsService } from '../../core/navigation/nav-counts.service';
import { BottomNavCenter, BottomNavItem } from '../ui/bottom-nav/bottom-nav.component';
import { SidebarNavItem } from '../ui/sidebar-nav/sidebar-nav.component';
import { MoreSheetLink } from './more-sheet.component';
import { SidebarShellComponent } from './sidebar-shell.component';

/** The seller's map, per the portal design: today, the farms, the
 * batches behind what is for sale, the listings, the orders, the money,
 * and the forum where breeders help each other. */
const ITEMS: readonly { labelKey: string; icon: IconName; route: string; exactMatch?: boolean }[] = [
  { labelKey: 'portalNav.dashboard', icon: 'home', route: '/portal', exactMatch: true },
  { labelKey: 'portalNav.farms', icon: 'leaf', route: '/portal/farms' },
  { labelKey: 'portalNav.cycles', icon: 'cycle', route: '/portal/cycles' },
  { labelKey: 'portalNav.listings', icon: 'grid', route: '/portal/listings' },
  { labelKey: 'portalNav.sales', icon: 'calendar', route: '/portal/reservations' },
  { labelKey: 'portalNav.payments', icon: 'wallet', route: '/portal/payments' },
  { labelKey: 'nav.forum', icon: 'message', route: '/forum' }
];

/** What the seller is told about the farm they're working on while it
 * can't sell, and where to act on it. An approved farm shows nothing. */
const BANNERS: Readonly<Partial<Record<FarmStatus, { tone: NoticeTone; icon: IconName; key: string }>>> = {
  draft: { tone: 'warning', icon: 'alert', key: 'draft' },
  pending: { tone: 'info', icon: 'clock', key: 'pending' },
  rejected: { tone: 'danger', icon: 'alert', key: 'rejected' },
  suspended: { tone: 'danger', icon: 'alert', key: 'suspended' }
};

const CENTER: BottomNavCenter = { labelKey: 'bottomNav.sell', icon: 'plus', route: '/sell' };

const MORE_LINKS: readonly MoreSheetLink[] = [
  { labelKey: 'portalNav.cycles', icon: 'cycle', route: '/portal/cycles' },
  { labelKey: 'portalNav.payments', icon: 'wallet', route: '/portal/payments' },
  { labelKey: 'portalNav.farm', icon: 'settings', route: '/portal/farm' },
  { labelKey: 'portalNav.farms', icon: 'leaf', route: '/portal/farms' },
  { labelKey: 'nav.forum', icon: 'message', route: '/forum' },
  { labelKey: 'portalNav.buyMode', icon: 'search', route: '/' },
  { labelKey: 'nav.myAccount', icon: 'user', route: '/account' }
];

/** The seller space: `SidebarShell` on `forest-deep`, per the design:
 * the farm switcher under the logo, the seller's pages, and at the foot
 * the account itself, which is also the way back to buying.
 * On phones and tablets: Accueil, Annonces, the raised Vendre, Ventes
 * (badged with orders waiting to be delivered) and "Plus" for batches,
 * payments, the farms, and the way back to buying. */
@Component({
  selector: 'bt-portal-shell',
  imports: [RouterLink, SidebarShellComponent, NoticeComponent, FarmSwitcherComponent, AvatarComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <bt-sidebar-shell
      [items]="items()"
      surface="forest-deep"
      spaceLabelKey="portalNav.space"
      navLabelKey="portalNav.navLabel"
      homeRoute="/portal"
      [bottomItems]="bottomItems()"
      [center]="center"
      [moreLinks]="moreLinks"
    >
      <bt-farm-switcher sidebarTop class="block" />
      <a sidebarFooter routerLink="/account" class="flex items-center gap-2.5 rounded-[14px] bg-on-forest-deep/6 p-3 transition hover:bg-on-forest-deep/12">
        <bt-avatar [name]="accountName()" [size]="38" tone="clay" class="shrink-0" />
        <span class="flex min-w-0 flex-col leading-snug">
          <b class="truncate text-small text-on-forest-deep">{{ accountName() }}</b>
          <span class="truncate text-caption text-on-forest-deep/70">{{ 'portalNav.accountHint' | t }}</span>
        </span>
      </a>
      @if (banner(); as b) {
        <div shellBanner class="px-4 pt-4 sm:px-8 lg:px-10 lg:pt-8">
          <bt-notice [tone]="b.tone" [icon]="b.icon" [title]="b.titleKey | t: { farm: b.farm }">
            {{ b.reason || (b.bodyKey | t) }}
            <a routerLink="/portal/farms" class="ml-1 font-semibold underline underline-offset-2">{{ b.actionKey | t }}</a>
          </bt-notice>
        </div>
      }
    </bt-sidebar-shell>
  `
})
export class PortalShellComponent {
  private readonly locale = inject(LocaleService);
  private readonly navCounts = inject(NavCountsService);
  protected readonly seller = inject(SellerSpaceService);

  protected readonly accountName = computed(() => this.seller.account()?.displayName || this.seller.account()?.email || '');

  constructor() {
    // Entering the seller space: staff may have decided on a farm since.
    this.seller.refresh();
  }

  private readonly router = inject(Router);
  private readonly onFarmsPage = toSignal(
    this.router.events.pipe(
      filter((e) => e instanceof NavigationEnd),
      map(() => this.router.url.startsWith('/portal/farms'))
    ),
    { initialValue: this.router.url.startsWith('/portal/farms') }
  );

  protected readonly banner = computed(() => {
    const farm = this.seller.currentFarm();
    const def = farm ? BANNERS[farm.status] : undefined;
    // "Mes fermes" says the same per farm, with the action beside it.
    if (!farm || !def || this.onFarmsPage()) return null;
    return {
      tone: def.tone,
      icon: def.icon,
      farm: farm.name,
      titleKey: `farmStatusBanner.${def.key}Title`,
      bodyKey: `farmStatusBanner.${def.key}Body`,
      actionKey: `farmStatusBanner.${def.key}Action`,
      // Staff's own words, when they gave a reason.
      reason: farm.status === 'rejected' || farm.status === 'suspended' ? (farm.review?.rejectionReason ?? '') : ''
    };
  });

  protected readonly center = CENTER;
  protected readonly moreLinks = MORE_LINKS;

  protected readonly items = computed<SidebarNavItem[]>(() =>
    ITEMS.map(({ labelKey, ...item }): SidebarNavItem => {
      const base = { ...item, label: this.locale.t(labelKey) };
      switch (item.route) {
        case '/portal/farms':
          return { ...base, badgeCount: this.seller.farms().length, badgeTone: 'plain' };
        case '/portal/reservations':
          return { ...base, badgeCount: this.navCounts.counts().seller, badgeTone: 'clay' };
        default:
          return base;
      }
    })
  );

  protected readonly bottomItems = computed<readonly BottomNavItem[]>(() => [
    { kind: 'link', labelKey: 'portalNav.dashboardShort', icon: 'home', route: '/portal', exact: true },
    { kind: 'link', labelKey: 'portalNav.listingsShort', icon: 'grid', route: '/portal/listings' },
    { kind: 'link', labelKey: 'portalNav.sales', icon: 'calendar', route: '/portal/reservations', badge: this.navCounts.counts().seller },
    { kind: 'more', labelKey: 'moreSheet.tab', icon: 'more' }
  ]);
}
