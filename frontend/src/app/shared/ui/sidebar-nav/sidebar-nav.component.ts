import { NgTemplateOutlet } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { IconComponent } from '../icon/icon.component';
import { IconName } from '../icon/icon-paths';
import { LogoComponent } from '../logo/logo.component';

export type SidebarBadgeTone = 'clay' | 'alert' | 'muted' | 'plain';

export interface SidebarNavItem {
  readonly label: string;
  readonly icon: IconName;
  readonly route: string;
  readonly badgeCount?: number;
  /** `plain` is a bare count in mono (how many farms, how many cycles);
   * the others are chips for work that is waiting. */
  readonly badgeTone?: SidebarBadgeTone;
  /** Exact-match this item's `routerLinkActive` — the overview route
   * (`/portal`, `/admin`) is a URL prefix of every other item's route, so
   * without this it stays highlighted alongside whichever child page is
   * actually open. */
  readonly exactMatch?: boolean;
  /** Section heading shown above the first item of each group. */
  readonly group?: string;
  /** Set when the signed-in role can't open the page: the item stays
   * visible, disabled, and says who can. */
  readonly lockedReason?: string;
}

export type SidebarNavSurface = 'forest-deep' | 'ink';

const BADGE_CLASSES: Readonly<Record<Exclude<SidebarBadgeTone, 'plain'>, string>> = {
  clay: 'bg-clay text-on-clay',
  alert: 'bg-alert text-white',
  muted: 'bg-white/12 text-on-forest-deep'
};

/** The desktop sidebar of the two workspaces, per the design: the
 * seller portal on `forest-deep` (260px, the active page on a cream
 * pill) and the back office on `ink` (240px, grouped, the active page
 * tinted clay). Same component; the surface carries the difference.
 *
 * Slots: `[navTop]` sits under the logo (the farm switcher), the default
 * slot at the foot (who is signed in, and the way out). */
@Component({
  selector: 'bt-sidebar-nav',
  imports: [NgTemplateOutlet, RouterLink, RouterLinkActive, IconComponent, LogoComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <ng-template #badge let-item>
      @if (item.badgeCount) {
        @if (item.badgeTone === 'plain') {
          <span class="ml-auto font-mono text-caption font-medium">{{ item.badgeCount }}</span>
        } @else {
          <span [class]="'ml-auto inline-flex h-5 min-w-5 items-center justify-center rounded-full px-1.5 text-[11px] font-semibold ' + badgeClass(item)">
            {{ item.badgeCount > 99 ? '99+' : item.badgeCount }}
          </span>
        }
      }
    </ng-template>

    <nav [attr.aria-label]="label()" [class]="'flex h-full flex-col overflow-y-auto ' + frameClass()">
      <div class="flex items-center gap-2.5 px-2">
        <bt-logo tone="light" [markSize]="ink() ? 28 : 30" />
        @if (badgeLabel(); as text) {
          @if (ink()) {
            <span class="ml-auto font-mono text-[10.5px] font-medium uppercase tracking-[0.1em] text-clay">{{ text }}</span>
          } @else {
            <span class="ml-auto inline-flex h-[22px] items-center rounded-full bg-clay/20 px-2.5 text-[11px] font-semibold text-clay-soft">{{ text }}</span>
          }
        }
      </div>

      <ng-content select="[navTop]" />

      <div class="flex flex-col gap-0.5">
        @for (item of items(); track item.route; let i = $index) {
          @if (item.group && item.group !== items()[i - 1]?.group) {
            <span class="px-3 pb-1.5 font-mono text-[11px] font-medium uppercase tracking-[0.12em] text-on-forest-deep/50" [class.pt-4]="i > 0">
              {{ item.group }}
            </span>
          }
          @if (item.lockedReason; as reason) {
            <span
              aria-disabled="true"
              [attr.title]="reason"
              [class]="'flex cursor-not-allowed items-center gap-3 rounded-[10px] px-3 text-small text-on-forest-deep/40 ' + rowHeight()"
            >
              <bt-icon [name]="item.icon" [size]="17" />
              <span class="min-w-0 flex-1 truncate">{{ item.label }}</span>
              <bt-icon name="lock" [size]="14" />
              <span class="sr-only">{{ reason }}</span>
            </span>
          } @else {
            <a
              [routerLink]="item.route"
              routerLinkActive
              [routerLinkActiveOptions]="{ exact: !!item.exactMatch }"
              #rla="routerLinkActive"
              [attr.aria-current]="rla.isActive ? 'page' : null"
              [class]="'flex items-center gap-3 rounded-[10px] px-3 text-small transition ' + rowHeight() + ' ' + (rla.isActive ? activeClass() : idleClass())"
            >
              <bt-icon [name]="item.icon" [size]="ink() ? 17 : 18" />
              <span class="min-w-0 flex-1 truncate">{{ item.label }}</span>
              <ng-container *ngTemplateOutlet="badge; context: { $implicit: item }" />
            </a>
          }
        }
      </div>

      <div class="mt-auto flex flex-col gap-2">
        <ng-content />
      </div>
    </nav>
  `
})
export class SidebarNavComponent {
  readonly items = input.required<SidebarNavItem[]>();
  readonly surface = input<SidebarNavSurface>('forest-deep');
  /** Names the space next to the logo ("Vendeur", "Staff"). */
  readonly badgeLabel = input<string>();
  /** The navigation's accessible name. */
  readonly label = input<string>();

  protected readonly ink = computed(() => this.surface() === 'ink');

  protected frameClass(): string {
    return this.ink() ? 'w-sidebar-admin gap-5 bg-ink px-3.5 py-6' : 'w-sidebar gap-[22px] bg-forest-deep px-[18px] py-7';
  }

  protected rowHeight(): string {
    return this.ink() ? 'min-h-[42px]' : 'min-h-11';
  }

  protected activeClass(): string {
    return this.ink() ? 'bg-clay/16 font-bold text-clay-soft' : 'bg-surface font-bold text-forest-deep';
  }

  protected idleClass(): string {
    return this.ink()
      ? 'font-medium text-on-forest-deep/70 hover:bg-white/6 hover:text-white'
      : 'font-medium text-on-forest-deep/80 hover:bg-on-forest-deep/8 hover:text-on-forest-deep';
  }

  protected badgeClass(item: SidebarNavItem): string {
    const tone = item.badgeTone ?? 'clay';
    return tone === 'plain' ? '' : BADGE_CLASSES[tone];
  }
}
