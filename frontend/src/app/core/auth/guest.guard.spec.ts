import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot, UrlTree, provideRouter } from '@angular/router';
import { Role } from '../../domain/enums';
import { AuthService } from './auth.service';
import { guestGuard } from './guest.guard';

describe('guestGuard', () => {
  const authenticated = signal(false);
  const role = signal<Role | null>(null);

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideRouter([]), { provide: AuthService, useValue: { isAuthenticated: authenticated, role } }]
    });
  });

  function run(): boolean | UrlTree {
    return TestBed.runInInjectionContext(() =>
      guestGuard({} as ActivatedRouteSnapshot, {} as RouterStateSnapshot)
    ) as boolean | UrlTree;
  }

  it('lets an anonymous visitor through', () => {
    authenticated.set(false);
    role.set(null);
    expect(run()).toBeTrue();
  });

  for (const [r, home] of [
    ['buyer', '/marketplace'],
    ['farmer', '/portal'],
    ['admin', '/admin']
  ] as const) {
    it(`sends a signed-in ${r} to ${home}`, () => {
      authenticated.set(true);
      role.set(r);
      const result = run();
      expect(result instanceof UrlTree).toBeTrue();
      expect(TestBed.inject(Router).serializeUrl(result as UrlTree)).toBe(home);
    });
  }
});
