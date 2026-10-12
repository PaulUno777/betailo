import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { catchError, map, of } from 'rxjs';
import { AccountApi } from '../../data-access/account/account.api';
import { AuthService } from './auth.service';

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

/** The back office is for the platform's team. The API refuses every
 * `/admin` call from anyone else; this only keeps them off the page. */
export const staffGuard: CanActivateFn = () => {
  return inject(AuthService).isStaff() ? true : inject(Router).createUrlTree(['/']);
};
