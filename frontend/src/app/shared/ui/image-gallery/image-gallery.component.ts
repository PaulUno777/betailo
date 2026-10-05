import { ChangeDetectionStrategy, Component, ElementRef, input, signal, viewChild } from '@angular/core';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

/** Swipeable photo strip: native horizontal scroll with snap points, so
 * it pans under the thumb with platform momentum and needs no gesture
 * library. Dots track the visible photo and jump to one on tap. Overlay
 * controls (back, share, favorite) go in the projected content, which
 * sits over the top of the strip. */
@Component({
  selector: 'bt-image-gallery',
  imports: [TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="relative overflow-hidden bg-surface-sunken" [class]="frameClass()">
      @if (photos().length) {
        <div
          #strip
          (scroll)="onScroll()"
          class="no-scrollbar flex h-full snap-x snap-mandatory overflow-x-auto overscroll-x-contain"
          [attr.aria-label]="'gallery.label' | t: { count: photos().length }"
          role="region"
        >
          @for (photo of photos(); track photo; let i = $index) {
            <img
              [src]="photo"
              [alt]="alt() + ' (' + (i + 1) + '/' + photos().length + ')'"
              [attr.loading]="i === 0 ? 'eager' : 'lazy'"
              [attr.fetchpriority]="i === 0 ? 'high' : null"
              decoding="async"
              class="h-full w-full shrink-0 snap-center object-cover"
            />
          }
        </div>
        @if (photos().length > 1) {
          <div class="pointer-events-none absolute inset-x-0 bottom-9 flex justify-center gap-1.5">
            @for (photo of photos(); track photo; let i = $index) {
              <button
                type="button"
                (click)="goTo(i)"
                [attr.aria-label]="'gallery.goTo' | t: { n: i + 1 }"
                [attr.aria-current]="i === index() ? 'true' : null"
                class="pointer-events-auto h-2 rounded-full transition-all"
                [class]="i === index() ? 'w-5 bg-white' : 'w-2 bg-white/60'"
              ></button>
            }
          </div>
        }
      } @else {
        <div class="flex h-full items-center justify-center text-small text-ink-muted">{{ 'gallery.noPhotos' | t }}</div>
      }
      <div class="absolute inset-x-0 top-0 flex items-start justify-between gap-2 p-4" style="padding-top: max(1rem, env(safe-area-inset-top))">
        <ng-content />
      </div>
    </div>
  `
})
export class ImageGalleryComponent {
  readonly photos = input.required<readonly string[]>();
  readonly alt = input.required<string>();
  /** Sizing for the frame (aspect ratio / height) — set by the page. */
  readonly frameClass = input('aspect-[4/3]');

  protected readonly index = signal(0);
  private readonly strip = viewChild<ElementRef<HTMLElement>>('strip');

  protected onScroll(): void {
    const el = this.strip()?.nativeElement;
    if (el && el.clientWidth) {
      this.index.set(Math.round(el.scrollLeft / el.clientWidth));
    }
  }

  protected goTo(i: number): void {
    const el = this.strip()?.nativeElement;
    el?.scrollTo({ left: i * el.clientWidth, behavior: 'smooth' });
  }
}
