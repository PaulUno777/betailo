import { DOCUMENT } from '@angular/common';
import { DestroyRef, Injectable, effect, inject, signal } from '@angular/core';
import { NavigationEnd, Router } from '@angular/router';
import { filter } from 'rxjs';
import { AuthService } from '../auth/auth.service';
import { AccountApi, NavCounts } from '../../data-access/account/account.api';

const POLL_MS = 60_000;
/** Navigating often (a farmer moving between tabs) re-reads the counts
 * at most this often. */
const MIN_GAP_MS = 15_000;

const ZERO: NavCounts = { buyer: 0, seller: 0 };

/** The badges on "Mes achats" and "Ventes": how many reservations wait
 * on the user, decided server-side. Read on sign-in, after navigating
 * (an action on one page usually changes them), when the tab comes
 * back to the foreground, and once a minute otherwise. */
@Injectable({ providedIn: 'root' })
export class NavCountsService {
  private readonly api = inject(AccountApi);
  private readonly auth = inject(AuthService);
  private readonly document = inject(DOCUMENT);

  private readonly state = signal<NavCounts>(ZERO);
  readonly counts = this.state.asReadonly();
  private lastRead = 0;

  constructor() {
    const timer = setInterval(() => this.refresh(true), POLL_MS);
    const onVisible = () => {
      if (this.document.visibilityState === 'visible') this.refresh(true);
    };
    this.document.addEventListener('visibilitychange', onVisible);
    const nav = inject(Router)
      .events.pipe(filter((e) => e instanceof NavigationEnd))
      .subscribe(() => this.refresh());
    inject(DestroyRef).onDestroy(() => {
      clearInterval(timer);
      this.document.removeEventListener('visibilitychange', onVisible);
      nav.unsubscribe();
    });
    effect(() => {
      if (this.auth.isAuthenticated()) {
        this.refresh(true);
      } else {
        this.state.set(ZERO);
      }
    });
  }

  refresh(force = false): void {
    if (!this.auth.isAuthenticated() || this.document.visibilityState === 'hidden') return;
    const now = Date.now();
    if (!force && now - this.lastRead < MIN_GAP_MS) return;
    this.lastRead = now;
    this.api.navCounts().subscribe({
      next: (c) => this.state.set(c),
      error: () => {
        // A missed read is harmless; the next one catches up.
      }
    });
  }
}
