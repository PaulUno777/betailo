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

/** A right password on an account with a second factor: no session
 * yet. `mfaToken` names the pending sign-in for its second step. */
export interface MfaChallenge {
  readonly mfaRequired: true;
  readonly mfaToken: string;
  readonly methods: readonly MfaMethod[];
}

export type MfaMethod = 'totp' | 'passkey' | 'recovery';

export function isMfaChallenge(result: LoginResult | MfaChallenge): result is MfaChallenge {
  return (result as MfaChallenge).mfaRequired === true;
}

export interface PasskeyInfo {
  readonly id: string;
  readonly name?: string;
  readonly createdAt: string;
}

export interface MfaStatus {
  readonly totp: boolean;
  readonly passkeys: readonly PasskeyInfo[];
  readonly recoveryCodes: number;
  readonly enabled: boolean;
  /** This session passed a second factor. */
  readonly sessionVerified?: boolean;
  /** This deployment can register passkeys. */
  readonly passkeysAvailable?: boolean;
}

export interface TotpSetup {
  readonly secret: string;
  readonly url: string;
  /** The same address as a QR code, a PNG data URL. */
  readonly qr: string;
}

/** After a factor is added: recovery codes shown once, and an access
 * token for a session that now counts as verified. */
export interface Enrolled {
  readonly recoveryCodes?: readonly string[];
  readonly accessToken: string;
  readonly status: MfaStatus;
}

export interface PasskeyOptions {
  readonly options: { publicKey: Record<string, unknown> };
  readonly flowToken?: string;
}

export interface InvitationPreview {
  readonly email: string;
  readonly role: string;
  readonly accountExists: boolean;
}

/** A new account is signed in at once; an existing one signs in itself. */
export interface InvitationAccepted {
  readonly accessToken?: string;
  readonly user?: User;
  readonly signIn: boolean;
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

  login(input: LoginInput): Observable<LoginResult | MfaChallenge> {
    return this.api.post<LoginResult | MfaChallenge>('/auth/login', input);
  }

  /** The second step with a typed code: authenticator or recovery. */
  verifyMfa(mfaToken: string, code: { code?: string; recoveryCode?: string }): Observable<LoginResult> {
    return this.api.post<LoginResult>('/auth/mfa/verify', { mfaToken, ...code });
  }

  mfaPasskeyOptions(mfaToken: string): Observable<PasskeyOptions> {
    return this.api.post<PasskeyOptions>('/auth/mfa/passkey/options', { mfaToken });
  }

  verifyMfaPasskey(mfaToken: string, credential: unknown): Observable<LoginResult> {
    return this.api.post<LoginResult>('/auth/mfa/passkey/verify', { mfaToken, credential });
  }

  mfaStatusUrl(): string {
    return this.api.url('/me/mfa');
  }

  setupTotp(): Observable<TotpSetup> {
    return this.api.post<TotpSetup>('/me/mfa/totp/setup');
  }

  confirmTotp(code: string): Observable<Enrolled> {
    return this.api.post<Enrolled>('/me/mfa/totp/confirm', { code });
  }

  disableTotp(password: string): Observable<MfaStatus> {
    return this.api.post<MfaStatus>('/me/mfa/totp/disable', { password });
  }

  regenerateRecoveryCodes(password: string): Observable<{ recoveryCodes: string[] }> {
    return this.api.post<{ recoveryCodes: string[] }>('/me/mfa/recovery-codes', { password });
  }

  passkeyOptions(): Observable<PasskeyOptions> {
    return this.api.post<PasskeyOptions>('/me/mfa/passkeys/options');
  }

  addPasskey(flowToken: string, name: string, credential: unknown): Observable<Enrolled> {
    return this.api.post<Enrolled>('/me/mfa/passkeys', { flowToken, name, credential });
  }

  removePasskey(id: string, password: string): Observable<MfaStatus> {
    return this.api.post<MfaStatus>('/me/mfa/passkeys/remove', { id, password });
  }

  invitation(token: string): Observable<InvitationPreview> {
    return this.api.get<InvitationPreview>(`/auth/invitations/${encodeURIComponent(token)}`);
  }

  acceptInvitation(token: string, displayName: string, password: string): Observable<InvitationAccepted> {
    return this.api.post<InvitationAccepted>('/auth/invitations/accept', { token, displayName, password });
  }

  /** Always succeeds: the answer never says whether the address exists. */
  forgotPassword(email: string): Observable<void> {
    return this.api.post<void>('/auth/password/forgot', { email });
  }

  resetPassword(token: string, password: string): Observable<void> {
    return this.api.post<void>('/auth/password/reset', { token, password });
  }

  refresh(): Observable<RefreshResult> {
    return this.api.post<RefreshResult>('/auth/refresh');
  }

  logout(): Observable<void> {
    return this.api.post<void>('/auth/logout');
  }
}
