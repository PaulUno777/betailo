import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { Role } from '../../domain/enums';
import { AuthService } from './auth.service';

/** `roleGuard('farmer')` — used alongside `authGuard` on routes that need
 * a specific role (e.g. the farmer portal), never in place of it. */
export function roleGuard(role: Role): CanActivateFn {
  return () => {
    const auth = inject(AuthService);
    if (auth.role() === role) {
      return true;
    }
    const router = inject(Router);
    return router.createUrlTree(['/']);
  };
}
