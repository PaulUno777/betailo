import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { IconComponent } from '../icon/icon.component';
import { IconName } from '../icon/icon-paths';
import { LogoComponent } from '../logo/logo.component';

export interface SidebarNavItem {
  readonly label: string;
  readonly icon: IconName;
  readonly route: string;
  readonly badgeCount?: number;
  /** Exact-match this item's `routerLinkActive` — the overview route
   * (`/portal`, `/admin`) is a URL prefix of every other item's route, so
   * without this it stays highlighted alongside whichever child page is
   * actually open. */
  readonly exactMatch?: boolean;
}

export type SidebarNavSurface = 'forest-deep' | 'ink';

/** Farmer-portal navigation on `forest-deep`; admin reuses the same
 * component on `ink` in its own layout (README) — the surface, not the
 * component, is what makes admin visually distinct. */
@Component({
  selector: 'bt-sidebar-nav',
  imports: [RouterLink, RouterLinkActive, IconComponent, LogoComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <nav [class]="'flex h-full w-sidebar flex-col gap-1 p-5 ' + surfaceClass()">
      <div class="mb-6 px-2">
        <bt-logo tone="light" />
      </div>
      @for (item of items(); track item.route) {
        <a
          [routerLink]="item.route"
          routerLinkActive
          [routerLinkActiveOptions]="{ exact: !!item.exactMatch }"
          #rla="routerLinkActive"
          [class]="
            'flex items-center gap-3 rounded-md px-3 py-2.5 text-body-strong transition ' +
            (rla.isActive ? 'bg-surface text-ink' : 'text-on-forest-deep/80 hover:bg-surface/10')
          "
        >
          <bt-icon [name]="item.icon" [size]="20" />
          <span class="flex-1">{{ item.label }}</span>
          @if (item.badgeCount) {
            <span class="rounded-full bg-clay px-2 py-0.5 text-caption text-on-clay">{{ item.badgeCount }}</span>
          }
        </a>
      }
    </nav>
  `
})
export class SidebarNavComponent {
  readonly items = input.required<SidebarNavItem[]>();
  readonly surface = input<SidebarNavSurface>('forest-deep');

  protected surfaceClass(): string {
    return this.surface() === 'forest-deep' ? 'bg-forest-deep' : 'bg-ink';
  }
}
