import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { LocaleService } from '../../core/i18n/locale.service';
import { SidebarNavItem } from '../ui/sidebar-nav/sidebar-nav.component';
import { SidebarShellComponent } from './sidebar-shell.component';

const ITEMS: readonly (Omit<SidebarNavItem, 'label'> & { labelKey: string })[] = [
  { labelKey: 'portalNav.dashboard', icon: 'home', route: '/portal', exactMatch: true },
  { labelKey: 'portalNav.cycles', icon: 'cycle', route: '/portal/cycles' },
  { labelKey: 'portalNav.listings', icon: 'grid', route: '/portal/listings' },
  { labelKey: 'portalNav.reservations', icon: 'wallet', route: '/portal/reservations' },
  { labelKey: 'portalNav.payments', icon: 'chart', route: '/portal/payments' },
  { labelKey: 'portalNav.forum', icon: 'message', route: '/forum' }
];

/** The farmer portal: `SidebarShell` on `forest-deep`, per the design —
 * desktop-first (a farmer on mobile still uses the public shell's
 * `BottomNav`, whose "Publier" tab already routes into `/sell`; nothing
 * in the design shows a distinct mobile portal board), with a mobile
 * fallback nav built into the shared shell regardless. */
@Component({
  selector: 'bt-portal-shell',
  imports: [SidebarShellComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<bt-sidebar-shell [items]="items()" surface="forest-deep" />`
})
export class PortalShellComponent {
  private readonly locale = inject(LocaleService);

  protected readonly items = computed<SidebarNavItem[]>(() =>
    ITEMS.map(({ labelKey, ...item }) => ({ ...item, label: this.locale.t(labelKey) }))
  );
}
