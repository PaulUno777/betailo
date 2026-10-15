import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { Account, Profile } from '../../domain/models';
import { ApiClient } from '../../core/http/api-client';
import { LoginResult } from '../auth/auth.api';
import { PhotoSizes, PresignedUpload } from '../../core/http/photo-upload.service';

export interface ProfileInput {
  readonly displayName: string;
  /** As typed; the API normalizes it to E.164 and validates it. */
  readonly phone: string;
  readonly city: string;
}

/** `GET /me/nav-counts`: reservations waiting on the user, per side. */
export interface NavCounts {
  readonly buyer: number;
  readonly seller: number;
  /** The seller's planned or running cycles. */
  readonly cycles: number;
}

export interface ChangePasswordInput {
  readonly currentPassword: string;
  readonly newPassword: string;
  readonly remember: boolean;
}

/** The signed-in user's own account: profile and password. */
@Injectable({ providedIn: 'root' })
export class AccountApi {
  private readonly api = inject(ApiClient);

  profileUrl(): string {
    return this.api.url('/me');
  }

  profile(): Observable<Account> {
    return this.api.get<Account>('/me');
  }

  navCounts(): Observable<NavCounts> {
    return this.api.get<NavCounts>('/me/nav-counts');
  }

  updateProfile(input: ProfileInput): Observable<Profile> {
    return this.api.patch<Profile>('/me', input);
  }

  presignAvatar(sizes: PhotoSizes): Observable<PresignedUpload> {
    return this.api.post<PresignedUpload>('/me/avatar/presign', sizes);
  }

  /** Makes an uploaded photo the profile photo. */
  setAvatar(key: string): Observable<Profile> {
    return this.api.put<Profile>('/me/avatar', { key });
  }

  removeAvatar(): Observable<Profile> {
    return this.api.delete<Profile>('/me/avatar');
  }

  /** Ends every other session; this device is signed straight back in
   * with the new password, so the result is a fresh session. */
  changePassword(input: ChangePasswordInput): Observable<LoginResult> {
    return this.api.post<LoginResult>('/me/password', input);
  }
}
