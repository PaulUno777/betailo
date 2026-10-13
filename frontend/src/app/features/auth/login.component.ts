import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AuthApi, LoginResult, MfaChallenge, isMfaChallenge } from '../../data-access/auth/auth.api';
import { passkeysSupported, usePasskey } from '../../core/auth/webauthn';
import { AuthService } from '../../core/auth/auth.service';
import { AppError } from '../../core/http/app-error';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { AuthLayoutComponent } from './auth-layout.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';

interface LoginForm {
  email: FormControl<string>;
  password: FormControl<string>;
}

@Component({
  selector: 'bt-login',
  imports: [ReactiveFormsModule, RouterLink, AuthLayoutComponent, TextFieldComponent, ButtonComponent, NoticeComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <bt-auth-layout>
      @if (challenge(); as c) {
        <!-- The password was right; the account asks for its second factor. -->
        <form (ngSubmit)="verify()" class="flex flex-col gap-5">
          <h2 class="text-heading-3">{{ 'auth.mfaTitle' | t }}</h2>
          <p class="-mt-2 text-small text-ink-muted">{{ (useRecovery() ? 'auth.mfaRecoveryIntro' : 'auth.mfaIntro') | t }}</p>
          @if (error(); as message) {
            <bt-notice tone="danger">{{ message | t }}</bt-notice>
          }
          @if (useRecovery() || c.methods.includes('totp')) {
            <bt-text-field
              [label]="(useRecovery() ? 'auth.mfaRecoveryLabel' : 'auth.mfaCodeLabel') | t"
              [inputmode]="useRecovery() ? 'text' : 'numeric'"
              autocomplete="one-time-code"
              markRequired
              [formControl]="code"
            />
            <bt-button type="submit" block size="lg" [disabled]="code.invalid || submitting()">
              {{ (submitting() ? 'auth.loginSubmitting' : 'auth.mfaSubmit') | t }}
            </bt-button>
          }
          @if (!useRecovery() && c.methods.includes('passkey') && passkeys) {
            <bt-button type="button" [variant]="c.methods.includes('totp') ? 'outline' : 'primary'" block size="lg" [disabled]="submitting()" (click)="verifyWithPasskey()">
              {{ 'auth.mfaPasskey' | t }}
            </bt-button>
          }
          <div class="flex flex-wrap justify-between gap-3 text-small">
            @if (c.methods.includes('recovery')) {
              <button type="button" class="min-h-11 font-semibold text-forest underline underline-offset-2" (click)="toggleRecovery()">
                {{ (useRecovery() ? 'auth.mfaUseApp' : 'auth.mfaUseRecovery') | t }}
              </button>
            }
            <button type="button" class="min-h-11 text-ink-muted underline underline-offset-2" (click)="cancelChallenge()">{{ 'auth.mfaBack' | t }}</button>
          </div>
        </form>
      } @else {
      <form [formGroup]="form" (ngSubmit)="submit()" class="flex flex-col gap-5">
        <h2 class="text-heading-3">{{ 'auth.loginTitle' | t }}</h2>
        @if (error(); as message) {
          <bt-notice tone="danger">{{ message | t }}</bt-notice>
        }
        <bt-text-field
          [label]="'auth.loginEmailLabel' | t"
          type="email"
          inputmode="email"
          autocomplete="email"
          markRequired
          formControlName="email"
          [error]="form.controls.email.touched && form.controls.email.invalid ? ('apiErrors.auth.email_invalid' | t) : undefined"
        />
        <bt-text-field [label]="'auth.loginPasswordLabel' | t" type="password" autocomplete="current-password" markRequired formControlName="password" />
        <label class="-my-2 flex min-h-11 cursor-pointer items-center gap-2 text-small text-ink-muted">
          <input
            type="checkbox"
            [checked]="remember()"
            (change)="remember.set(!remember())"
            class="h-[18px] w-[18px] rounded border-line-strong accent-forest"
          />
          {{ 'auth.rememberMe' | t }}
          <a routerLink="/forgot-password" class="ml-auto font-semibold text-forest underline underline-offset-2">{{ 'auth.forgotLink' | t }}</a>
        </label>
        <bt-button type="submit" block size="lg" [disabled]="form.invalid || submitting()">
          {{ (submitting() ? 'auth.loginSubmitting' : 'auth.loginSubmit') | t }}
        </bt-button>
        <p class="text-center text-small text-ink-muted">
          {{ 'auth.loginNoAccount' | t }}
          <a routerLink="/register" queryParamsHandling="preserve" class="font-semibold text-forest underline underline-offset-2">{{ 'auth.loginCreateAccount' | t }}</a>
        </p>
      </form>
      }
    </bt-auth-layout>
  `
})
export class LoginComponent {
  private readonly authApi = inject(AuthApi);
  private readonly authService = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  protected readonly submitting = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly remember = signal(true);

  protected readonly form = new FormGroup<LoginForm>({
    email: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.email] }),
    password: new FormControl('', { nonNullable: true, validators: [Validators.required] })
  });

  /** The pending sign-in when the account has a second factor. */
  protected readonly challenge = signal<MfaChallenge | null>(null);
  protected readonly useRecovery = signal(false);
  protected readonly code = new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.minLength(6)] });
  protected readonly passkeys = passkeysSupported();

  protected submit(): void {
    if (this.form.invalid || this.submitting()) {
      return;
    }
    this.submitting.set(true);
    this.error.set(null);
    const { email, password } = this.form.getRawValue();
    this.authApi.login({ email, password, remember: this.remember() }).subscribe({
      next: (result) => {
        if (isMfaChallenge(result)) {
          this.submitting.set(false);
          this.code.reset('');
          this.useRecovery.set(false);
          this.challenge.set(result);
          return;
        }
        this.enter(result);
      },
      error: (err: AppError) => {
        this.submitting.set(false);
        this.error.set(err.kind === 'unauthorized' ? 'auth.loginErrorInvalid' : err.message);
      }
    });
  }

  private enter({ accessToken, user }: LoginResult): void {
    this.authService.setSession(accessToken, user);
    // A staff member signs in to work: the back office is their home.
    const redirectTo = this.route.snapshot.queryParamMap.get('redirectTo') ?? (user.staffRole ? '/admin' : '/');
    this.router.navigateByUrl(redirectTo);
  }

  private fail(err: AppError): void {
    this.submitting.set(false);
    // An expired step can't be retried: back to the password.
    if (err.code === 'auth.mfa_challenge_expired') this.challenge.set(null);
    this.error.set(err.message);
  }

  protected verify(): void {
    const c = this.challenge();
    if (!c || this.code.invalid || this.submitting()) return;
    this.submitting.set(true);
    this.error.set(null);
    const value = this.code.value.trim();
    this.authApi.verifyMfa(c.mfaToken, this.useRecovery() ? { recoveryCode: value } : { code: value }).subscribe({
      next: (result) => this.enter(result),
      error: (err: AppError) => this.fail(err)
    });
  }

  protected verifyWithPasskey(): void {
    const c = this.challenge();
    if (!c || this.submitting()) return;
    this.submitting.set(true);
    this.error.set(null);
    this.authApi.mfaPasskeyOptions(c.mfaToken).subscribe({
      next: ({ options }) => {
        usePasskey(options.publicKey).then(
          (credential) =>
            this.authApi.verifyMfaPasskey(c.mfaToken, credential).subscribe({
              next: (result) => this.enter(result),
              error: (err: AppError) => this.fail(err)
            }),
          () => {
            // The person closed the browser's prompt, or no passkey matched.
            this.submitting.set(false);
            this.error.set('auth.mfaPasskeyCancelled');
          }
        );
      },
      error: (err: AppError) => this.fail(err)
    });
  }

  protected toggleRecovery(): void {
    this.useRecovery.set(!this.useRecovery());
    this.code.reset('');
    this.error.set(null);
  }

  protected cancelChallenge(): void {
    this.challenge.set(null);
    this.error.set(null);
  }
}
