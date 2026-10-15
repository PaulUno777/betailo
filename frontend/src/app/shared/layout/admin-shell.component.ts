import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink } from '@angular/router';
import { filter } from 'rxjs';
import { StaffSpaceService } from '../../core/auth/staff-space.service';
import { apiResource } from '../../core/http/api-resource';
import { LocaleService } from '../../core/i18n/locale.service';
import { AdminApi } from '../../data-access/admin/admin.api';
import { Permission } from '../../domain/enums';
import { AdminNavCounts, StaffOverview } from '../../domain/models';
import { AvatarComponent } from '../ui/avatar/avatar.component';
import { BottomNavItem } from '../ui/bottom-nav/bottom-nav.component';
import { IconName } from '../ui/icon/icon-paths';
import { SidebarBadgeTone, SidebarNavItem } from '../ui/sidebar-nav/sidebar-nav.component';
import { StaffRoleBadgeComponent } from '../ui/staff-role-badge/staff-role-badge.component';
import { MoreSheetLink } from './more-sheet.component';
import { SidebarShellComponent } from './sidebar-shell.component';

type Group = 'steer' | 'trust' | 'system';
type CountKey = keyof AdminNavCounts;

interface NavDef {
  readonly labelKey: string;
  readonly icon: IconName;
  readonly route: string;
  readonly group: Group;
  /** What the page needs. A role without it sees the item locked, with
   * the roles to ask; the API refuses the page's calls either way. */
  readonly permission: Permission;
  readonly exactMatch?: boolean;
  readonly count?: CountKey;
  readonly countTone?: SidebarBadgeTone;
}

/** The back office's map, in the design's three groups: running the
 * platform, keeping it honest, and the team itself. */
const NAV: readonly NavDef[] = [
  { labelKey: 'adminNav.overview', icon: 'chart', route: '/admin', group: 'steer', permission: 'overview.read', exactMatch: true },
  { labelKey: 'adminNav.users', icon: 'users', route: '/admin/users', group: 'steer', permission: 'accounts.read' },
  { labelKey: 'adminNav.listings', icon: 'cycle', route: '/admin/listings', group: 'steer', permission: 'catalog.read' },
  { labelKey: 'adminNav.escrow', icon: 'wallet', route: '/admin/escrow', group: 'steer', permission: 'escrow.read' },
  { labelKey: 'adminNav.farms', icon: 'leaf', route: '/admin/farms', group: 'trust', permission: 'farms.read', count: 'pendingFarms', countTone: 'clay' },
  { labelKey: 'adminNav.reports', icon: 'flag', route: '/admin/reports', group: 'trust', permission: 'reports.read', count: 'openReports', countTone: 'alert' },
  { labelKey: 'adminNav.disputes', icon: 'alert', route: '/admin/disputes', group: 'trust', permission: 'reports.read', count: 'openDisputes', countTone: 'muted' },
  { labelKey: 'adminNav.forum', icon: 'message', route: '/forum', group: 'trust', permission: 'overview.read' },
  { labelKey: 'adminNav.staff', icon: 'shield', route: '/admin/staff', group: 'system', permission: 'staff.manage' },
  { labelKey: 'adminNav.settings', icon: 'settings', route: '/admin/settings', group: 'system', permission: 'staff.manage' },
  { labelKey: 'adminNav.audit', icon: 'clock', route: '/admin/audit', group: 'system', permission: 'audit.read' }
];

/** The phone's four tabs, in the order staff reach for them; a tab the
 * role can't open is left out and the next one takes its place. */
const TAB_ROUTES: readonly string[] = ['/admin', '/admin/farms', '/admin/reports', '/admin/escrow', '/admin/users'];
const TAB_LABELS: Readonly<Record<string, string>> = { '/admin': 'adminNav.overviewShort', '/admin/farms': 'adminNav.farmsShort' };

/** The back office: `SidebarShell` on `ink`, a deliberately distinct
 * look from the seller portal's green. The navigation is drawn from
 * what the signed-in role may do (`GET /me`), with the work waiting for
 * staff counted beside it (`GET /admin/nav-counts`). */
