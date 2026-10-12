import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AuthApi } from '../../data-access/auth/auth.api';
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
      <form [formGroup]="form" (ngSubmit)="submit()" class="flex flex-col gap-5">
        <h2 class="text-heading-3">{{ 'auth.loginTitle' | t }}</h2>
        @if (error(); as message) {
          <bt-notice tone="danger">{{ message | t }}</bt-notice>
        }
        <bt-text-field [label]="'auth.loginEmailLabel' | t" type="email" autocomplete="email" formControlName="email" />
        <bt-text-field [label]="'auth.loginPasswordLabel' | t" type="password" autocomplete="current-password" formControlName="password" />
        <label class="flex items-center gap-2 text-small text-ink-muted">
          <input
            type="checkbox"
            [checked]="remember()"
            (change)="remember.set(!remember())"
            class="h-[18px] w-[18px] rounded border-line-strong accent-forest"
          />
          {{ 'auth.rememberMe' | t }}
        </label>
        <bt-button type="submit" block size="lg" [disabled]="form.invalid || submitting()">
          {{ (submitting() ? 'auth.loginSubmitting' : 'auth.loginSubmit') | t }}
        </bt-button>
        <p class="text-center text-small text-ink-muted">
          {{ 'auth.loginNoAccount' | t }}
          <a routerLink="/register" class="text-forest">{{ 'auth.loginCreateAccount' | t }}</a>
        </p>
      </form>
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

  protected submit(): void {
    if (this.form.invalid || this.submitting()) {
      return;
    }
    this.submitting.set(true);
    this.error.set(null);
    const { email, password } = this.form.getRawValue();
    this.authApi.login({ email, password, remember: this.remember() }).subscribe({
      next: ({ accessToken, user }) => {
        this.authService.setSession(accessToken, user);
        const redirectTo = this.route.snapshot.queryParamMap.get('redirectTo') ?? '/';
        this.router.navigateByUrl(redirectTo);
      },
      error: (err: AppError) => {
        this.submitting.set(false);
        this.error.set(err.kind === 'unauthorized' ? 'auth.loginErrorInvalid' : err.message);
      }
    });
  }
}
