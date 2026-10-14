import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { User } from '../../domain/models';
import { ApiClient } from '../../core/http/api-client';

export interface RegisterInput {
  readonly email: string;
  readonly password: string;
  readonly displayName?: string;
}

export interface LoginInput {
  readonly email: string;
  readonly password: string;
  /** Whether the refresh-token cookie the server sets gets a real
   * `Expires` (persists across browser restarts) or none (session-only
   * — see docs/adr/0008). Client-side storage no longer makes this
   * distinction itself; it's entirely the server's cookie now. */
  readonly remember: boolean;
}

export interface LoginResult {
  readonly accessToken: string;
  readonly user: User;
}

export interface RefreshResult {
  readonly accessToken: string;
}

/** `POST /auth/register` returns the created account, not a session —
 * the caller logs in with the same credentials to obtain a token.
 * `refresh`/`logout` both rely on the httpOnly `refresh_token` cookie
 * the browser already holds (see docs/adr/0008) — neither takes a body. */
@Injectable({ providedIn: 'root' })
export class AuthApi {
  private readonly api = inject(ApiClient);

  register(input: RegisterInput): Observable<User> {
    return this.api.post<User>('/auth/register', input);
  }

  login(input: LoginInput): Observable<LoginResult> {
    return this.api.post<LoginResult>('/auth/login', input);
  }

  refresh(): Observable<RefreshResult> {
    return this.api.post<RefreshResult>('/auth/refresh');
  }

  logout(): Observable<void> {
    return this.api.post<void>('/auth/logout');
  }
}
