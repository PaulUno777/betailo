import { Injectable, computed, effect, inject, signal, untracked } from '@angular/core';
import { Account, Farm } from '../../domain/models';
import { AccountApi } from '../../data-access/account/account.api';
import { apiResource } from '../http/api-resource';
import { AuthService } from './auth.service';

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

  private readonly account = apiResource<Account>(() => (this.auth.isAuthenticated() ? this.accountApi.profileUrl() : undefined));
  private readonly chosenFarmId = signal<string | null>(null);

  /** False until the first answer for this session has arrived. */
  readonly loaded = computed(() => this.account.hasValue());
  readonly farms = computed<readonly Farm[]>(() => (this.auth.isAuthenticated() ? (this.account.value()?.farms ?? []) : []));
  readonly hasFarm = computed(() => this.farms().length > 0);
  readonly farmLimit = computed(() => this.account.value()?.farmLimit ?? 0);
  readonly canAddFarm = computed(() => this.auth.isAuthenticated() && (this.account.value()?.canAddFarm ?? false));

  /** The farm the portal is working on: the one chosen, else the first. */
  readonly currentFarm = computed<Farm | null>(() => {
    const farms = this.farms();
    return farms.find((f) => f.id === this.chosenFarmId()) ?? farms[0] ?? null;
  });

  constructor() {
    // A different account signed in: its own farms, its own choice.
    effect(() => {
      const userId = this.auth.user()?.id;
      untracked(() => {
        this.chosenFarmId.set(userId ? readStoredFarmId(userId) : null);
        if (userId) this.account.reload();
      });
    });
  }

  select(farmId: string): void {
    this.chosenFarmId.set(farmId);
    const userId = this.auth.user()?.id;
    if (!userId) return;
    try {
      localStorage.setItem(`${CURRENT_FARM_KEY}.${userId}`, farmId);
    } catch {
      // Remembering the choice is a convenience only.
    }
  }

  /** Read again: a farm was created or submitted, or staff decided. */
  refresh(): void {
    if (this.auth.isAuthenticated()) this.account.reload();
  }
}
