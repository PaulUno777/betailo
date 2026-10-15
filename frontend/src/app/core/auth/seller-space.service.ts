import { Injectable, computed, effect, inject, signal, untracked } from '@angular/core';
import { Account, Farm } from '../../domain/models';
import { AccountApi } from '../../data-access/account/account.api';
import { apiResource } from '../http/api-resource';
import { AuthService } from './auth.service';

/** Stored in place of a farm id when the combined view is picked. */
const ALL_FARMS = 'all';
const CURRENT_FARM_KEY = 'betailo.currentFarm';

function readStoredFarmId(userId: string): string | null {
  try {
    return localStorage.getItem(`${CURRENT_FARM_KEY}.${userId}`);
  } catch {
    return null;
  }
}

/** The seller side of an account. Everyone has a personal account;
 * opening a farm is what makes one a seller, and each farm sells once
 * staff approved it. An account may own several farms (up to the limit
 * the API gives it): the portal works on one at a time, the "current"
 * farm, remembered per account on this device.
 *
 * Everything here is the API's answer (`GET /me`): which farms, their
 * review state, whether another can be added. This service only reads
 * it for the menus, the portal and the banners. */
@Injectable({ providedIn: 'root' })
export class SellerSpaceService {
  private readonly auth = inject(AuthService);
  private readonly accountApi = inject(AccountApi);

  private readonly resource = apiResource<Account>(() => (this.auth.isAuthenticated() ? this.accountApi.profileUrl() : undefined));
  private readonly chosenFarmId = signal<string | null>(null);
  /** "Toutes mes fermes" is picked (the default with several farms). */
  private readonly allChosen = signal(true);

  /** False until the first answer for this session has arrived. */
  readonly loaded = computed(() => this.resource.hasValue());
  /** The whole `GET /me` answer, for what isn't about selling (the
   * staff role and its permissions, see StaffSpaceService). */
  readonly account = computed<Account | null>(() => (this.auth.isAuthenticated() ? (this.resource.value() ?? null) : null));
  readonly farms = computed<readonly Farm[]>(() => this.account()?.farms ?? []);
  readonly hasFarm = computed(() => this.farms().length > 0);
  readonly farmLimit = computed(() => this.account()?.farmLimit ?? 0);
  readonly canAddFarm = computed(() => this.account()?.canAddFarm ?? false);

  /** The farm the portal is working on: the one chosen, else the first. */
  readonly currentFarm = computed<Farm | null>(() => {
    const farms = this.farms();
    return farms.find((f) => f.id === this.chosenFarmId()) ?? farms[0] ?? null;
  });

  /** The portal shows every farm together: the dashboard sums them.
   * Pages that work on one farm still use `currentFarm`. */
  readonly allFarms = computed(() => this.allChosen() && this.farms().length > 1);

  constructor() {
    // A different account signed in: its own farms, its own choice.
    const signedInId = computed(() => this.auth.user()?.id);
    effect(() => {
      const userId = signedInId();
      untracked(() => {
        const stored = userId ? readStoredFarmId(userId) : null;
        this.allChosen.set(stored === null || stored === ALL_FARMS);
        this.chosenFarmId.set(stored === ALL_FARMS ? null : stored);
        if (userId) this.resource.reload();
      });
    });

    // The account as the API sees it now is the reference for the role
    // and the profile photo.
    effect(() => {
      const account = this.resource.value();
      if (account && this.auth.isAuthenticated()) untracked(() => this.auth.syncAccount(account));
    });
  }

  select(farmId: string): void {
    this.chosenFarmId.set(farmId);
    this.allChosen.set(false);
    this.remember(farmId);
  }

  /** Back to the combined view. */
  selectAll(): void {
    this.allChosen.set(true);
    this.remember(ALL_FARMS);
  }

  private remember(choice: string): void {
    const userId = this.auth.user()?.id;
    if (!userId) return;
    try {
      localStorage.setItem(`${CURRENT_FARM_KEY}.${userId}`, choice);
    } catch {
      // Remembering the choice is a convenience only.
    }
  }

  /** Read again: a farm was created or submitted, or staff decided. */
  refresh(): void {
    if (this.auth.isAuthenticated()) this.resource.reload();
  }
}
