import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { catchError, map, of } from 'rxjs';
import { AccountApi } from '../../data-access/account/account.api';
import { Permission } from '../../domain/enums';

/** The seller portal is for accounts that own a farm; anyone else is
 * sent to open one. Asked of the API each time (`GET /me`), so it is
 * right the moment a farm is created. Used alongside `authGuard`. */
export const sellerGuard: CanActivateFn = () => {
  const router = inject(Router);
  return inject(AccountApi)
    .profile()
    .pipe(
      map((account) => (account.farms.length ? true : router.createUrlTree(['/sell']))),
      catchError(() => of(router.createUrlTree(['/'])))
    );
};

/** The back office is for the platform's team. Asked of the API each
 * time (`GET /me`), like the seller space: a role granted since the
 * last sign-in opens it, a revoked one closes it. The API refuses every
 * `/admin` call from anyone else; this only keeps them off the page. */
export const staffGuard: CanActivateFn = () => {
  const router = inject(Router);
  return inject(AccountApi)
    .profile()
    .pipe(
      map((account) => (account.staffRole ? true : router.createUrlTree(['/']))),
      catchError(() => of(router.createUrlTree(['/'])))
    );
};

/** A back-office page that needs one permission. Staff without it land
 * on the overview, which every role may open. */
export function permissionGuard(permission: Permission): CanActivateFn {
  return () => {
    const router = inject(Router);
    return inject(AccountApi)
      .profile()
      .pipe(
        map((account) => (account.permissions.includes(permission) ? true : router.createUrlTree(['/admin']))),
        catchError(() => of(router.createUrlTree(['/'])))
      );
  };
}
