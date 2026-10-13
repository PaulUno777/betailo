import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { map } from 'rxjs';
import { AuthApi, Enrolled, MfaStatus, PasskeyInfo, TotpSetup } from '../../data-access/auth/auth.api';
import { AuthService } from '../../core/auth/auth.service';
import { createPasskey, passkeysSupported } from '../../core/auth/webauthn';
import { apiResource } from '../../core/http/api-resource';
import { AppError } from '../../core/http/app-error';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ToastService } from '../../core/notifications/toast.service';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';

/** What the password dialog is about to do once the password is typed. */
type Guarded = { kind: 'disable-totp' } | { kind: 'remove-passkey'; passkey: PasskeyInfo } | { kind: 'recovery-codes' };

/** `/account/security` — the account's second factors: an authenticator
 * app, passkeys, and the recovery codes that stand in for both.
 *
 * Staff reach it from the back office when the platform requires a
 * second factor (`?required=1`): the API refuses their requests until
 * one is set up, or until they sign in again with the one they have.
 * Everything here is the API's: which factors exist, the QR code, the
 * codes, and the upgraded session once a factor is added. Removing a
 * factor or replacing the recovery codes asks for the password. */
@Component({
  selector: 'bt-account-security',
  imports: [FormsModule, RouterLink, ButtonComponent, DialogComponent, IconComponent, NoticeComponent, TextFieldComponent, LocalDatePipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto flex max-w-2xl flex-col gap-6 p-4 sm:p-8">
      <a routerLink="/account" class="-my-2 inline-flex min-h-11 items-center self-start text-small text-ink-muted hover:text-forest">← {{ 'security.back' | t }}</a>
      <header class="flex flex-col gap-1.5">
        <h1 class="text-heading-2">{{ 'security.title' | t }}</h1>
        <p class="text-body text-ink-muted">{{ 'security.intro' | t }}</p>
      </header>

      @if (status(); as s) {
        @if (required() && !s.enabled) {
          <bt-notice tone="warning" icon="lock" [title]="'security.requiredTitle' | t">{{ 'security.requiredBody' | t }}</bt-notice>
        } @else if (required() && s.enabled && !s.sessionVerified) {
          <bt-notice tone="warning" icon="lock" [title]="'security.reloginTitle' | t">
            {{ 'security.reloginBody' | t }}
            <button type="button" class="ml-1 font-semibold underline underline-offset-2" (click)="signInAgain()">{{ 'security.reloginAction' | t }}</button>
          </bt-notice>
        } @else if (required() && s.sessionVerified) {
          <bt-notice tone="success" icon="check" [title]="'security.readyTitle' | t">
            <a routerLink="/admin" class="font-semibold underline underline-offset-2">{{ 'security.readyAction' | t }}</a>
          </bt-notice>
        }

        @if (codes(); as list) {
          <section class="flex flex-col gap-3 rounded-xl border-2 border-clay bg-surface-card p-5" aria-live="polite">
            <h2 class="text-heading-4">{{ 'security.codesTitle' | t }}</h2>
            <p class="text-small text-ink-muted">{{ 'security.codesBody' | t }}</p>
            <ul class="grid grid-cols-2 gap-2 font-mono text-body">
              @for (c of list; track c) {
                <li class="rounded-md bg-surface-sunken px-3 py-2 text-center tracking-wider">{{ c }}</li>
              }
            </ul>
            <div class="flex flex-wrap gap-2">
              <bt-button variant="outline" size="sm" (click)="copyCodes(list)">{{ 'security.codesCopy' | t }}</bt-button>
              <bt-button variant="ghost" size="sm" (click)="codes.set(null)">{{ 'security.codesDone' | t }}</bt-button>
            </div>
          </section>
        }

        <section class="flex flex-col gap-4 rounded-xl border border-line bg-surface-card p-5">
          <div class="flex items-start gap-3">
            <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-forest-soft text-forest"><bt-icon name="clock" [size]="20" /></span>
            <div class="flex min-w-0 flex-1 flex-col gap-0.5">
              <h2 class="text-heading-4">{{ 'security.totpTitle' | t }}</h2>
              <p class="text-small text-ink-muted">{{ 'security.totpBody' | t }}</p>
            </div>
            <span class="inline-flex h-6 shrink-0 items-center rounded-full px-2.5 text-caption font-semibold" [class]="s.totp ? 'bg-forest-soft text-forest' : 'bg-surface-sunken text-ink-muted'">
              {{ (s.totp ? 'security.on' : 'security.off') | t }}
            </span>
          </div>

          @if (s.totp) {
            <bt-button class="self-start" variant="outline" size="sm" (click)="ask({ kind: 'disable-totp' })">{{ 'security.totpDisable' | t }}</bt-button>
          } @else if (setup(); as t) {
            <div class="flex flex-col gap-4 border-t border-line pt-4 sm:flex-row">
              <img [src]="t.qr" [alt]="'security.qrAlt' | t" width="180" height="180" class="h-[180px] w-[180px] shrink-0 self-center rounded-lg border border-line bg-white p-2" />
              <div class="flex min-w-0 flex-1 flex-col gap-3">
                <p class="text-small">{{ 'security.totpStep1' | t }}</p>
                <p class="text-small text-ink-muted">
                  {{ 'security.totpManual' | t }}
                  <span class="mt-1 block rounded-md bg-surface-sunken px-3 py-2 font-mono text-small wrap-anywhere text-ink select-all">{{ t.secret }}</span>
                </p>
                <form class="flex flex-col gap-3" (ngSubmit)="confirm()">
                  <bt-text-field [label]="'security.totpStep2' | t" inputmode="numeric" autocomplete="one-time-code" [ngModel]="code()" (ngModelChange)="code.set($event)" name="code" />
                  <div class="flex flex-wrap gap-2">
                    <bt-button type="submit" variant="solid" [disabled]="busy() || code().trim().length < 6">{{ 'security.totpConfirm' | t }}</bt-button>
                    <bt-button type="button" variant="ghost" (click)="setup.set(null)">{{ 'common.cancel' | t }}</bt-button>
                  </div>
                </form>
              </div>
            </div>
          } @else {
            <bt-button class="self-start" variant="solid" [disabled]="busy()" (click)="startTotp()">{{ 'security.totpSetup' | t }}</bt-button>
          }
        </section>

        <section class="flex flex-col gap-4 rounded-xl border border-line bg-surface-card p-5">
          <div class="flex items-start gap-3">
            <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-sky-soft text-sky"><bt-icon name="shield" [size]="20" /></span>
            <div class="flex min-w-0 flex-1 flex-col gap-0.5">
              <h2 class="text-heading-4">{{ 'security.passkeysTitle' | t }}</h2>
              <p class="text-small text-ink-muted">{{ 'security.passkeysBody' | t }}</p>
            </div>
            <span class="inline-flex h-6 shrink-0 items-center rounded-full px-2.5 text-caption font-semibold" [class]="s.passkeys.length ? 'bg-forest-soft text-forest' : 'bg-surface-sunken text-ink-muted'">
              {{ s.passkeys.length ? ('security.passkeyCount' | t: { count: s.passkeys.length }) : ('security.off' | t) }}
            </span>
          </div>
          @if (s.passkeys.length) {
            <ul class="flex flex-col border-t border-line">
              @for (p of s.passkeys; track p.id) {
                <li class="flex min-h-14 items-center gap-3 border-b border-line py-2 last:border-b-0">
                  <span class="flex min-w-0 flex-1 flex-col">
                    <b class="truncate text-small">{{ p.name || ('security.passkeyUnnamed' | t) }}</b>
                    <span class="text-caption text-ink-muted">{{ 'security.passkeyAdded' | t: { date: (p.createdAt | localDate: 'dayMonthYear') } }}</span>
                  </span>
                  <bt-button variant="ghost" size="sm" (click)="ask({ kind: 'remove-passkey', passkey: p })">{{ 'security.passkeyRemove' | t }}</bt-button>
                </li>
              }
            </ul>
          }
          @if (s.passkeysAvailable === false) {
            <p class="text-small text-ink-muted">{{ 'security.passkeysUnavailable' | t }}</p>
          } @else if (!passkeys) {
            <p class="text-small text-ink-muted">{{ 'security.passkeysUnsupported' | t }}</p>
          } @else {
            <form class="flex flex-wrap items-end gap-2" (ngSubmit)="addPasskey()">
              <bt-text-field class="min-w-[200px] flex-1" [label]="'security.passkeyName' | t" [ngModel]="passkeyName()" (ngModelChange)="passkeyName.set($event)" name="passkeyName" />
              <bt-button type="submit" variant="outline" [disabled]="busy()">{{ 'security.passkeyAdd' | t }}</bt-button>
            </form>
          }
        </section>

        @if (s.enabled) {
          <section class="flex flex-wrap items-center gap-3 rounded-xl border border-line bg-surface-card p-5">
            <div class="flex min-w-0 flex-1 flex-col gap-0.5">
              <h2 class="text-heading-4">{{ 'security.recoveryTitle' | t }}</h2>
              <p class="text-small" [class]="s.recoveryCodes <= 2 ? 'font-semibold text-clay-ink' : 'text-ink-muted'">{{ 'security.recoveryLeft' | t: { count: s.recoveryCodes } }}</p>
            </div>
            <bt-button variant="outline" size="sm" (click)="ask({ kind: 'recovery-codes' })">{{ 'security.recoveryNew' | t }}</bt-button>
          </section>
        }
      } @else if (statusResource.error()) {
        <div class="flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="statusResource.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      } @else {
        <div class="h-48 animate-pulse rounded-xl bg-surface-sunken" aria-busy="true"></div>
      }
    </div>

    <bt-dialog
      [open]="guarded() !== null"
      [title]="'security.passwordTitle' | t"
      [primaryLabel]="'security.passwordConfirm' | t"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="runGuarded()"
      (secondaryClick)="guarded.set(null)"
      (closed)="guarded.set(null)"
    >
      <div class="flex flex-col gap-3">
        <p>{{ 'security.passwordBody.' + guardedKey() | t }}</p>
        <bt-text-field [label]="'auth.loginPasswordLabel' | t" type="password" autocomplete="current-password" [ngModel]="password()" (ngModelChange)="password.set($event)" />
      </div>
    </bt-dialog>
  `
})
export class AccountSecurityComponent {
  private readonly authApi = inject(AuthApi);
  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  /** The back office sent the member here: it needs a second factor. */
  protected readonly required = toSignal(this.route.queryParamMap.pipe(map((p) => p.get('required') === '1')), { initialValue: false });
  protected readonly passkeys = passkeysSupported();

  protected readonly statusResource = apiResource<MfaStatus>(() => this.authApi.mfaStatusUrl());
  /** What the API last said after a change, else the first read. */
  private readonly changed = signal<MfaStatus | null>(null);
  protected readonly status = computed(() => this.changed() ?? this.statusResource.value());

  protected readonly busy = signal(false);
  protected readonly setup = signal<TotpSetup | null>(null);
  protected readonly code = signal('');
  protected readonly passkeyName = signal('');
  /** Recovery codes just issued: shown once. */
  protected readonly codes = signal<readonly string[] | null>(null);
  protected readonly guarded = signal<Guarded | null>(null);
  protected readonly password = signal('');
  protected readonly guardedKey = computed(() => ({ 'disable-totp': 'disableTotp', 'remove-passkey': 'removePasskey', 'recovery-codes': 'recoveryCodes' })[this.guarded()?.kind ?? 'disable-totp']);

  private failed(err: AppError): void {
    this.busy.set(false);
    this.toast.show({ tone: 'danger', title: 'security.failed', detail: err.message });
  }

  /** A factor was added: the session now counts as verified. */
  private enrolled(result: Enrolled, toastKey: string): void {
    this.busy.set(false);
    this.auth.useAccessToken(result.accessToken);
    this.changed.set({ ...result.status, sessionVerified: true, passkeysAvailable: this.status()?.passkeysAvailable });
    if (result.recoveryCodes?.length) this.codes.set(result.recoveryCodes);
    this.toast.show({ tone: 'success', title: toastKey });
  }

  protected startTotp(): void {
    this.busy.set(true);
    this.authApi.setupTotp().subscribe({
      next: (t) => {
        this.busy.set(false);
        this.code.set('');
        this.setup.set(t);
      },
      error: (err: AppError) => this.failed(err)
    });
  }

  protected confirm(): void {
    if (this.busy()) return;
    this.busy.set(true);
    this.authApi.confirmTotp(this.code().trim()).subscribe({
      next: (result) => {
        this.setup.set(null);
        this.enrolled(result, 'security.totpEnabled');
      },
      error: (err: AppError) => this.failed(err)
    });
  }

  protected addPasskey(): void {
    if (this.busy()) return;
    this.busy.set(true);
    this.authApi.passkeyOptions().subscribe({
      next: ({ options, flowToken }) => {
        createPasskey(options.publicKey).then(
          (credential) =>
            this.authApi.addPasskey(flowToken ?? '', this.passkeyName().trim(), credential).subscribe({
              next: (result) => {
                this.passkeyName.set('');
                this.enrolled(result, 'security.passkeyAddedToast');
              },
              error: (err: AppError) => this.failed(err)
            }),
          () => {
            // The browser's prompt was closed, or this device already has one.
            this.busy.set(false);
            this.toast.show({ tone: 'danger', title: 'security.passkeyCancelled' });
          }
        );
      },
      error: (err: AppError) => this.failed(err)
    });
  }

  protected ask(action: Guarded): void {
    this.password.set('');
    this.guarded.set(action);
  }

  protected runGuarded(): void {
    const action = this.guarded();
    const password = this.password();
    if (!action || !password || this.busy()) return;
    this.busy.set(true);
    const done = (status: MfaStatus, toastKey: string): void => {
      this.busy.set(false);
      this.guarded.set(null);
      this.changed.set({ ...status, sessionVerified: this.status()?.sessionVerified, passkeysAvailable: this.status()?.passkeysAvailable });
      this.toast.show({ tone: 'success', title: toastKey });
    };
    switch (action.kind) {
      case 'disable-totp':
        this.authApi.disableTotp(password).subscribe({ next: (s) => done(s, 'security.totpDisabled'), error: (err: AppError) => this.failed(err) });
        break;
      case 'remove-passkey':
        this.authApi.removePasskey(action.passkey.id, password).subscribe({ next: (s) => done(s, 'security.passkeyRemoved'), error: (err: AppError) => this.failed(err) });
        break;
      case 'recovery-codes':
        this.authApi.regenerateRecoveryCodes(password).subscribe({
          next: ({ recoveryCodes }) => {
            this.busy.set(false);
            this.guarded.set(null);
            this.codes.set(recoveryCodes);
            const current = this.status();
            if (current) this.changed.set({ ...current, recoveryCodes: recoveryCodes.length });
          },
          error: (err: AppError) => this.failed(err)
        });
        break;
    }
  }

  protected copyCodes(list: readonly string[]): void {
    navigator.clipboard?.writeText(list.join('\n')).then(
      () => this.toast.show({ tone: 'success', title: 'security.codesCopied' }),
      () => undefined
    );
  }

  /** The session predates the factor: only a fresh sign-in passes it. */
  protected signInAgain(): void {
    this.auth.signOut();
    this.router.navigate(['/login'], { queryParams: { redirectTo: '/admin' } });
  }
}
