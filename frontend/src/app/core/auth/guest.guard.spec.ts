import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot, UrlTree, provideRouter } from '@angular/router';
import { AuthService } from './auth.service';
import { guestGuard } from './guest.guard';

describe('guestGuard', () => {
  const authenticated = signal(false);
  const staff = signal(false);

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideRouter([]), { provide: AuthService, useValue: { isAuthenticated: authenticated, isStaff: staff } }]
    });
  });

  function run(): boolean | UrlTree {
    return TestBed.runInInjectionContext(() =>
      guestGuard({} as ActivatedRouteSnapshot, {} as RouterStateSnapshot)
    ) as boolean | UrlTree;
  }

  it('lets an anonymous visitor through', () => {
    authenticated.set(false);
    staff.set(false);
    expect(run()).toBeTrue();
  });

  for (const [who, isStaff, home] of [
    ['personal account', false, '/marketplace'],
    ['staff member', true, '/admin']
  ] as const) {
    it(`sends a signed-in ${who} to ${home}`, () => {
      authenticated.set(true);
      staff.set(isStaff);
      const result = run();
      expect(result instanceof UrlTree).toBeTrue();
      expect(TestBed.inject(Router).serializeUrl(result as UrlTree)).toBe(home);
    });
  }
});
