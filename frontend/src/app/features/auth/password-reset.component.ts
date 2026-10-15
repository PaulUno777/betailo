import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormControl, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { AuthApi } from '../../data-access/auth/auth.api';
import { AppError } from '../../core/http/app-error';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { AuthLayoutComponent } from './auth-layout.component';

/** `/forgot-password` and `/reset-password/:token` — one page, two
 * steps. Without a token it asks for the address and always answers the
 * same thing, whether or not that address has an account. With the
 * token from the e-mail it takes the new password. */
@Component({
  selector: 'bt-password-reset',
  imports: [ReactiveFormsModule, RouterLink, AuthLayoutComponent, ButtonComponent, NoticeComponent, TextFieldComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <bt-auth-layout>
      @if (done()) {
        <div class="flex flex-col gap-4">
          <h2 class="text-heading-3">{{ (token ? 'passwordReset.changedTitle' : 'passwordReset.sentTitle') | t }}</h2>
          <p class="text-body text-ink-muted">{{ (token ? 'passwordReset.changedBody' : 'passwordReset.sentBody') | t }}</p>
          <a routerLink="/login" class="font-semibold text-forest underline underline-offset-2">{{ 'passwordReset.backToLogin' | t }}</a>
        </div>
      } @else if (token) {
        <form (submit)="$event.preventDefault(); reset()" class="flex flex-col gap-5">
          <h2 class="text-heading-3">{{ 'passwordReset.newTitle' | t }}</h2>
          @if (error(); as message) {
            <bt-notice tone="danger">{{ message }}</bt-notice>
          }
          <bt-text-field [label]="'passwordReset.newLabel' | t" type="password" autocomplete="new-password" markRequired [formControl]="password" />
          <bt-button type="submit" block size="lg" [disabled]="password.invalid || submitting()">{{ 'passwordReset.newSubmit' | t }}</bt-button>
        </form>
      } @else {
        <form (submit)="$event.preventDefault(); request()" class="flex flex-col gap-5">
          <h2 class="text-heading-3">{{ 'passwordReset.title' | t }}</h2>
          <p class="-mt-2 text-small text-ink-muted">{{ 'passwordReset.intro' | t }}</p>
          @if (error(); as message) {
            <bt-notice tone="danger">{{ message }}</bt-notice>
          }
          <bt-text-field [label]="'auth.loginEmailLabel' | t" type="email" inputmode="email" autocomplete="email" markRequired [formControl]="email" />
          <bt-button type="submit" block size="lg" [disabled]="email.invalid || submitting()">{{ 'passwordReset.submit' | t }}</bt-button>
          <a routerLink="/login" class="text-center text-small font-semibold text-forest underline underline-offset-2">{{ 'passwordReset.backToLogin' | t }}</a>
        </form>
      }
    </bt-auth-layout>
  `
})
export class PasswordResetComponent {
  private readonly authApi = inject(AuthApi);
  protected readonly token = inject(ActivatedRoute).snapshot.paramMap.get('token') ?? '';

  protected readonly email = new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.email] });
  protected readonly password = new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.minLength(8)] });
  protected readonly submitting = signal(false);
  protected readonly done = signal(false);
  protected readonly error = signal<string | null>(null);

  private finish = {
    next: () => {
      this.submitting.set(false);
      this.done.set(true);
    },
    error: (err: AppError) => {
      this.submitting.set(false);
      this.error.set(err.message);
    }
  };

  protected request(): void {
    if (this.email.invalid || this.submitting()) return;
    this.submitting.set(true);
    this.error.set(null);
    this.authApi.forgotPassword(this.email.value).subscribe(this.finish);
  }

  protected reset(): void {
    if (this.password.invalid || this.submitting()) return;
    this.submitting.set(true);
    this.error.set(null);
    this.authApi.resetPassword(this.token, this.password.value).subscribe(this.finish);
  }
}
