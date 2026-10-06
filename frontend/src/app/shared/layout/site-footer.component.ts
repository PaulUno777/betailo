import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { LogoComponent } from '../ui/logo/logo.component';

/** The site-wide sitemap footer, per `Home.dc.html`'s footer board — one
 * shell-level component rather than duplicated per public route, since
 * nothing in the design ties it to Home specifically (the mockup's fixed
 * canvas height just doesn't have room to show it on every board). */
@Component({
  selector: 'bt-site-footer',
  imports: [RouterLink, LogoComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <footer class="bg-forest-deep text-on-forest-deep">
      <!-- Phones get a one-line footer: a sitemap is desktop furniture; on
           mobile the bottom nav already is the sitemap. -->
      <div class="flex items-center justify-between gap-4 px-5 py-6 sm:hidden">
        <bt-logo tone="light" />
        <p class="text-caption text-on-forest-deep/70">{{ 'footer.tagline' | t }}</p>
      </div>
      <div class="mx-auto hidden max-w-content-max gap-10 px-5 py-14 sm:grid sm:grid-cols-12 sm:px-18">
        <div class="flex flex-col gap-3 sm:col-span-5">
          <bt-logo tone="light" />
          <p class="max-w-xs text-small text-on-forest-deep/70">{{ 'footer.tagline' | t }}</p>
        </div>
        <div class="flex flex-col gap-2.5 sm:col-span-2">
          <span class="text-small font-medium">{{ 'footer.marketplaceHeading' | t }}</span>
          <a routerLink="/marketplace" class="text-small text-on-forest-deep/70 hover:text-on-forest-deep">{{ 'footer.allCycles' | t }}</a>
          <a routerLink="/marketplace" [queryParams]="{ species: 'jeunes-poulets' }" class="text-small text-on-forest-deep/70 hover:text-on-forest-deep">{{ 'footer.poultry' | t }}</a>
          <a routerLink="/marketplace" [queryParams]="{ species: 'porcs' }" class="text-small text-on-forest-deep/70 hover:text-on-forest-deep">{{ 'footer.pigs' | t }}</a>
        </div>
        <div class="flex flex-col gap-2.5 sm:col-span-2">
          <span class="text-small font-medium">{{ 'footer.farmersHeading' | t }}</span>
          <a routerLink="/sell" class="text-small text-on-forest-deep/70 hover:text-on-forest-deep">{{ 'footer.publishCycle' | t }}</a>
          <a routerLink="/" fragment="eleveurs" class="text-small text-on-forest-deep/70 hover:text-on-forest-deep">{{ 'nav.farmers' | t }}</a>
          <span class="text-small text-on-forest-deep/70">{{ 'footer.pricing' | t }}</span>
          <span class="text-small text-on-forest-deep/70">{{ 'footer.healthGuide' | t }}</span>
        </div>
        <div class="flex flex-col gap-2.5 sm:col-span-3">
          <span class="text-small font-medium">{{ 'footer.trustHeading' | t }}</span>
          <a routerLink="/" fragment="how" class="text-small text-on-forest-deep/70 hover:text-on-forest-deep">{{ 'nav.howItWorks' | t }}</a>
          <span class="text-small text-on-forest-deep/70">{{ 'footer.howEscrowWorks' | t }}</span>
          <span class="text-small text-on-forest-deep/70">{{ 'footer.reportScam' | t }}</span>
          <span class="text-small text-on-forest-deep/70">{{ 'footer.terms' | t }}</span>
        </div>
      </div>
    </footer>
  `
})
export class SiteFooterComponent {}
