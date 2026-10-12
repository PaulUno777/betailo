import { NgTemplateOutlet } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { IconComponent } from '../icon/icon.component';
import { IconName } from '../icon/icon-paths';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

export interface BottomNavLink {
  readonly kind: 'link';
  readonly labelKey: string;
  readonly icon: IconName;
  readonly route: string;
  /** Exact-match the active state — needed for a section root ("/",
   * "/portal") that prefixes every other tab's route. */
  readonly exact?: boolean;
  /** Things waiting on the user here (reservations to pay or deliver);
   * shown on the icon when above zero. */
  readonly badge?: number;
}

/** Opens an overflow sheet instead of navigating — for sections with
 * more destinations than fit in five thumb-sized tabs. */
export interface BottomNavMore {
  readonly kind: 'more';
  readonly labelKey: string;
  readonly icon: IconName;
}

export type BottomNavItem = BottomNavLink | BottomNavMore;

/** The raised clay button in the middle of the bar ("Publier"). */
export interface BottomNavCenter {
  readonly labelKey: string;
  readonly icon: IconName;
  readonly route: string;
}

/** Phone and tablet navigation: up to five labelled slots, with an
 * optional raised center action inserted between the two halves of
 * `items`. Fixed to the bottom with the safe-area inset added under its
 * 80px, hidden from `lg` up, where each shell has its own header or
 * sidebar — a tablet keeps the thumb-zone tabs rather than a cramped
 * desktop sidebar. */
@Component({
  selector: 'bt-bottom-nav',
  imports: [NgTemplateOutlet, RouterLink, RouterLinkActive, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <ng-template #tab let-item>
      @if (item.kind === 'link') {
        <a
          [routerLink]="item.route"
          routerLinkActive
          #rla="routerLinkActive"
          [routerLinkActiveOptions]="{ exact: !!item.exact }"
          [attr.aria-current]="rla.isActive ? 'page' : null"
          [class]="tabClass + (rla.isActive ? ' text-forest' : ' text-ink-muted')"
        >
          <span class="relative">
            <bt-icon [name]="item.icon" [size]="22" />
            @if (item.badge) {
              <span
                class="absolute -right-2.5 -top-1.5 flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-clay px-1 text-[11px] font-bold leading-none text-on-clay ring-2 ring-surface-raised"
              >
                {{ item.badge > 9 ? '9+' : item.badge }}
              </span>
              <span class="sr-only">{{ 'bottomNav.badge' | t: { count: item.badge } }}</span>
            }
          </span>
          <span class="max-w-full truncate px-0.5 text-caption">{{ item.labelKey | t }}</span>
        </a>
      } @else {
        <button type="button" (click)="more.emit()" [class]="tabClass + ' text-ink-muted'">
          <bt-icon [name]="item.icon" [size]="22" />
          <span class="max-w-full truncate px-0.5 text-caption">{{ item.labelKey | t }}</span>
        </button>
      }
    </ng-template>

    <nav
      [attr.aria-label]="'bottomNav.label' | t"
      class="fixed inset-x-0 bottom-0 z-40 border-t border-line bg-surface-raised/95 backdrop-blur lg:hidden"
      style="padding-bottom: env(safe-area-inset-bottom)"
    >
      <div class="flex h-bottom-nav items-stretch px-1">
        @for (item of leading(); track item.labelKey) {
          <ng-container *ngTemplateOutlet="tab; context: { $implicit: item }" />
        }
        @if (center(); as c) {
          <div class="flex flex-1 items-start justify-center">
            <!-- Labelled like every other tab: an icon alone isn't enough
                 for someone new to shopping apps. -->
            <a [routerLink]="c.route" class="group flex flex-col items-center gap-1 text-clay-ink">
              <span
                class="-mt-5 flex h-14 w-14 items-center justify-center rounded-full bg-clay text-on-clay ring-4 ring-surface-raised transition group-active:scale-95"
                style="box-shadow: var(--shadow-cta)"
              >
                <bt-icon [name]="c.icon" [size]="26" />
              </span>
              <span class="text-caption font-semibold">{{ c.labelKey | t }}</span>
            </a>
          </div>
        }
        @for (item of trailing(); track item.labelKey) {
          <ng-container *ngTemplateOutlet="tab; context: { $implicit: item }" />
        }
      </div>
    </nav>
  `
})
export class BottomNavComponent {
  readonly items = input.required<readonly BottomNavItem[]>();
  readonly center = input<BottomNavCenter>();
  readonly more = output<void>();

  protected readonly tabClass =
    'flex min-w-0 flex-1 flex-col items-center justify-center gap-1 rounded-md transition active:bg-surface-sunken';

  private readonly splitAt = computed(() => (this.center() ? Math.ceil(this.items().length / 2) : this.items().length));
  protected readonly leading = computed(() => this.items().slice(0, this.splitAt()));
  protected readonly trailing = computed(() => this.items().slice(this.splitAt()));
}
