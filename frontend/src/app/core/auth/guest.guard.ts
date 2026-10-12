import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from './auth.service';

/** Keeps a signed-in visitor out of `/login` and `/register`, sending
 * them where they work instead: the back office for staff, the
 * marketplace for everyone else. */
export const guestGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  if (!auth.isAuthenticated()) {
    return true;
  }
  return inject(Router).createUrlTree([auth.isStaff() ? '/admin' : '/marketplace']);
};
