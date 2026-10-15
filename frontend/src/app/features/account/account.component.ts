import { ChangeDetectionStrategy, Component, computed, effect, inject, signal, untracked } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { switchMap } from 'rxjs';
import { Router, RouterLink } from '@angular/router';
import { Profile } from '../../domain/models';
import { AccountApi } from '../../data-access/account/account.api';
import { AuthService } from '../../core/auth/auth.service';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { AppError } from '../../core/http/app-error';
import { PhotoUploadService } from '../../core/http/photo-upload.service';
import { ToastService } from '../../core/notifications/toast.service';
import { ThemeService } from '../../core/theme/theme.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { AvatarComponent } from '../../shared/ui/avatar/avatar.component';
import { BottomSheetComponent } from '../../shared/ui/bottom-sheet/bottom-sheet.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { IconName } from '../../shared/ui/icon/icon-paths';
import { LocaleToggleComponent } from '../../shared/ui/locale-toggle/locale-toggle.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';

interface AccountLink {
  readonly route: string;
  readonly labelKey: string;
  readonly icon: IconName;
}

type Sheet = 'profile' | 'password' | null;

/** `/account` — "Compte", the last tab on phones: who you are, the one
 * big card that moves you between buying and selling (or makes you a
 * seller), your lists, and the settings that don't fit in a phone
 * header (language, theme, password, sign-out). Everything is one tap
 * deep; edits open in a sheet. */
