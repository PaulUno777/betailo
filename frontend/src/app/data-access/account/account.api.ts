import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { Farm, Profile } from '../../domain/models';
import { ApiClient } from '../../core/http/api-client';
import { FarmInput } from '../farms/farms.api';
import { LoginResult } from '../auth/auth.api';

export interface ProfileInput {
  readonly displayName: string;
  /** As typed; the API normalizes it to E.164 and validates it. */
  readonly phone: string;
  readonly city: string;
}

export interface ChangePasswordInput {
  readonly currentPassword: string;
  readonly newPassword: string;
  readonly remember: boolean;
}

/** The signed-in user's own account: profile, password, and the switch
 * from buyer to seller. */
@Injectable({ providedIn: 'root' })
export class AccountApi {
  private readonly api = inject(ApiClient);

  profileUrl(): string {
    return this.api.url('/me');
  }

  profile(): Observable<Profile> {
    return this.api.get<Profile>('/me');
  }

  updateProfile(input: ProfileInput): Observable<Profile> {
    return this.api.patch<Profile>('/me', input);
  }

  /** Ends every other session; this device is signed straight back in
   * with the new password, so the result is a fresh session. */
  changePassword(input: ChangePasswordInput): Observable<LoginResult> {
    return this.api.post<LoginResult>('/me/password', input);
  }

  /** Creates the farm and makes the account a farmer, in one step. The
   * current access token still says "buyer": refresh the session next. */
  becomeSeller(farm: FarmInput): Observable<Farm> {
    return this.api.post<Farm>('/me/become-seller', { farm });
  }
}
