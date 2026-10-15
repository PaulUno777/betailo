import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { FarmProfile } from '../../../domain/models';
import { telUrl, whatsappUrl } from '../../../domain/rules/contact';
import { AvatarComponent } from '../avatar/avatar.component';
import { IconComponent } from '../icon/icon.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { LocalDatePipe } from '../../pipes/local-date.pipe';

/** The seller as a buyer sizes them up: name, place, completed sales,
 * how long they've been here, and the server-computed trust score.
 * With a contact phone (only sent to signed-in viewers) it adds
 * WhatsApp and call buttons; otherwise, for a signed-out viewer, a
 * prompt to sign in to contact the farmer. */
@Component({
  selector: 'bt-farm-card',
  imports: [RouterLink, AvatarComponent, IconComponent, TranslatePipe, LocalDatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-4 rounded-lg border border-line bg-surface-card p-4">
      <div class="flex items-center gap-3">
        <bt-avatar [name]="farm().name" shape="square" tone="forest" [size]="52" />
        <div class="flex min-w-0 flex-1 flex-col">
          <span class="flex items-center gap-1.5 text-body-strong">
            <span class="truncate">{{ farm().name }}</span>
            @if (farm().verified) {
              <bt-icon name="shield" [size]="16" class="shrink-0 text-forest" [label]="'farmCard.verified' | t" />
            }
          </span>
          <span class="text-small text-ink-muted">
            {{ farm().location }} · {{ 'farmCard.sales' | t: { count: farm().salesCount } }} · {{ 'farmCard.since' | t: { year: (farm().memberSince | localDate: 'year') } }}
          </span>
        </div>
        <div class="flex shrink-0 flex-col items-center rounded-md bg-forest-soft px-3 py-1.5 text-forest">
          <span class="text-heading-3 leading-none">{{ trust() }}</span>
          <span class="text-[10px] font-semibold uppercase tracking-wide">{{ 'farmCard.trust' | t }}</span>
        </div>
      </div>
      @if (farm().ownerName || farm().ownerAvatarUrl) {
        <div class="flex items-center gap-2.5 border-t border-line pt-3">
          <bt-avatar [name]="farm().ownerName || farm().name" [src]="farm().ownerAvatarUrl" [size]="28" tone="clay" />
          <span class="text-small text-ink-muted">{{ 'farmCard.farmer' | t: { name: farm().ownerName || '' } }}</span>
        </div>
      }
      @if (farm().contactPhone; as phone) {
        <div class="grid grid-cols-2 gap-2">
          <a
            [href]="whatsapp()"
            target="_blank"
            rel="noopener"
            class="flex h-control-md items-center justify-center gap-2 rounded-full bg-forest text-body-strong text-on-forest"
          >
            <bt-icon name="message" [size]="18" />
            {{ 'farmCard.whatsapp' | t }}
          </a>
          <a [href]="tel()" class="flex h-control-md items-center justify-center gap-2 rounded-full border border-line-strong text-body-strong">
            {{ 'farmCard.call' | t }}
          </a>
        </div>
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
  protected readonly whatsapp = computed(() => whatsappUrl(this.farm().contactPhone ?? '', this.whatsappMessage()));
  protected readonly tel = computed(() => telUrl(this.farm().contactPhone ?? ''));
}
