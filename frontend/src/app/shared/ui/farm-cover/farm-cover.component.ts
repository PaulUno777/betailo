import { ChangeDetectionStrategy, Component, booleanAttribute, input, output } from '@angular/core';
import { FarmImage } from '../../../data-access/farms/farms.api';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { AvatarComponent } from '../avatar/avatar.component';
import { IconComponent } from '../icon/icon.component';

/** The default banner: the brand's animal filigree, transparent, drawn
 * over a soft tint so it reads in both themes. */
export const DEFAULT_FARM_BANNER = '/images/farm-banner-default.webp';

/** A farm's face, laid out like a profile: a 3:1 banner (1500×500 is
 * the size to upload; phones re-encode to at most 1600px wide) with the
 * round logo overlapping its bottom edge. Without a banner it shows the
 * filigree, without a logo the farm's initials. `editable` adds camera
 * badges, as on the account photo, that pick a new image, plus links to
 * remove it; the parent uploads. */
@Component({
  selector: 'bt-farm-cover',
  imports: [AvatarComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col">
      <div class="relative aspect-3/1 w-full overflow-hidden rounded-lg bg-forest-soft" [class.opacity-60]="busy() === 'banner'">
        <img
          [src]="bannerUrl() || defaultBanner"
          alt=""
          class="h-full w-full object-cover"
          [class.opacity-80]="!bannerUrl()"
          loading="lazy"
        />
        @if (editable()) {
          <label
            class="absolute bottom-2 right-2 flex h-9 cursor-pointer items-center gap-1.5 rounded-full bg-surface-card/90 px-3 text-small font-semibold text-ink shadow-sm"
            [title]="'farmCover.bannerHint' | t"
          >
            <bt-icon name="camera" [size]="16" />
            {{ (bannerUrl() ? 'farmCover.bannerChange' : 'farmCover.bannerAdd') | t }}
            <input type="file" accept="image/*" class="sr-only" [disabled]="!!busy()" (change)="pick('banner', $event)" />
          </label>
        }
      </div>

      <div class="-mt-10 flex items-end gap-3 px-4">
        <div class="relative shrink-0 rounded-full border-4 border-surface bg-surface" [class.opacity-60]="busy() === 'logo'">
          <bt-avatar [name]="name() || '?'" [src]="logoUrl()" tone="forest" [size]="size()" />
          @if (editable()) {
            <!-- The whole logo picks a new one; the badge says so. -->
            <label class="absolute inset-0 cursor-pointer rounded-full" [title]="'farmCover.logoHint' | t">
              <span class="absolute -bottom-1 -right-1 flex h-7 w-7 items-center justify-center rounded-full border-2 border-surface bg-forest text-on-forest">
                <bt-icon name="camera" [size]="14" />
              </span>
              <input
                type="file"
                accept="image/*"
                class="sr-only"
                [disabled]="!!busy()"
                [attr.aria-label]="(logoUrl() ? 'farmCover.logoChange' : 'farmCover.logoAdd') | t"
                (change)="pick('logo', $event)"
              />
            </label>
          }
        </div>
        <div class="flex min-w-0 flex-1 flex-col pb-1">
          <ng-content />
          @if (editable()) {
            <span class="flex flex-wrap gap-x-3 text-small text-ink-muted">
              @if (busy()) {
                <span>{{ 'farmCover.uploading' | t }}</span>
              } @else {
                @if (logoUrl()) {
                  <button type="button" class="underline underline-offset-2" (click)="removed.emit('logo')">{{ 'farmCover.logoRemove' | t }}</button>
                }
                @if (bannerUrl()) {
                  <button type="button" class="underline underline-offset-2" (click)="removed.emit('banner')">{{ 'farmCover.bannerRemove' | t }}</button>
                }
              }
            </span>
          }
        </div>
      </div>
      @if (editable()) {
        <p class="mt-2 px-1 text-caption text-ink-muted">{{ 'farmCover.sizesHint' | t }}</p>
      }
    </div>
  `
})
export class FarmCoverComponent {
  readonly name = input('');
  readonly logoUrl = input<string>();
  readonly bannerUrl = input<string>();
  /** Logo diameter, in px. */
  readonly size = input(80);
  readonly editable = input(false, { transform: booleanAttribute });
  /** The image being uploaded, if any: the pickers wait for it. */
  readonly busy = input<FarmImage | null>(null);

  readonly picked = output<{ image: FarmImage; file: File }>();
  readonly removed = output<FarmImage>();

  protected readonly defaultBanner = DEFAULT_FARM_BANNER;

  protected pick(image: FarmImage, event: Event): void {
    const el = event.target as HTMLInputElement;
    const file = el.files?.[0];
    el.value = '';
    if (file) this.picked.emit({ image, file });
  }
}
