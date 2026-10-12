import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { LocaleService } from '../../core/i18n/locale.service';
import { BottomNavCenter, BottomNavItem } from '../ui/bottom-nav/bottom-nav.component';
import { SidebarNavItem } from '../ui/sidebar-nav/sidebar-nav.component';
import { MoreSheetLink } from './more-sheet.component';
import { SidebarShellComponent } from './sidebar-shell.component';

const ITEMS: readonly (Omit<SidebarNavItem, 'label'> & { labelKey: string })[] = [
  { labelKey: 'portalNav.dashboard', icon: 'home', route: '/portal', exactMatch: true },
  { labelKey: 'portalNav.cycles', icon: 'cycle', route: '/portal/cycles' },
  { labelKey: 'portalNav.listings', icon: 'grid', route: '/portal/listings' },
  { labelKey: 'portalNav.reservations', icon: 'wallet', route: '/portal/reservations' },
  { labelKey: 'portalNav.payments', icon: 'chart', route: '/portal/payments' },
  { labelKey: 'portalNav.forum', icon: 'message', route: '/forum' }
];

const BOTTOM_ITEMS: readonly BottomNavItem[] = [
  { kind: 'link', labelKey: 'portalNav.dashboardShort', icon: 'home', route: '/portal', exact: true },
  { kind: 'link', labelKey: 'portalNav.cycles', icon: 'cycle', route: '/portal/cycles' },
  { kind: 'link', labelKey: 'portalNav.reservations', icon: 'calendar', route: '/portal/reservations' },
  { kind: 'more', labelKey: 'moreSheet.tab', icon: 'grid' }
];

const CENTER: BottomNavCenter = { labelKey: 'bottomNav.publish', icon: 'plus', route: '/portal/listings/new' };

const MORE_LINKS: readonly MoreSheetLink[] = [
  { labelKey: 'portalNav.listings', icon: 'grid', route: '/portal/listings' },
  { labelKey: 'portalNav.payments', icon: 'wallet', route: '/portal/payments' },
  { labelKey: 'portalNav.forum', icon: 'message', route: '/forum' },
  { labelKey: 'nav.marketplace', icon: 'search', route: '/marketplace' },
  { labelKey: 'nav.myAccount', icon: 'user', route: '/account' }
];

/** The farmer portal: `SidebarShell` on `forest-deep`, per the design.
 * On phones the same destinations move to a portal-specific bottom nav
 * (dashboard, cycles, publish, reservations, and a "Plus" sheet for the
 * rest), so a farmer never has to switch to a desktop to run the farm. */
@Component({
  selector: 'bt-portal-shell',
  imports: [SidebarShellComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <bt-sidebar-shell
      [items]="items()"
      surface="forest-deep"
      homeRoute="/portal"
      [bottomItems]="bottomItems"
      [center]="center"
      [moreLinks]="moreLinks"
    />
  `
})
export class PortalShellComponent {
  private readonly locale = inject(LocaleService);

  protected readonly bottomItems = BOTTOM_ITEMS;
  protected readonly center = CENTER;
  protected readonly moreLinks = MORE_LINKS;

  protected readonly items = computed<SidebarNavItem[]>(() =>
    ITEMS.map(({ labelKey, ...item }) => ({ ...item, label: this.locale.t(labelKey) }))
  );
}
