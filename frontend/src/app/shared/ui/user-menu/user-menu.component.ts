import { ChangeDetectionStrategy, Component, ElementRef, HostListener, computed, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { Role } from '../../../domain/enums';
import { AvatarComponent } from '../avatar/avatar.component';
import { IconComponent } from '../icon/icon.component';
import { IconName } from '../icon/icon-paths';
import { ThemeService } from '../../../core/theme/theme.service';
import { LocaleToggleComponent } from '../locale-toggle/locale-toggle.component';

interface RoleLink {
  readonly route: string;
  readonly labelKey: string;
  readonly icon: IconName;
}

const MY_RESERVATIONS_LINK: RoleLink = { route: '/account/reservations', labelKey: 'nav.myReservations', icon: 'wallet' };
const FAVORITES_LINK: RoleLink = { route: '/account/favorites', labelKey: 'nav.favorites', icon: 'heart' };

// Extra destination on top of "Mes réservations" — the backend has no
// role check on creating a reservation, so a farmer or admin can buy too;
// this only adds their own dashboard, it never hides the buyer one.
const EXTRA_ROLE_LINK: Readonly<Partial<Record<Role, RoleLink>>> = {
  farmer: { route: '/portal', labelKey: 'nav.sellerSpace', icon: 'home' },
  admin: { route: '/admin', labelKey: 'nav.admin', icon: 'chart' }
};

/** The one entry point for everything account-related — avatar (+ name on
 * wider screens), a dropdown for the role destination, account settings
 * and sign-out. Consolidates what used to be several separate header
 * controls (a nav link, a bare sign-out icon) into a single trigger, so
 * the header's right side stays short regardless of role. */
@Component({
  selector: 'bt-user-menu',
  imports: [RouterLink, AvatarComponent, IconComponent, LocaleToggleComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="relative">
      <button
        type="button"
        (click)="open.set(!open())"
        [attr.aria-expanded]="open()"
        aria-haspopup="menu"
        class="flex items-center gap-2 rounded-full py-1 pl-1 pr-2 transition hover:bg-surface-sunken"
      >
        <bt-avatar [name]="displayName()" [size]="32" />
        <span class="hidden max-w-32 truncate text-small text-ink sm:inline">{{ displayName() }}</span>
        <bt-icon name="chevron-down" [size]="16" class="text-ink-muted" />
      </button>

      @if (open()) {
        <div
          role="menu"
          class="absolute right-0 top-full z-40 mt-2 w-56 rounded-lg border border-line bg-surface-card py-2"
          style="box-shadow: var(--shadow-dialog)"
        >
          <p class="truncate px-4 pb-2 text-small text-ink-muted">{{ auth.user()?.email }}</p>
          <div class="border-t border-line"></div>
          @for (link of roleLinks(); track link.route) {
            <a
              [routerLink]="link.route"
              role="menuitem"
              (click)="open.set(false)"
              class="flex items-center gap-3 px-4 py-2.5 text-body text-ink hover:bg-surface-sunken"
            >
              <bt-icon [name]="link.icon" [size]="18" />
              {{ link.labelKey | t }}
            </a>
          }
          <a
            routerLink="/account"
            role="menuitem"
            (click)="open.set(false)"
            class="flex items-center gap-3 px-4 py-2.5 text-body text-ink hover:bg-surface-sunken"
          >
            <bt-icon name="user" [size]="18" />
            {{ 'nav.myAccount' | t }}
          </a>
          <!-- Phones only: the compact mobile header drops these two. -->
          <div class="border-t border-line sm:hidden"></div>
          <div class="flex items-center justify-between gap-3 px-4 py-2 sm:hidden">
            <span class="text-body text-ink">{{ 'moreSheet.language' | t }}</span>
            <bt-locale-toggle />
          </div>
          <button
            type="button"
            role="menuitem"
            (click)="theme.toggle()"
            class="flex w-full items-center gap-3 px-4 py-2.5 text-left text-body text-ink hover:bg-surface-sunken sm:hidden"
          >
            <bt-icon [name]="theme.isDark() ? 'sun' : 'moon'" [size]="18" />
            {{ (theme.isDark() ? 'moreSheet.lightMode' : 'moreSheet.darkMode') | t }}
          </button>
          <div class="border-t border-line"></div>
          <button
            type="button"
            role="menuitem"
            (click)="signOut()"
            class="flex w-full items-center gap-3 px-4 py-2.5 text-left text-body text-alert hover:bg-alert-soft"
          >
            <bt-icon name="logout" [size]="18" />
            {{ 'nav.signOut' | t }}
          </button>
        </div>
      }
    </div>
  `
})
export class UserMenuComponent {
  protected readonly auth = inject(AuthService);
  protected readonly theme = inject(ThemeService);
  private readonly router = inject(Router);
  private readonly elementRef = inject(ElementRef<HTMLElement>);

  protected readonly open = signal(false);

  protected readonly displayName = computed(() => this.auth.user()?.email ?? '');

  protected readonly roleLinks = computed<readonly RoleLink[]>(() => {
    const role = this.auth.role();
    if (!role) {
      return [];
    }
    const extra = EXTRA_ROLE_LINK[role];
    return extra ? [extra, MY_RESERVATIONS_LINK, FAVORITES_LINK] : [MY_RESERVATIONS_LINK, FAVORITES_LINK];
  });

  protected signOut(): void {
    this.open.set(false);
    this.auth.signOut();
    this.router.navigateByUrl('/');
  }

  @HostListener('document:click', ['$event'])
  protected onDocumentClick(event: MouseEvent): void {
    if (this.open() && !this.elementRef.nativeElement.contains(event.target as Node)) {
      this.open.set(false);
    }
  }

  @HostListener('document:keydown.escape')
  protected onEscape(): void {
    this.open.set(false);
  }
}
