import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { AbstractControl, FormControl, FormGroup, ReactiveFormsModule, ValidationErrors, Validators } from '@angular/forms';
import { toSignal } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { Role } from '../../domain/enums';
import { passwordStrength } from '../../domain/rules/password-strength';
import { AuthApi } from '../../data-access/auth/auth.api';
import { AuthService } from '../../core/auth/auth.service';
import { AppError } from '../../core/http/app-error';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { AuthLayoutComponent } from './auth-layout.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { ChoiceCardComponent } from '../../shared/ui/choice-card/choice-card.component';

interface RegisterForm {
  role: FormControl<Role>;
  displayName: FormControl<string>;
  email: FormControl<string>;
  password: FormControl<string>;
  confirmPassword: FormControl<string>;
  acceptTerms: FormControl<boolean>;
}

const MIN_PASSWORD_LENGTH = 8;

function passwordsMatchValidator(control: AbstractControl): ValidationErrors | null {
  const group = control as FormGroup<RegisterForm>;
  return group.controls.password.value === group.controls.confirmPassword.value ? null : { passwordMismatch: true };
}

@Component({
  selector: 'bt-register',
  imports: [
    ReactiveFormsModule,
    RouterLink,
    AuthLayoutComponent,
    TextFieldComponent,
    ButtonComponent,
    NoticeComponent,
    ChoiceCardComponent,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <bt-auth-layout>
      <form [formGroup]="form" (ngSubmit)="submit()" class="flex flex-col gap-5">
        <h2 class="text-heading-3">{{ 'auth.registerTitle' | t }}</h2>
        @if (error(); as message) {
          <bt-notice tone="danger">{{ message | t }}</bt-notice>
        }
        <div role="radiogroup" [attr.aria-label]="'auth.registerRoleLegend' | t" class="flex flex-col gap-3">
          <bt-choice-card
            name="role"
            value="buyer"
            [title]="'auth.registerRoleBuyerTitle' | t"
            [description]="'auth.registerRoleBuyerDesc' | t"
            [checked]="form.controls.role.value === 'buyer'"
            (selected)="form.controls.role.setValue('buyer')"
          />
          <bt-choice-card
            name="role"
            value="farmer"
            [title]="'auth.registerRoleFarmerTitle' | t"
            [description]="'auth.registerRoleFarmerDesc' | t"
            [checked]="form.controls.role.value === 'farmer'"
            (selected)="form.controls.role.setValue('farmer')"
          />
        </div>
        <bt-text-field
          [label]="'auth.registerNameLabel' | t"
          [hint]="'auth.registerNameHint' | t"
          autocomplete="name"
          formControlName="displayName"
        />
        <bt-text-field
          [label]="'auth.registerEmailLabel' | t"
          type="email"
          inputmode="email"
          autocomplete="email"
          markRequired
          formControlName="email"
          [error]="form.controls.email.touched && form.controls.email.invalid ? ('apiErrors.auth.email_invalid' | t) : undefined"
        />
        <div class="flex flex-col gap-1.5">
          <bt-text-field
            [label]="'auth.registerPasswordLabel' | t"
            type="password"
            autocomplete="new-password"
            markRequired
            formControlName="password"
            [hint]="'apiErrors.auth.password_too_short' | t: { min: minPasswordLength }"
            [error]="
              form.controls.password.touched && form.controls.password.invalid
                ? ('apiErrors.auth.password_too_short' | t: { min: minPasswordLength })
                : undefined
            "
          />
          @if (passwordValue()) {
            <div class="flex items-center gap-2">
              <div class="flex h-1 flex-1 gap-1">
                <span class="flex-1 rounded-full" [class]="strengthBarClass(0)"></span>
                <span class="flex-1 rounded-full" [class]="strengthBarClass(1)"></span>
                <span class="flex-1 rounded-full" [class]="strengthBarClass(2)"></span>
              </div>
              <span class="text-caption text-ink-muted" aria-live="polite">{{ strengthLabelKey() | t }}</span>
            </div>
          }
        </div>
        <bt-text-field
          [label]="'auth.registerConfirmLabel' | t"
          type="password"
          autocomplete="new-password"
          markRequired
          formControlName="confirmPassword"
          [error]="form.errors?.['passwordMismatch'] && form.controls.confirmPassword.touched ? ('auth.registerPasswordMismatch' | t) : undefined"
        />
        <label class="-my-2 flex min-h-11 cursor-pointer items-start gap-2 py-2.5 text-small text-ink-muted">
          <input
            type="checkbox"
            formControlName="acceptTerms"
            class="mt-0.5 h-[18px] w-[18px] shrink-0 rounded border-line-strong accent-forest"
          />
          {{ 'auth.registerTermsLabel' | t }}
        </label>
        <bt-button type="submit" block size="lg" [disabled]="form.invalid || submitting()">
          {{ (submitting() ? 'auth.registerSubmitting' : 'auth.registerSubmit') | t }}
        </bt-button>
        <p class="text-center text-small text-ink-muted">
          {{ 'auth.registerHasAccount' | t }}
          <a routerLink="/login" queryParamsHandling="preserve" class="font-semibold text-forest underline underline-offset-2">{{ 'auth.registerLogin' | t }}</a>
        </p>
      </form>
    </bt-auth-layout>
  `
})
export class RegisterComponent {
  private readonly authApi = inject(AuthApi);
  private readonly authService = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  protected readonly submitting = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly minPasswordLength = MIN_PASSWORD_LENGTH;

  protected readonly form = new FormGroup<RegisterForm>(
    {
      role: new FormControl<Role>('buyer', { nonNullable: true }),
      displayName: new FormControl('', { nonNullable: true, validators: [Validators.maxLength(60)] }),
      email: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.email] }),
      password: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.minLength(MIN_PASSWORD_LENGTH)] }),
      confirmPassword: new FormControl('', { nonNullable: true, validators: [Validators.required] }),
      acceptTerms: new FormControl(false, { nonNullable: true, validators: [Validators.requiredTrue] })
    },
    { validators: passwordsMatchValidator }
  );

  protected readonly passwordValue = toSignal(this.form.controls.password.valueChanges, { initialValue: '' });
  protected readonly strength = computed(() => passwordStrength(this.passwordValue()));
  private readonly STRENGTH_LEVEL: Readonly<Record<ReturnType<typeof passwordStrength>, number>> = { weak: 1, medium: 2, strong: 3 };

  protected strengthBarClass(index: number): string {
    const filled = index < this.STRENGTH_LEVEL[this.strength()];
    if (!filled) return 'bg-line';
    return this.strength() === 'weak' ? 'bg-alert' : this.strength() === 'medium' ? 'bg-amber' : 'bg-forest';
  }

  protected strengthLabelKey(): string {
    return `auth.passwordStrength${this.strength()[0].toUpperCase()}${this.strength().slice(1)}`;
  }

  protected submit(): void {
    if (this.form.invalid || this.submitting()) {
      return;
    }
    this.submitting.set(true);
    this.error.set(null);
    const { role, displayName, email, password } = this.form.getRawValue();
    this.authApi.register({ email, password, role, displayName: displayName.trim() || undefined }).subscribe({
      next: () => {
        this.authApi.login({ email, password, remember: true }).subscribe({
          next: ({ accessToken, user }) => {
            this.authService.setSession(accessToken, user);
            // Back to what they were doing (Vendre, a checkout…); a new
            // farmer otherwise goes straight to presenting their farm.
            const redirectTo = this.route.snapshot.queryParamMap.get('redirectTo');
            this.router.navigateByUrl(redirectTo ?? (role === 'farmer' ? '/sell' : '/'));
          },
          error: (err: AppError) => {
            this.submitting.set(false);
            this.error.set(err.message);
          }
        });
      },
      error: (err: AppError) => {
        this.submitting.set(false);
        this.error.set(err.kind === 'conflict' ? 'auth.registerErrorConflict' : err.message);
      }
    });
  }
}
