import { Injectable, computed, inject } from '@angular/core';
import { Farm, Profile } from '../../domain/models';
import { AccountApi } from '../../data-access/account/account.api';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { apiResource } from '../http/api-resource';
import { AuthService } from './auth.service';

/** The seller side of an account: its farm, and whether an admin has
 * verified the seller yet. Everyone signs up with a personal account;
 * adding a farm opens this space, and verification is what lets its
 * listings go live. Both facts are the API's (`GET /farms/mine`,
 * `GET /me`): this only reads them for the menus and the banner. */
@Injectable({ providedIn: 'root' })
export class SellerSpaceService {
  private readonly auth = inject(AuthService);
  private readonly accountApi = inject(AccountApi);
  private readonly farmsApi = inject(FarmsApi);

  private readonly isSeller = computed(() => this.auth.role() === 'farmer');

  private readonly farms = apiResource<Farm[]>(() => (this.isSeller() ? this.farmsApi.mineUrl() : undefined));
  private readonly profile = apiResource<Profile>(() => (this.isSeller() ? this.accountApi.profileUrl() : undefined));

  readonly farm = computed(() => (this.isSeller() ? (this.farms.value()?.[0] ?? null) : null));

  /** A seller who can prepare cycles and listings but not publish yet. */
  readonly pendingVerification = computed(() => this.isSeller() && this.profile.value()?.verified === false);

  /** Read again: an admin may have verified the seller since. */
  refresh(): void {
    if (this.isSeller()) {
      this.profile.reload();
      this.farms.reload();
    }
  }
}
