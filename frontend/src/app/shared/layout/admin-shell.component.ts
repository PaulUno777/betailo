import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { LocaleService } from '../../core/i18n/locale.service';
import { BottomNavItem } from '../ui/bottom-nav/bottom-nav.component';
import { SidebarNavItem } from '../ui/sidebar-nav/sidebar-nav.component';
import { MoreSheetLink } from './more-sheet.component';
import { SidebarShellComponent } from './sidebar-shell.component';

const ITEMS: readonly (Omit<SidebarNavItem, 'label'> & { labelKey: string })[] = [
  { labelKey: 'adminNav.overview', icon: 'chart', route: '/admin', exactMatch: true },
  { labelKey: 'adminNav.users', icon: 'users', route: '/admin/users' },
  { labelKey: 'adminNav.listings', icon: 'grid', route: '/admin/listings' },
  { labelKey: 'adminNav.escrow', icon: 'wallet', route: '/admin/escrow' },
  { labelKey: 'adminNav.reports', icon: 'flag', route: '/admin/reports' },
  { labelKey: 'adminNav.forum', icon: 'message', route: '/forum' },
  { labelKey: 'adminNav.settings', icon: 'settings', route: '/admin/settings' }
];

// No raised center action: admins moderate, they don't publish.
const BOTTOM_ITEMS: readonly BottomNavItem[] = [
  { kind: 'link', labelKey: 'adminNav.overviewShort', icon: 'chart', route: '/admin', exact: true },
  { kind: 'link', labelKey: 'adminNav.reports', icon: 'flag', route: '/admin/reports' },
  { kind: 'link', labelKey: 'adminNav.escrow', icon: 'wallet', route: '/admin/escrow' },
  { kind: 'link', labelKey: 'adminNav.users', icon: 'users', route: '/admin/users' },
  { kind: 'more', labelKey: 'moreSheet.tab', icon: 'grid' }
];

const MORE_LINKS: readonly MoreSheetLink[] = [
  { labelKey: 'adminNav.listings', icon: 'grid', route: '/admin/listings' },
  { labelKey: 'adminNav.forum', icon: 'message', route: '/forum' },
  { labelKey: 'adminNav.settings', icon: 'settings', route: '/admin/settings' },
  { labelKey: 'nav.marketplace', icon: 'search', route: '/marketplace' }
];

/** Admin: `SidebarShell` on `ink` — a deliberately distinct back-office
 * look, never the portal's green, per the design (admin is a separate
 * visual theme, not a reuse of the seller shell — only the surface
 * differs, the shell component is the same one). */
@Component({
  selector: 'bt-admin-shell',
  imports: [SidebarShellComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <bt-sidebar-shell
      [items]="items()"
      surface="ink"
      homeRoute="/admin"
      [bottomItems]="bottomItems"
      [moreLinks]="moreLinks"
    />
  `
})
export class AdminShellComponent {
  private readonly locale = inject(LocaleService);

  protected readonly bottomItems = BOTTOM_ITEMS;
  protected readonly moreLinks = MORE_LINKS;

  protected readonly items = computed<SidebarNavItem[]>(() =>
    ITEMS.map(({ labelKey, ...item }) => ({ ...item, label: this.locale.t(labelKey) }))
  );
}