@Component({
  selector: 'bt-admin-shell',
  imports: [RouterLink, SidebarShellComponent, AvatarComponent, StaffRoleBadgeComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <bt-sidebar-shell
      [items]="items()"
      surface="ink"
      spaceLabelKey="adminNav.space"
      navLabelKey="adminNav.navLabel"
      homeRoute="/admin"
      [bottomItems]="bottomItems()"
      [moreLinks]="moreLinks()"
    >
      <a sidebarFooter routerLink="/account" class="flex items-center gap-2.5 rounded-xl bg-white/5 px-3 py-2.5 transition hover:bg-white/10">
        <bt-avatar [name]="staff.name()" [src]="staff.avatarUrl()" [size]="34" tone="forest" />
        <span class="flex min-w-0 flex-col items-start gap-1">
          <b class="max-w-full truncate text-[13.5px] text-on-forest-deep">{{ staff.name() }}</b>
          @if (staff.role(); as role) {
            <bt-staff-role-badge [role]="role" onDark />
          }
        </span>
      </a>
    </bt-sidebar-shell>
  `
})
export class AdminShellComponent {
  private readonly locale = inject(LocaleService);
  private readonly adminApi = inject(AdminApi);
  private readonly router = inject(Router);
  protected readonly staff = inject(StaffSpaceService);

  /** Re-read on every page change: a farm approved or a report closed
   * on one page moves the counters on all of them. */
  private readonly navigated = toSignal(this.router.events.pipe(filter((e) => e instanceof NavigationEnd)), { initialValue: null });
  private readonly counts = apiResource<AdminNavCounts>(() => {
    this.navigated();
    return this.staff.role() ? this.adminApi.navCountsUrl() : undefined;
  });
  /** The permission table, to name who can open a locked page. */
  private readonly table = apiResource<StaffOverview>(() => (this.staff.role() ? this.adminApi.staffUrl() : undefined));

  constructor() {
    // Entering the back office: the role may have changed since sign-in.
    this.staff.refresh();
  }

  private lockedReason(permission: Permission): string {
    const roles = (this.table.value()?.roles ?? [])
      .filter((r) => r.permissions.includes(permission))
      .map((r) => this.locale.t(`staffRole.${r.role}`));
    return roles.length ? this.locale.t('adminNav.reservedFor', { roles: roles.join(', ') }) : this.locale.t('adminNav.reserved');
  }

  protected readonly items = computed<SidebarNavItem[]>(() => {
    const counts = this.counts.value();
    return NAV.map((def) => ({
      label: this.locale.t(def.labelKey),
      icon: def.icon,
      route: def.route,
      exactMatch: def.exactMatch,
      group: this.locale.t(`adminNav.group.${def.group}`),
      badgeCount: def.count ? counts?.[def.count] : undefined,
      badgeTone: def.countTone,
      lockedReason: this.staff.loaded() && !this.staff.can(def.permission) ? this.lockedReason(def.permission) : undefined
    }));
  });

  private readonly allowed = computed(() => NAV.filter((def) => this.staff.can(def.permission)));

  protected readonly bottomItems = computed<readonly BottomNavItem[]>(() => {
    const allowed = this.allowed();
    const tabs = TAB_ROUTES.map((route) => allowed.find((def) => def.route === route))
      .filter((def): def is NavDef => !!def)
      .slice(0, 4);
    const counts = this.counts.value();
    return [
      ...tabs.map(
        (def): BottomNavItem => ({
          kind: 'link',
          labelKey: TAB_LABELS[def.route] ?? def.labelKey,
          icon: def.icon,
          route: def.route,
          exact: def.exactMatch,
          badge: def.count ? counts?.[def.count] : undefined
        })
      ),
      { kind: 'more', labelKey: 'moreSheet.tab', icon: 'more' }
    ];
  });

  protected readonly moreLinks = computed<readonly MoreSheetLink[]>(() => {
    const inTabs = new Set(this.bottomItems().flatMap((item) => (item.kind === 'link' ? [item.route] : [])));
    return [
      ...this.allowed()
        .filter((def) => !inTabs.has(def.route))
        .map((def) => ({ labelKey: def.labelKey, icon: def.icon, route: def.route })),
      { labelKey: 'adminNav.viewSite', icon: 'search' as const, route: '/' },
      { labelKey: 'nav.myAccount', icon: 'user' as const, route: '/account' }
    ];
  });
}
