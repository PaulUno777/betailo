import { NgTemplateOutlet } from '@angular/common';
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
  imports: [NgTemplateOutlet, RouterLink, RouterLinkActive, IconComponent, LogoComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <ng-template #link let-item>
      <a
        [routerLink]="item.route"
        routerLinkActive
        [routerLinkActiveOptions]="{ exact: !!item.exactMatch }"
        #rla="routerLinkActive"
        [attr.aria-current]="rla.isActive ? 'page' : null"
        [class]="
          'flex min-h-11 items-center gap-3 rounded-md px-3 py-2.5 text-body-strong transition ' +
          (rla.isActive ? 'bg-surface text-ink' : 'text-on-forest-deep/80 hover:bg-surface/10')
        "
      >
        <bt-icon [name]="item.icon" [size]="20" />
        <span class="flex-1">{{ item.label }}</span>
        @if (item.badgeCount) {
          <span class="rounded-full bg-clay px-2 py-0.5 text-caption text-on-clay">{{ item.badgeCount > 9 ? '9+' : item.badgeCount }}</span>
        }
      </a>
    </ng-template>

    <nav [class]="'flex h-full w-sidebar flex-col gap-1 overflow-y-auto p-5 ' + surfaceClass()">
      <div class="mb-6 flex items-center gap-2 px-2">
        <bt-logo tone="light" />
        @if (badgeLabel(); as label) {
          <span class="rounded-full bg-on-forest-deep/15 px-2 py-0.5 text-caption font-semibold text-on-forest-deep">{{ label }}</span>
        }
      </div>
      @for (item of items(); track item.route) {
        <ng-container *ngTemplateOutlet="link; context: { $implicit: item }" />
      }
      <div class="mt-auto flex flex-col gap-1 border-t border-on-forest-deep/15 pt-4">
        <ng-content />
        @for (item of footer(); track item.route) {
          <ng-container *ngTemplateOutlet="link; context: { $implicit: item }" />
        }
      </div>
    </nav>
  `
})
export class SidebarNavComponent {
  readonly items = input.required<SidebarNavItem[]>();
  readonly surface = input<SidebarNavSurface>('forest-deep');
  /** Pinned to the bottom: the way out of the section (back to the
   * market, the account), under whatever the caller projects there. */
  readonly footer = input<SidebarNavItem[]>([]);
  /** A chip next to the logo naming the space ("Vendeur", "Admin"). */
  readonly badgeLabel = input<string>();

  protected surfaceClass(): string {
    return this.surface() === 'forest-deep' ? 'bg-forest-deep' : 'bg-ink';
  }
}
