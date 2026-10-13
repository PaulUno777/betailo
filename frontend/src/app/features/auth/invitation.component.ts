import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormControl, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AuthApi, InvitationPreview } from '../../data-access/auth/auth.api';
import { AuthService } from '../../core/auth/auth.service';
import { AppError } from '../../core/http/app-error';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { StaffRoleBadgeComponent } from '../../shared/ui/staff-role-badge/staff-role-badge.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { StaffRole } from '../../domain/enums';
import { AuthLayoutComponent } from './auth-layout.component';

/** `/invitation/:token` — where a staff invitation's link lands. The API
 * says who is invited and as what; a new address chooses a name and a
 * password and is signed in, an address that already has an account
 * proves it with its password and then signs in the usual way. Either
 * way the member is sent to set up their second factor next. */
@Component({
  selector: 'bt-invitation',
  imports: [ReactiveFormsModule, RouterLink, AuthLayoutComponent, ButtonComponent, NoticeComponent, StaffRoleBadgeComponent, TextFieldComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <bt-auth-layout>
      @if (preview(); as p) {
        <form (ngSubmit)="accept()" class="flex flex-col gap-5">
          <h2 class="text-heading-3">{{ 'invitation.title' | t }}</h2>
          <p class="-mt-2 flex flex-wrap items-center gap-2 text-small text-ink-muted">
            <span>{{ 'invitation.for' | t: { email: p.email } }}</span>
            <bt-staff-role-badge [role]="role(p)" />
          </p>
          @if (error(); as message) {
            <bt-notice tone="danger">{{ message }}</bt-notice>
          }
          @if (p.accountExists) {
            <bt-notice tone="info">{{ 'invitation.existing' | t }}</bt-notice>
          } @else {
            <bt-text-field [label]="'auth.registerNameLabel' | t" autocomplete="name" [formControl]="displayName" />
          }
          <bt-text-field
            [label]="(p.accountExists ? 'auth.loginPasswordLabel' : 'invitation.newPassword') | t"
            type="password"
            [autocomplete]="p.accountExists ? 'current-password' : 'new-password'"
            markRequired
            [formControl]="password"
          />
          <bt-button type="submit" block size="lg" [disabled]="password.invalid || submitting()">{{ 'invitation.accept' | t }}</bt-button>
        </form>
      } @else if (gone()) {
        <div class="flex flex-col gap-4">
          <h2 class="text-heading-3">{{ 'invitation.goneTitle' | t }}</h2>
          <p class="text-body text-ink-muted">{{ 'invitation.goneBody' | t }}</p>
          <a routerLink="/login" class="font-semibold text-forest underline underline-offset-2">{{ 'auth.loginSubmit' | t }}</a>
        </div>
      } @else {
        <div class="h-40 animate-pulse rounded-xl bg-surface-sunken" aria-busy="true"></div>
      }
    </bt-auth-layout>
  `
})
export class InvitationComponent {
  private readonly authApi = inject(AuthApi);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly token = inject(ActivatedRoute).snapshot.paramMap.get('token') ?? '';

  protected readonly preview = signal<InvitationPreview | null>(null);
  protected readonly gone = signal(false);
  protected readonly submitting = signal(false);
  protected readonly error = signal<string | null>(null);

  protected readonly displayName = new FormControl('', { nonNullable: true });
  protected readonly password = new FormControl('', { nonNullable: true, validators: [Validators.required] });

  constructor() {
    this.authApi.invitation(this.token).subscribe({
      next: (p) => this.preview.set(p),
      error: () => this.gone.set(true)
    });
  }

  protected role(p: InvitationPreview): StaffRole {
    return p.role as StaffRole;
  }

  protected accept(): void {
    if (this.password.invalid || this.submitting()) return;
    this.submitting.set(true);
    this.error.set(null);
    this.authApi.acceptInvitation(this.token, this.displayName.value.trim(), this.password.value).subscribe({
      next: (result) => {
        if (result.accessToken && result.user) {
          this.auth.setSession(result.accessToken, result.user);
          // A new member's first stop: the second factor the back office asks for.
          this.router.navigate(['/account/security'], { queryParams: { required: 1 } });
          return;
        }
        this.router.navigate(['/login'], { queryParams: { redirectTo: '/admin' } });
      },
      error: (err: AppError) => {
        this.submitting.set(false);
        if (err.code === 'auth.invitation_invalid') {
          this.preview.set(null);
          this.gone.set(true);
          return;
        }
        this.error.set(err.message);
      }
    });
  }
}
