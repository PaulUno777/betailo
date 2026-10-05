import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { Role } from '../../domain/enums';
import { AuthService } from './auth.service';

const ROLE_HOME: Readonly<Record<Role, string>> = {
  buyer: '/marketplace',
  farmer: '/portal',
  admin: '/admin'
};

/** For /login and /register: a visitor who's already signed in has
 * nothing to do there, so send them to their role's home instead of
 * showing a form that would silently replace their session. */
export const guestGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const role = auth.role();
  if (!auth.isAuthenticated() || !role) {
    return true;
  }
  return inject(Router).createUrlTree([ROLE_HOME[role]]);
};
