import { ChangeDetectionStrategy, Component, ElementRef, HostListener, computed, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink } from '@angular/router';
import { filter, map } from 'rxjs';
import { AuthService } from '../../../core/auth/auth.service';
import { SellerSpaceService } from '../../../core/auth/seller-space.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { farmStatusEntry } from '../../../domain/rules/status-registry';
import { AvatarComponent } from '../avatar/avatar.component';
import { IconComponent } from '../icon/icon.component';
import { IconName } from '../icon/icon-paths';

interface RoleLink {
  readonly route: string;
  readonly labelKey: string;
  readonly icon: IconName;
}

const MY_RESERVATIONS_LINK: RoleLink = { route: '/account/reservations', labelKey: 'nav.myReservations', icon: 'wallet' };
const FAVORITES_LINK: RoleLink = { route: '/account/favorites', labelKey: 'nav.favorites', icon: 'heart' };

/** One of the places an account acts from: its personal side (buying),
 * its farm (selling), or the admin console. */
interface Space {
  readonly id: 'personal' | 'farm' | 'admin';
  /** Unique in the list: farms share `id`. */
  readonly key: string;
  readonly farmId?: string;
  readonly route: string;
  readonly name: string;
  readonly hintKey: string;
}

type SpaceId = Space['id'];

function spaceOf(url: string): SpaceId {
  if (url.startsWith('/portal') || url.startsWith('/sell')) return 'farm';
  return url.startsWith('/admin') ? 'admin' : 'personal';
}

/** Desktop's one entry point for everything account-related: avatar and
 * name, then a dropdown listing the account's spaces (personal, and its
 * farm once it has one, the way a social account lists the pages it
 * runs), an invitation to add a farm for those who don't sell yet,
 * account settings and sign-out. Phones and tablets use the Compte tab instead. Consolidates what used to be several separate header
 * controls (a nav link, a bare sign-out icon) into a single trigger, so
 * the header's right side stays short regardless of role. */
@Component({
  selector: 'bt-user-menu',
  imports: [RouterLink, AvatarComponent, IconComponent, TranslatePipe],
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
        <bt-avatar [name]="displayName()" [src]="auth.user()?.avatarUrl" [size]="32" />
        <span class="max-w-32 truncate text-small text-ink">{{ displayName() }}</span>
        <bt-icon name="chevron-down" [size]="16" class="text-ink-muted" />
      </button>

      @if (open()) {
        <div
          role="menu"
          class="absolute right-0 top-full z-40 mt-2 w-72 rounded-lg border border-line bg-surface-card py-2"
          style="box-shadow: var(--shadow-dialog)"
        >
          <p class="truncate px-4 pb-2 text-small text-ink-muted">{{ auth.user()?.email }}</p>
          <div class="border-t border-line"></div>
          <p class="px-4 pb-1 pt-3 text-eyebrow text-ink-muted">{{ 'nav.spaces' | t }}</p>
          @for (space of spaces(); track space.key) {
            <a
              [routerLink]="space.route"
              role="menuitem"
              [attr.aria-current]="isCurrent(space) ? 'true' : null"
              (click)="enter(space)"
              class="flex items-center gap-3 px-4 py-2 text-ink hover:bg-surface-sunken"
            >
              <bt-avatar [name]="space.name" [src]="space.id === 'personal' ? auth.user()?.avatarUrl : undefined" [size]="36" [shape]="space.id === 'personal' ? 'round' : 'square'" [tone]="space.id === 'personal' ? 'clay' : 'forest'" />
              <span class="flex min-w-0 flex-1 flex-col">
                <span class="truncate text-body-strong">{{ space.name }}</span>
                <span class="truncate text-small text-ink-muted">{{ space.hintKey | t }}</span>
              </span>
              @if (isCurrent(space)) {
                <bt-icon name="check" [size]="18" class="shrink-0 text-forest" />
              }
            </a>
          }
          @if (canAddFarm()) {
            <a routerLink="/sell" role="menuitem" (click)="open.set(false)" class="flex items-center gap-3 px-4 py-2 text-ink hover:bg-surface-sunken">
              <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-forest-soft text-forest">
                <bt-icon name="plus" [size]="18" />
              </span>
              <span class="flex min-w-0 flex-1 flex-col">
                <span class="text-body-strong">{{ 'nav.addFarm' | t }}</span>
                <span class="text-small text-ink-muted">{{ 'nav.addFarmHint' | t }}</span>
              </span>
            </a>
          } @else if (atFarmLimit()) {
            <p class="px-4 py-2 text-small text-ink-muted">{{ 'nav.farmLimitReached' | t: { count: farmLimit() } }}</p>
          }
          <div class="mt-2 border-t border-line"></div>
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
  private readonly seller = inject(SellerSpaceService);
  private readonly router = inject(Router);
  private readonly elementRef = inject(ElementRef<HTMLElement>);

  protected readonly open = signal(false);

  protected readonly displayName = computed(() => this.auth.user()?.displayName || this.auth.user()?.email || '');

  /** Buying is personal: a seller or a staff member can reserve too,
   * so these two never go away. */
  protected readonly roleLinks = computed<readonly RoleLink[]>(() => (this.auth.isAuthenticated() ? [MY_RESERVATIONS_LINK, FAVORITES_LINK] : []));

  /** Personal first, then each farm the account owns (opening one makes
   * it the farm the portal works on), then the back office for staff. */
  protected readonly spaces = computed<readonly Space[]>(() => {
    const list: Space[] = [{ id: 'personal', key: 'personal', route: '/', name: this.displayName(), hintKey: 'nav.personalSpace' }];
    for (const farm of this.seller.farms()) {
      list.push({ id: 'farm', key: farm.id, farmId: farm.id, route: '/portal', name: farm.name, hintKey: farmStatusEntry(farm.status).labelKey });
    }
    if (this.auth.isStaff()) {
      list.push({ id: 'admin', key: 'admin', route: '/admin', name: 'Betailo', hintKey: 'nav.admin' });
    }
    return list;
  });

  /** The API's call: under the account's farm limit, and not staff. */
  protected readonly canAddFarm = computed(() => this.seller.canAddFarm());
  /** At the limit: said rather than silently hiding "Add a farm". */
  protected readonly atFarmLimit = computed(() => this.seller.hasFarm() && !this.seller.canAddFarm());
  protected readonly farmLimit = computed(() => this.seller.farmLimit());

  protected isCurrent(space: Space): boolean {
    if (space.id !== this.current()) return false;
    return space.id !== 'farm' || space.farmId === this.seller.currentFarm()?.id;
  }

  protected enter(space: Space): void {
    if (space.farmId) this.seller.select(space.farmId);
    this.open.set(false);
  }

  protected readonly current = toSignal(
    this.router.events.pipe(
      filter((e) => e instanceof NavigationEnd),
      map(() => spaceOf(this.router.url))
    ),
    { initialValue: spaceOf(this.router.url) }
  );

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
