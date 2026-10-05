import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { LogoComponent } from '../../shared/ui/logo/logo.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { SegmentedControlComponent } from '../../shared/ui/segmented-control/segmented-control.component';
import { TranslatePipe } from '../../core/i18n/translate.pipe';

/** The split layout Auth.dc.html shows for both Login and Register — kept
 * as one shell so the brand panel and the Connexion/Inscription tabs
 * aren't duplicated across the two routed pages. */
@Component({
  selector: 'bt-auth-layout',
  imports: [RouterLink, RouterLinkActive, LogoComponent, IconComponent, SegmentedControlComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto grid min-h-[calc(100vh-4rem)] max-w-content-max grid-cols-1 lg:grid-cols-2">
      <div class="hidden flex-col justify-between bg-forest-deep p-12 text-on-forest-deep lg:flex">
        <bt-logo tone="light" />
        <div class="flex flex-col gap-6">
          <h1 class="text-display-l">{{ 'auth.brandHeadline' | t }}</h1>
          <ul class="flex flex-col gap-3 text-body-l">
            <li class="flex items-center gap-3"><bt-icon name="shield" [size]="20" /> {{ 'auth.brandBulletVerified' | t }}</li>
            <li class="flex items-center gap-3"><bt-icon name="lock" [size]="20" /> {{ 'auth.brandBulletEscrow' | t }}</li>
            <li class="flex items-center gap-3"><bt-icon name="star" [size]="20" /> {{ 'auth.brandBulletReviews' | t }}</li>
          </ul>
        </div>
        <p class="text-eyebrow opacity-70">{{ 'auth.brandFootnote' | t }}</p>
      </div>
      <div class="flex flex-col justify-center px-5 py-12 sm:px-18">
        <div class="mx-auto w-full max-w-sm">
          <bt-segmented-control class="mb-8">
            <a
              routerLink="/login"
              routerLinkActive
              #loginTab="routerLinkActive"
              [class]="
                'flex-1 rounded-full py-2.5 text-center text-body-strong transition ' +
                (loginTab.isActive ? 'bg-surface-card text-ink' : 'text-ink-muted hover:text-ink')
              "
              >{{ 'auth.connexionTab' | t }}</a
            >
            <a
              routerLink="/register"
              routerLinkActive
              #registerTab="routerLinkActive"
              [class]="
                'flex-1 rounded-full py-2.5 text-center text-body-strong transition ' +
                (registerTab.isActive ? 'bg-surface-card text-ink' : 'text-ink-muted hover:text-ink')
              "
              >{{ 'auth.inscriptionTab' | t }}</a
            >
          </bt-segmented-control>
          <ng-content />
        </div>
      </div>
    </div>
  `
})
export class AuthLayoutComponent {}
