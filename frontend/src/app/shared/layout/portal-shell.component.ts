import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { NoticeComponent } from '../ui/notice/notice.component';
import { NavCountsService } from '../../core/navigation/nav-counts.service';
import { BottomNavCenter, BottomNavItem } from '../ui/bottom-nav/bottom-nav.component';
import { SidebarNavItem } from '../ui/sidebar-nav/sidebar-nav.component';
import { MoreSheetLink } from './more-sheet.component';
import { SidebarShellComponent } from './sidebar-shell.component';

type ItemDef = Omit<SidebarNavItem, 'label'> & { readonly labelKey: string };

/** In the order a farmer reaches for them: what needs doing today, what
 * is for sale, the sales themselves, then the batches behind them. */
const ITEMS: readonly ItemDef[] = [
  { labelKey: 'portalNav.dashboard', icon: 'home', route: '/portal', exactMatch: true },
  { labelKey: 'portalNav.listings', icon: 'grid', route: '/portal/listings' },
  { labelKey: 'portalNav.sales', icon: 'calendar', route: '/portal/reservations' },
  { labelKey: 'portalNav.cycles', icon: 'cycle', route: '/portal/cycles' },
  { labelKey: 'portalNav.payments', icon: 'wallet', route: '/portal/payments' },
  { labelKey: 'portalNav.farm', icon: 'settings', route: '/portal/farm' }
];

const FOOTER: readonly ItemDef[] = [
  { labelKey: 'nav.forum', icon: 'message', route: '/forum' },
  { labelKey: 'nav.myAccount', icon: 'user', route: '/account' }
];

const CENTER: BottomNavCenter = { labelKey: 'bottomNav.sell', icon: 'plus', route: '/sell' };

const MORE_LINKS: readonly MoreSheetLink[] = [
  { labelKey: 'portalNav.cycles', icon: 'cycle', route: '/portal/cycles' },
  { labelKey: 'portalNav.payments', icon: 'wallet', route: '/portal/payments' },
  { labelKey: 'portalNav.farm', icon: 'settings', route: '/portal/farm' },
  { labelKey: 'nav.forum', icon: 'message', route: '/forum' },
  { labelKey: 'portalNav.buyMode', icon: 'search', route: '/' },
  { labelKey: 'nav.myAccount', icon: 'user', route: '/account' }
];

/** The seller space: `SidebarShell` on `forest-deep`, per the design.
 * On phones and tablets: Accueil, Annonces, the raised Vendre, Ventes
 * (badged with orders waiting to be delivered) and "Plus" for batches,
 * payments, the farm, and the way back to buying. */
@Component({
  selector: 'bt-portal-shell',
  imports: [SidebarShellComponent, NoticeComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <bt-sidebar-shell
      [items]="items()"
      [footer]="footer()"
      surface="forest-deep"
      spaceLabelKey="portalNav.space"
      [modeSwitch]="true"
      homeRoute="/portal"
      [bottomItems]="bottomItems()"
      [center]="center"
      [moreLinks]="moreLinks"
    >
      @if (seller.pendingVerification()) {
        <div shellBanner class="px-4 pt-4 sm:px-8">
          <bt-notice tone="warning" icon="clock" [title]="'sellerSpace.pendingTitle' | t">{{ 'sellerSpace.pendingBody' | t }}</bt-notice>
        </div>
      }
    </bt-sidebar-shell>
  `
})
export class PortalShellComponent {
  private readonly locale = inject(LocaleService);
  private readonly navCounts = inject(NavCountsService);
  protected readonly seller = inject(SellerSpaceService);

  constructor() {
    // Entering the seller space: an admin may have verified them since.
    this.seller.refresh();
  }

  protected readonly center = CENTER;
  protected readonly moreLinks = MORE_LINKS;

  protected readonly items = computed<SidebarNavItem[]>(() =>
    ITEMS.map(({ labelKey, ...item }) => ({
      ...item,
      label: this.locale.t(labelKey),
      badgeCount: item.route === '/portal/reservations' ? this.navCounts.counts().seller : undefined
    }))
  );

  protected readonly footer = computed<SidebarNavItem[]>(() =>
    FOOTER.map(({ labelKey, ...item }) => ({ ...item, label: this.locale.t(labelKey) }))
  );

  protected readonly bottomItems = computed<readonly BottomNavItem[]>(() => [
    { kind: 'link', labelKey: 'portalNav.dashboardShort', icon: 'home', route: '/portal', exact: true },
    { kind: 'link', labelKey: 'portalNav.listingsShort', icon: 'grid', route: '/portal/listings' },
    { kind: 'link', labelKey: 'portalNav.sales', icon: 'calendar', route: '/portal/reservations', badge: this.navCounts.counts().seller },
    { kind: 'more', labelKey: 'moreSheet.tab', icon: 'more' }
  ]);
}
