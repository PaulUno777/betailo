import { Injectable, computed, inject } from '@angular/core';
import { Permission, StaffRole } from '../../domain/enums';
import { SellerSpaceService } from './seller-space.service';

/** The staff side of an account: its role and what that role may do in
 * the back office, as the API answers on `GET /me`. The back office
 * draws its navigation and its buttons from `can()`; the API refuses
 * the rest whatever the page shows.
 *
 * Read from the account, not from the cached sign-in: a role granted or
 * revoked since shows up on the next read. */
@Injectable({ providedIn: 'root' })
export class StaffSpaceService {
  private readonly space = inject(SellerSpaceService);

  readonly loaded = this.space.loaded;
  readonly role = computed<StaffRole | null>(() => this.space.account()?.staffRole ?? null);
  readonly isRoot = computed(() => this.space.account()?.isRoot === true);
  readonly email = computed(() => this.space.account()?.email ?? '');
  readonly name = computed(() => this.space.account()?.displayName || this.space.account()?.email || '');
  private readonly permissions = computed(() => new Set<Permission>(this.space.account()?.permissions ?? []));

  can(permission: Permission): boolean {
    return this.permissions().has(permission);
  }

  refresh(): void {
    this.space.refresh();
  }
}