@Component({
  selector: 'bt-account',
  imports: [
    FormsModule,
    RouterLink,
    AvatarComponent,
    BottomSheetComponent,
    ButtonComponent,
    IconComponent,
    LocaleToggleComponent,
    TextFieldComponent,
    LocalDatePipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto flex max-w-2xl flex-col gap-6 p-4 sm:p-8">
      <section class="flex items-center gap-4">
        <!-- The profile photo: the camera badge picks a new one. -->
        <label class="relative shrink-0 cursor-pointer" [class.opacity-60]="photoBusy()" [title]="'account.photoHint' | t">
          <bt-avatar [name]="name()" [src]="profile.value()?.avatarUrl" [size]="64" />
          <span class="absolute -bottom-1 -right-1 flex h-7 w-7 items-center justify-center rounded-full border-2 border-surface bg-forest text-on-forest">
            <bt-icon name="camera" [size]="14" />
          </span>
          <input
            type="file"
            accept="image/*"
            class="sr-only"
            [disabled]="photoBusy()"
            [attr.aria-label]="(profile.value()?.avatarUrl ? 'account.photoChange' : 'account.photoAdd') | t"
            (change)="pickPhoto($event)"
          />
        </label>
        <div class="flex min-w-0 flex-col">
          <h1 class="truncate text-heading-2">{{ name() }}</h1>
          @if (profile.value(); as p) {
            <span class="text-small text-ink-muted">
              {{ 'account.memberSince' | t: { date: (p.memberSince | localDate: 'monthYear') } }}
            </span>
            @if (photoBusy()) {
              <span class="text-small text-ink-muted">{{ 'account.photoUploading' | t }}</span>
            } @else if (p.avatarUrl) {
              <button type="button" class="self-start text-small text-ink-muted underline underline-offset-2" (click)="removePhoto()">
                {{ 'account.photoRemove' | t }}
              </button>
            }
          }
        </div>
      </section>

      <a
        [routerLink]="modeCard().route"
        class="flex items-center gap-4 rounded-xl bg-forest-deep p-5 text-on-forest-deep transition active:scale-[0.99]"
      >
        <span class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-on-forest-deep/10">
          <bt-icon [name]="modeCard().icon" [size]="24" class="text-clay" />
        </span>
        <span class="flex min-w-0 flex-1 flex-col">
          <span class="text-heading-4">{{ modeCard().titleKey | t }}</span>
          <span class="text-small text-on-forest-deep/80">{{ modeCard().bodyKey | t }}</span>
        </span>
        <bt-icon name="chevron-right" [size]="22" />
      </a>

      <nav class="flex flex-col overflow-hidden rounded-lg border border-line bg-surface-card">
        @for (link of links(); track link.route) {
          <a [routerLink]="link.route" [class]="rowClass">
            <bt-icon [name]="link.icon" [size]="22" class="text-forest" />
            <span class="flex-1">{{ link.labelKey | t }}</span>
            <bt-icon name="chevron-right" [size]="18" class="text-ink-muted" />
          </a>
        }
      </nav>

      <section class="flex flex-col overflow-hidden rounded-lg border border-line bg-surface-card">
        <h2 class="px-4 pb-1 pt-4 text-eyebrow text-ink-muted">{{ 'account.settingsTitle' | t }}</h2>
        <button type="button" (click)="openProfile()" [class]="rowClass">
          <bt-icon name="user" [size]="22" class="text-ink-muted" />
          <span class="flex flex-1 flex-col text-left">
            <span>{{ 'account.profile' | t }}</span>
            @if (profile.value(); as p) {
              <span class="text-small text-ink-muted">{{ contactLine(p) || ('account.profileEmpty' | t) }}</span>
            }
          </span>
          <bt-icon name="chevron-right" [size]="18" class="text-ink-muted" />
        </button>
        <button type="button" (click)="openPassword()" [class]="rowClass">
          <bt-icon name="lock" [size]="22" class="text-ink-muted" />
          <span class="flex-1 text-left">{{ 'account.password' | t }}</span>
          <bt-icon name="chevron-right" [size]="18" class="text-ink-muted" />
        </button>
        <a routerLink="/account/security" [class]="rowClass">
          <bt-icon name="shield" [size]="22" class="text-ink-muted" />
          <span class="flex-1 text-left">{{ 'account.security' | t }}</span>
          <bt-icon name="chevron-right" [size]="18" class="text-ink-muted" />
        </a>
        <div [class]="rowClass">
          <span class="flex-1">{{ 'moreSheet.language' | t }}</span>
          <bt-locale-toggle />
        </div>
        <button type="button" (click)="theme.toggle()" [class]="rowClass">
          <bt-icon [name]="theme.isDark() ? 'sun' : 'moon'" [size]="22" class="text-ink-muted" />
          <span class="flex-1 text-left">{{ (theme.isDark() ? 'moreSheet.lightMode' : 'moreSheet.darkMode') | t }}</span>
        </button>
      </section>

      <bt-button variant="outline" size="lg" block (click)="signOut()">
        <bt-icon name="logout" [size]="20" />
        {{ 'nav.signOut' | t }}
      </bt-button>
    </div>

    <bt-bottom-sheet [open]="sheet() === 'profile'" [title]="'account.profile' | t" (closed)="sheet.set(null)">
      <form class="flex flex-col gap-4" (submit)="$event.preventDefault(); saveProfile()">
        <bt-text-field
          [label]="'account.displayName' | t"
          [hint]="'account.displayNameHint' | t"
          autocomplete="name"
          name="displayName"
          [ngModel]="displayName()"
          (ngModelChange)="displayName.set($event)"
        />
        <bt-text-field
          [label]="'account.phone' | t"
          type="tel"
          inputmode="tel"
          autocomplete="tel"
          name="phone"
          [ngModel]="phone()"
          (ngModelChange)="phone.set($event)"
        />
        <bt-text-field [label]="'account.city' | t" autocomplete="address-level2" name="city" [ngModel]="city()" (ngModelChange)="city.set($event)" />
        <bt-button type="submit" size="lg" block [disabled]="busy()">{{ 'account.save' | t }}</bt-button>
      </form>
    </bt-bottom-sheet>

    <bt-bottom-sheet [open]="sheet() === 'password'" [title]="'account.password' | t" (closed)="sheet.set(null)">
      <form class="flex flex-col gap-4" (submit)="$event.preventDefault(); savePassword()">
        <bt-text-field
          [label]="'account.currentPassword' | t"
          type="password"
          autocomplete="current-password"
          name="current"
          [ngModel]="currentPassword()"
          (ngModelChange)="currentPassword.set($event)"
        />
        <bt-text-field
          [label]="'account.newPassword' | t"
          [hint]="'account.newPasswordHint' | t"
          type="password"
          autocomplete="new-password"
          name="new"
          [ngModel]="newPassword()"
          (ngModelChange)="newPassword.set($event)"
        />
        <bt-button type="submit" size="lg" block [disabled]="busy() || !currentPassword() || !newPassword()">{{ 'account.save' | t }}</bt-button>
      </form>
    </bt-bottom-sheet>
  `
})
export class AccountComponent {
  protected readonly auth = inject(AuthService);
  protected readonly theme = inject(ThemeService);
  private readonly accountApi = inject(AccountApi);
  private readonly uploads = inject(PhotoUploadService);
  private readonly toast = inject(ToastService);
  private readonly router = inject(Router);
  private readonly seller = inject(SellerSpaceService);

  protected readonly profile = apiResource<Profile>(() => this.accountApi.profileUrl());

  protected readonly name = computed(() => {
    const p = this.profile.value();
    const u = this.auth.user();
    return p?.displayName || u?.displayName || p?.email || u?.email || '';
  });

  /** One card, three cases: someone with a farm switches to their
   * seller space, staff go to the back office, anyone else is invited
   * to open a farm. */
  protected readonly modeCard = computed(() => {
    if (this.auth.isStaff()) {
      return { route: '/admin', icon: 'shield' as IconName, titleKey: 'account.mode.adminTitle', bodyKey: 'account.mode.adminBody' };
    }
    if (this.seller.hasFarm()) {
      return { route: '/portal', icon: 'leaf' as IconName, titleKey: 'account.mode.sellerTitle', bodyKey: 'account.mode.sellerBody' };
    }
    return { route: '/sell', icon: 'leaf' as IconName, titleKey: 'account.mode.becomeTitle', bodyKey: 'account.mode.becomeBody' };
  });

  protected readonly links = computed<readonly AccountLink[]>(() => {
    const base: AccountLink[] = [
      { route: '/account/reservations', labelKey: 'nav.myReservations', icon: 'calendar' },
      { route: '/account/favorites', labelKey: 'nav.favorites', icon: 'heart' },
      { route: '/notifications', labelKey: 'nav.notifications', icon: 'bell' }
    ];
    if (this.seller.hasFarm()) {
      base.push({ route: '/portal/farms', labelKey: 'account.myFarms', icon: 'farm' });
    }
    return base;
  });

  protected readonly rowClass =
    'flex min-h-14 w-full items-center gap-3 border-t border-line px-4 text-body text-ink transition first:border-t-0 hover:bg-surface-sunken active:bg-surface-sunken';

  protected readonly sheet = signal<Sheet>(null);
  protected readonly photoBusy = signal(false);
  protected readonly busy = signal(false);
  protected readonly displayName = signal('');
  protected readonly phone = signal('');
  protected readonly city = signal('');
  protected readonly currentPassword = signal('');
  protected readonly newPassword = signal('');

  constructor() {
    // Keep the cached display name in step with the account, so the
    // header shows it even when it was set on another device.
    effect(() => {
      const p = this.profile.value();
      if (p?.displayName && p.displayName !== untracked(() => this.auth.user()?.displayName)) {
        untracked(() => this.auth.updateUser({ displayName: p.displayName }));
      }
    });
  }

  /** Resizes the picked photo, uploads it with its small copy, then
   * makes it the profile photo. */
  protected pickPhoto(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file || this.photoBusy()) return;
    this.photoBusy.set(true);
    this.uploads
      .upload(file, (sizes) => this.accountApi.presignAvatar(sizes), { thumb: true })
      .pipe(switchMap((key) => this.accountApi.setAvatar(key)))
      .subscribe({
        next: (p) => this.photoSaved(p, 'account.photoSaved'),
        error: (err: AppError) => this.photoFailed(err)
      });
  }

  protected removePhoto(): void {
    if (this.photoBusy()) return;
    this.photoBusy.set(true);
    this.accountApi.removeAvatar().subscribe({
      next: (p) => this.photoSaved(p, 'account.photoRemoved'),
      error: (err: AppError) => this.photoFailed(err)
    });
  }

  private photoSaved(p: Profile, title: string): void {
    this.photoBusy.set(false);
    this.auth.updateUser({ avatarUrl: p.avatarUrl });
    this.profile.reload();
    this.toast.show({ tone: 'success', title });
  }

  private photoFailed(err: AppError): void {
    this.photoBusy.set(false);
    this.toast.show({ tone: 'danger', title: 'account.photoFailed', detail: err.message });
  }

  protected contactLine(p: Profile): string {
    return [p.phone, p.city].filter(Boolean).join(' · ');
  }

  protected openProfile(): void {
    const p = this.profile.value();
    this.displayName.set(p?.displayName ?? '');
    this.phone.set(p?.phone ?? '');
    this.city.set(p?.city ?? '');
    this.sheet.set('profile');
  }

  protected openPassword(): void {
    this.currentPassword.set('');
    this.newPassword.set('');
    this.sheet.set('password');
  }

  protected saveProfile(): void {
    if (this.busy()) return;
    this.busy.set(true);
    this.accountApi.updateProfile({ displayName: this.displayName().trim(), phone: this.phone().trim(), city: this.city().trim() }).subscribe({
      next: (p) => {
        this.busy.set(false);
        this.sheet.set(null);
        this.auth.updateUser({ displayName: p.displayName });
        this.profile.reload();
        this.toast.show({ tone: 'success', title: 'account.savedToast' });
      },
      error: (err: AppError) => {
        this.busy.set(false);
        this.toast.show({ tone: 'danger', title: 'account.saveFailed', detail: err.message });
      }
    });
  }

  /** Every other device is signed out; this one gets a fresh session. */
  protected savePassword(): void {
    if (this.busy()) return;
    this.busy.set(true);
    this.accountApi.changePassword({ currentPassword: this.currentPassword(), newPassword: this.newPassword(), remember: true }).subscribe({
      next: (res) => {
        this.busy.set(false);
        this.sheet.set(null);
        this.auth.setSession(res.accessToken, res.user);
        this.toast.show({ tone: 'success', title: 'account.passwordChanged' });
      },
      error: (err: AppError) => {
        this.busy.set(false);
        this.toast.show({ tone: 'danger', title: 'account.saveFailed', detail: err.message });
      }
    });
  }

  protected signOut(): void {
    this.auth.signOut();
    this.router.navigateByUrl('/');
  }
}
