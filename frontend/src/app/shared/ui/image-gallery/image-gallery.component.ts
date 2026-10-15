import { ChangeDetectionStrategy, Component, DestroyRef, ElementRef, inject, input, signal, viewChild } from '@angular/core';
import { DOCUMENT } from '@angular/common';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { IconComponent } from '../icon/icon.component';

/** Swipeable photo strip: native horizontal scroll with snap points, so
 * it pans under the thumb with platform momentum and needs no gesture
 * library. Dots track the visible photo and jump to one on tap; from
 * `lg` up, arrows and the arrow keys step through them. Overlay controls
 * (back, share, favorite) go in the projected content, which sits over
 * the top of the strip.
 *
 * Every photo shows whole, at its own proportions (`object-contain`),
 * over a blurred copy of itself that fills the frame, so a portrait or
 * panoramic photo is never cropped. A tap opens it fullscreen. These are
 * the full-size `photos`, never the card thumbs. */
@Component({
  selector: 'bt-image-gallery',
  imports: [TranslatePipe, IconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(click)': 'onHostClick($event)' },
  styles: `
    /* No display here: it would beat the hidden / lg:inline-flex
       utilities (encapsulated selectors are more specific). */
    .nav-btn {
      height: 44px;
      width: 44px;
      align-items: center;
      justify-content: center;
      border-radius: 9999px;
      background: color-mix(in srgb, var(--color-surface-raised) 88%, transparent);
      color: var(--color-ink);
      box-shadow: 0 4px 14px rgba(22, 33, 28, 0.18);
    }
    .nav-btn:disabled {
      opacity: 0;
      pointer-events: none;
    }
    .viewer-img {
      touch-action: pan-x pinch-zoom;
    }
    dialog[open] {
      animation: bt-viewer-in 180ms ease-out;
    }
    @keyframes bt-viewer-in {
      from {
        opacity: 0;
      }
    }
  `,
  template: `
    <div class="group relative overflow-hidden bg-surface-sunken" [class]="frameClass()">
      @if (photos().length) {
        <div
          #strip
          (scroll)="onScroll()"
          (keydown)="onKey($event)"
          tabindex="0"
          class="no-scrollbar flex h-full snap-x snap-mandatory overflow-x-auto overscroll-x-contain focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-focus"
          [attr.aria-label]="'gallery.label' | t: { count: photos().length }"
          role="region"
        >
          @for (photo of photos(); track photo; let i = $index) {
            <button
              type="button"
              (click)="openViewer(i)"
              [attr.aria-label]="'gallery.open' | t: { n: i + 1 }"
              tabindex="-1"
              class="relative h-full w-full shrink-0 snap-center cursor-zoom-in overflow-hidden"
            >
              <img [src]="photo" alt="" aria-hidden="true" loading="lazy" decoding="async" class="absolute inset-0 h-full w-full scale-110 object-cover opacity-60 blur-2xl" />
              <img
                [src]="photo"
                [alt]="alt() + ' (' + (i + 1) + '/' + photos().length + ')'"
                [attr.loading]="i === 0 ? 'eager' : 'lazy'"
                [attr.fetchpriority]="i === 0 ? 'high' : null"
                decoding="async"
                class="relative h-full w-full object-contain"
                [class]="photoClass()"
              />
            </button>
          }
        </div>
        @if (photos().length > 1) {
          <button
            type="button"
            class="nav-btn absolute left-4 top-1/2 hidden -translate-y-1/2 opacity-0 transition group-hover:opacity-100 focus-visible:opacity-100 lg:inline-flex"
            [disabled]="index() === 0"
            (click)="goTo(index() - 1)"
            [attr.aria-label]="'gallery.prev' | t"
          >
            <bt-icon name="chevron-left" [size]="22" />
          </button>
          <button
            type="button"
            class="nav-btn absolute right-4 top-1/2 hidden -translate-y-1/2 opacity-0 transition group-hover:opacity-100 focus-visible:opacity-100 lg:inline-flex"
            [disabled]="index() === photos().length - 1"
            (click)="goTo(index() + 1)"
            [attr.aria-label]="'gallery.next' | t"
          >
            <bt-icon name="chevron-right" [size]="22" />
          </button>
          <div class="pointer-events-none absolute inset-x-0 bottom-9 flex justify-center gap-1.5 lg:bottom-4">
            @for (photo of photos(); track photo; let i = $index) {
              <button
                type="button"
                (click)="goTo(i)"
                [attr.aria-label]="'gallery.goTo' | t: { n: i + 1 }"
                [attr.aria-current]="i === index() ? 'true' : null"
                class="pointer-events-auto h-2 rounded-full shadow-sm transition-all"
                [class]="i === index() ? 'w-5 bg-white' : 'w-2 bg-white/60'"
              ></button>
            }
          </div>
        }
      } @else {
        <div class="flex h-full items-center justify-center text-small text-ink-muted">{{ 'gallery.noPhotos' | t }}</div>
      }
      <div class="pointer-events-none absolute inset-x-0 top-0 flex items-start justify-between gap-2 p-4 *:pointer-events-auto" style="padding-top: max(1rem, env(safe-area-inset-top))">
        <ng-content />
      </div>
    </div>

    <!-- Fullscreen: the native <dialog> gives the focus trap, Escape and
         top-layer stacking, as in BottomSheet. -->
    <dialog
      #viewer
      (close)="onViewerClose()"
      (keydown)="onViewerKey($event)"
      tabindex="-1"
      [attr.aria-label]="'gallery.label' | t: { count: photos().length }"
      class="m-0 h-dvh max-h-none w-dvw max-w-none bg-black p-0 text-white backdrop:bg-black"
    >
      @if (viewerOpen()) {
        <div
          #viewerStrip
          (scroll)="onViewerScroll()"
          class="no-scrollbar flex h-full snap-x snap-mandatory overflow-x-auto overscroll-contain"
        >
          @for (photo of photos(); track photo; let i = $index) {
            <div class="viewer-slide flex h-full w-full shrink-0 snap-center items-center justify-center">
              <img [src]="photo" [alt]="alt() + ' (' + (i + 1) + '/' + photos().length + ')'" decoding="async" class="viewer-img max-h-full max-w-full object-contain" />
            </div>
          }
        </div>
        <div
          class="absolute inset-x-0 top-0 flex items-center justify-between gap-3 bg-gradient-to-b from-black/60 to-transparent p-4"
          style="padding-top: max(1rem, env(safe-area-inset-top))"
        >
          <span class="font-mono text-small" aria-live="polite">{{ 'gallery.counter' | t: { n: viewerIndex() + 1, count: photos().length } }}</span>
          <button type="button" (click)="closeViewer()" [attr.aria-label]="'gallery.close' | t" class="nav-btn inline-flex">
            <bt-icon name="x" [size]="22" />
          </button>
        </div>
        @if (photos().length > 1) {
          <button
            type="button"
            class="nav-btn absolute left-4 top-1/2 hidden -translate-y-1/2 lg:inline-flex"
            [disabled]="viewerIndex() === 0"
            (click)="viewerGoTo(viewerIndex() - 1)"
            [attr.aria-label]="'gallery.prev' | t"
          >
            <bt-icon name="chevron-left" [size]="24" />
          </button>
          <button
            type="button"
            class="nav-btn absolute right-4 top-1/2 hidden -translate-y-1/2 lg:inline-flex"
            [disabled]="viewerIndex() === photos().length - 1"
            (click)="viewerGoTo(viewerIndex() + 1)"
            [attr.aria-label]="'gallery.next' | t"
          >
            <bt-icon name="chevron-right" [size]="24" />
          </button>
        }
      }
    </dialog>
  `
})
export class ImageGalleryComponent {
  private readonly document = inject(DOCUMENT);

  readonly photos = input.required<readonly string[]>();
  readonly alt = input.required<string>();
  /** Sizing for the frame (aspect ratio / height) — set by the page. */
  readonly frameClass = input('aspect-[4/3]');
  /** Extra classes for each photo, e.g. padding that keeps it clear of
   * content the page lays over the frame's edge. */
  readonly photoClass = input('');

  protected readonly index = signal(0);
  protected readonly viewerOpen = signal(false);
  protected readonly viewerIndex = signal(0);
  private readonly strip = viewChild<ElementRef<HTMLElement>>('strip');
  private readonly viewer = viewChild.required<ElementRef<HTMLDialogElement>>('viewer');
  private readonly viewerStrip = viewChild<ElementRef<HTMLElement>>('viewerStrip');

  constructor() {
    inject(DestroyRef).onDestroy(() => this.lockScroll(false));
  }

  protected onScroll(): void {
    const el = this.strip()?.nativeElement;
    if (el && el.clientWidth) {
      this.index.set(Math.round(el.scrollLeft / el.clientWidth));
    }
  }

  protected goTo(i: number, behavior: ScrollBehavior = 'smooth'): void {
    const el = this.strip()?.nativeElement;
    el?.scrollTo({ left: this.clamp(i) * el.clientWidth, behavior });
  }

  protected onKey(event: KeyboardEvent): void {
    const step = this.step(event);
    if (step) {
      event.preventDefault();
      this.goTo(this.index() + step);
    }
  }

  protected openViewer(i: number): void {
    this.viewerIndex.set(i);
    this.viewerOpen.set(true);
    this.viewer().nativeElement.showModal();
    this.lockScroll(true);
    // The strip renders with the dialog: jump to the tapped photo once
    // it has a width.
    requestAnimationFrame(() => this.viewerGoTo(i, 'instant'));
  }

  closeViewer(): void {
    this.viewer().nativeElement.close();
  }

  protected onViewerClose(): void {
    this.lockScroll(false);
    this.viewerOpen.set(false);
    // Back in the page on the photo last seen fullscreen.
    this.goTo(this.viewerIndex(), 'instant');
  }

  protected onViewerScroll(): void {
    const el = this.viewerStrip()?.nativeElement;
    if (el && el.clientWidth) {
      this.viewerIndex.set(Math.round(el.scrollLeft / el.clientWidth));
    }
  }

  protected viewerGoTo(i: number, behavior: ScrollBehavior = 'smooth'): void {
    const el = this.viewerStrip()?.nativeElement;
    el?.scrollTo({ left: this.clamp(i) * el.clientWidth, behavior });
    if (behavior === 'instant') this.viewerIndex.set(this.clamp(i));
  }

  protected onViewerKey(event: KeyboardEvent): void {
    const step = this.step(event);
    if (step) {
      event.preventDefault();
      this.viewerGoTo(this.viewerIndex() + step);
    }
  }

  /** A click on the dialog itself (around the photo) closes it. */
  protected onHostClick(event: MouseEvent): void {
    const dialog = this.viewer().nativeElement;
    const target = event.target as HTMLElement;
    if (dialog.open && (target === dialog || target.classList.contains('viewer-slide'))) {
      this.closeViewer();
    }
  }

  private step(event: KeyboardEvent): number {
    return event.key === 'ArrowRight' ? 1 : event.key === 'ArrowLeft' ? -1 : 0;
  }

  private clamp(i: number): number {
    return Math.max(0, Math.min(i, this.photos().length - 1));
  }

  private lockScroll(locked: boolean): void {
    this.document.documentElement.style.overflow = locked ? 'hidden' : '';
  }
}
