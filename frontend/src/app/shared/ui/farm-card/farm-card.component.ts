import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { FarmProfile } from '../../../domain/models';
import { telUrl, whatsappUrl } from '../../../domain/rules/contact';
import { AvatarComponent } from '../avatar/avatar.component';
import { IconComponent } from '../icon/icon.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { PlacePipe } from '../../../core/i18n/place.pipe';
import { LocalDatePipe } from '../../pipes/local-date.pipe';

/** The seller as a buyer sizes them up, compact enough to sit beside a
 * listing: the farm's logo (no banner, that's for its own page),
 * name, place, completed sales, how long they've been here, and the
 * server-computed trust score. With contact numbers (only sent to
 * signed-in viewers) it adds a call button for each, and a WhatsApp
 * one for those on WhatsApp; otherwise, for a signed-out viewer, a
 * prompt to sign in to contact the farmer. */
@Component({
  selector: 'bt-farm-card',
  imports: [RouterLink, AvatarComponent, IconComponent, TranslatePipe, PlacePipe, LocalDatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-3 rounded-lg border border-line bg-surface-card p-4">
      <div class="flex items-center gap-3">
        <bt-avatar [name]="farm().name" [src]="farm().logoThumbUrl" tone="forest" [size]="52" />
        <div class="flex min-w-0 flex-1 flex-col">
          <span class="flex items-center gap-1.5 text-body-strong">
            <span class="truncate">{{ farm().name }}</span>
            @if (farm().verified) {
              <bt-icon name="shield" [size]="16" class="shrink-0 text-forest" [label]="'farmCard.verified' | t" />
            }
          </span>
          <span class="text-small text-ink-muted">
            {{ farm().location | place }} · {{ 'farmCard.sales' | t: { count: farm().salesCount } }} · {{ 'farmCard.since' | t: { year: (farm().memberSince | localDate: 'year') } }}
          </span>
          @if (farm().ownerName) {
            <span class="flex items-center gap-1.5 pt-0.5 text-small text-ink-muted">
              <bt-avatar [name]="farm().ownerName || farm().name" [src]="farm().ownerAvatarUrl" [size]="20" tone="clay" />
              <span class="truncate">{{ 'farmCard.farmer' | t: { name: farm().ownerName || '' } }}</span>
            </span>
          }
        </div>
        <div class="flex shrink-0 flex-col items-center rounded-md bg-forest-soft px-3 py-1.5 text-forest">
          <span class="text-heading-3 leading-none">{{ trust() }}</span>
          <span class="text-[10px] font-semibold uppercase tracking-wide">{{ 'farmCard.trust' | t }}</span>
        </div>
      </div>
      @if (farm().contacts.length) {
        <!-- One line per number: the number, then how to reach it. -->
        <ul class="flex flex-col gap-2.5 border-t border-line pt-3">
          @for (c of farm().contacts; track c.phone) {
            <li class="flex items-center gap-2">
              <span class="flex min-w-0 flex-1 flex-col">
                <span class="truncate font-mono text-small text-ink">{{ c.phone }}</span>
                <span class="text-caption text-ink-muted">{{ (c.whatsapp ? 'farmCard.whatsappAndCalls' : 'farmCard.callOnly') | t }}</span>
              </span>
              @if (c.whatsapp) {
                <a
                  [href]="whatsapp(c.phone)"
                  target="_blank"
                  rel="noopener"
                  class="flex h-10 items-center gap-1.5 rounded-full bg-forest px-3.5 text-small font-semibold text-on-forest"
                >
                  <bt-icon name="message" [size]="16" />
                  {{ 'farmCard.whatsapp' | t }}
                </a>
              }
              <a
                [href]="tel(c.phone)"
                class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-line-strong text-ink"
                [attr.aria-label]="('farmCard.call' | t) + ' ' + c.phone"
                [title]="'farmCard.call' | t"
              >
                <bt-icon name="phone" [size]="18" />
              </a>
            </li>
          }
        </ul>
      } @else if (!signedIn()) {
        <a routerLink="/login" [queryParams]="{ redirectTo: returnUrl() }" class="text-small text-forest underline">
          {{ 'farmCard.signInToContact' | t }}
        </a>
      }
    </div>
  `
})
export class FarmCardComponent {
  readonly farm = input.required<FarmProfile>();
  readonly signedIn = input(false);
  readonly returnUrl = input('/');
  /** Already translated (the caller owns the wording and the locale). */
  readonly whatsappMessage = input('');

  protected readonly trust = computed(() => Math.round(this.farm().trustScore));
  protected whatsapp(phone: string): string {
    return whatsappUrl(phone, this.whatsappMessage());
  }

  protected tel(phone: string): string {
    return telUrl(phone);
  }
}
