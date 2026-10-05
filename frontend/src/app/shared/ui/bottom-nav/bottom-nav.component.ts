import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { IconComponent } from '../icon/icon.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

/** Mobile navigation, five slots, the raised clay "Publier" button centred
 * — fixed to the bottom with the safe-area inset added below its 80px. */
@Component({
  selector: 'bt-bottom-nav',
  imports: [RouterLink, RouterLinkActive, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <nav
      class="fixed inset-x-0 bottom-0 z-40 flex h-bottom-nav items-center justify-around bg-surface-raised border-t border-line sm:hidden"
      style="padding-bottom: env(safe-area-inset-bottom)"
    >
      <a
        routerLink="/"
        routerLinkActive
        #home="routerLinkActive"
        [routerLinkActiveOptions]="{ exact: true }"
        [class]="'flex flex-col items-center gap-1 ' + (home.isActive ? 'text-forest' : 'text-ink-muted')"
      >
        <bt-icon name="home" [size]="22" />
        <span class="text-caption">{{ 'bottomNav.home' | t }}</span>
      </a>
      <a
        routerLink="/marketplace"
        routerLinkActive
        #market="routerLinkActive"
        [class]="'flex flex-col items-center gap-1 ' + (market.isActive ? 'text-forest' : 'text-ink-muted')"
      >
        <bt-icon name="search" [size]="22" />
        <span class="text-caption">{{ 'bottomNav.market' | t }}</span>
      </a>
      <a routerLink="/sell" [attr.aria-label]="'bottomNav.publish' | t" class="-mt-8 flex h-14 w-14 items-center justify-center rounded-full bg-clay text-on-clay" style="box-shadow: var(--shadow-cta)">
        <bt-icon name="plus" [size]="26" />
      </a>
      <a
        routerLink="/account/reservations"
        routerLinkActive
        #reservations="routerLinkActive"
        [class]="'flex flex-col items-center gap-1 ' + (reservations.isActive ? 'text-forest' : 'text-ink-muted')"
      >
        <bt-icon name="wallet" [size]="22" />
        <span class="text-caption">{{ 'bottomNav.reservations' | t }}</span>
      </a>
      <a
        routerLink="/account"
        routerLinkActive
        #profile="routerLinkActive"
        [class]="'flex flex-col items-center gap-1 ' + (profile.isActive ? 'text-forest' : 'text-ink-muted')"
      >
        <bt-icon name="user" [size]="22" />
        <span class="text-caption">{{ 'bottomNav.profile' | t }}</span>
      </a>
    </nav>
  `
})
export class BottomNavComponent {}
