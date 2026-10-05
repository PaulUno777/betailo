import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from './auth.service';

/** The marketing `Home` board only exists for anonymous visitors (its own
 * design has the "Se connecter" / "Créer un compte" header). A signed-in
 * user — buyer, farmer or admin — lands on `/marketplace` instead, since
 * they're never the marketing page's audience. */
export const homeRedirectGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  if (auth.isAuthenticated()) {
    const router = inject(Router);
    return router.createUrlTree(['/marketplace']);
  }
  return true;
};
