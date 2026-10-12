import { ChangeDetectionStrategy, Component, DestroyRef, computed, inject, input, model } from '@angular/core';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { IconComponent } from '../icon/icon.component';

/** One photo in the picker: either already stored (`key`, shown from its
 * signed URL) or freshly picked on this device (`file`, shown from an
 * object URL until it's uploaded). */
export interface PickedPhoto {
  readonly id: string;
  readonly preview: string;
  readonly file?: File;
  readonly key?: string;
}

let nextId = 0;

/** "Montrez vos animaux", per the mobile design: a big camera button
 * (the rear camera opens directly on phones), thumbnails with the first
 * one marked as the cover, and a dashed "Galerie" tile. Tapping a
 * thumbnail makes it the cover. Picked files are only previewed here;
 * the page uploads them when it saves. */
@Component({
  selector: 'bt-photo-picker',
  imports: [IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-3">
      @if (photos().length < max()) {
        <label
          class="flex h-16 cursor-pointer items-center justify-center gap-3 rounded-lg bg-forest text-body-strong text-on-forest transition active:scale-[0.99] has-[:focus-visible]:shadow-[var(--shadow-focus)]"
        >
          <bt-icon name="camera" [size]="22" />
          {{ 'photoPicker.takePhoto' | t }}
          <input type="file" accept="image/*" capture="environment" class="sr-only" (change)="add($event)" />
        </label>
      }
      <div class="grid grid-cols-3 gap-2" role="list">
        @for (p of photos(); track p.id; let i = $index) {
          <div class="relative aspect-square" role="listitem">
            <button
              type="button"
              class="h-full w-full overflow-hidden rounded-lg"
              (click)="makeCover(i)"
              [attr.aria-label]="(i === 0 ? 'photoPicker.isCover' : 'photoPicker.makeCover') | t"
            >
              <img [src]="p.preview" alt="" class="h-full w-full object-cover" />
            </button>
            @if (i === 0) {
              <span class="pointer-events-none absolute bottom-2 left-2 rounded-full bg-ink/85 px-2.5 py-1 text-caption font-semibold text-surface">
                {{ 'photoPicker.cover' | t }}
              </span>
            }
            <button
              type="button"
              (click)="remove(i)"
              [attr.aria-label]="'photoPicker.remove' | t"
              class="absolute right-1 top-1 flex h-9 w-9 items-center justify-center rounded-full bg-ink/70 text-surface"
            >
              <bt-icon name="x" [size]="16" />
            </button>
          </div>
        }
        @if (photos().length < max()) {
          <label
            class="flex aspect-square cursor-pointer flex-col items-center justify-center gap-1 rounded-lg border-2 border-dashed border-line-strong text-ink-muted transition hover:border-forest has-[:focus-visible]:shadow-[var(--shadow-focus)]"
          >
            <bt-icon name="plus" [size]="24" />
            <span class="text-small">{{ 'photoPicker.gallery' | t }}</span>
            <input type="file" accept="image/*" multiple class="sr-only" (change)="add($event)" />
          </label>
        }
      </div>
      <p class="text-small text-ink-muted">
        {{ 'photoPicker.tip' | t }}
        @if (remaining() <= 3) {
          · {{ 'photoPicker.remaining' | t: { count: remaining() } }}
        }
      </p>
    </div>
  `
})
export class PhotoPickerComponent {
  readonly photos = model.required<PickedPhoto[]>();
  readonly max = input(12);

  protected readonly remaining = computed(() => Math.max(0, this.max() - this.photos().length));

  constructor() {
    inject(DestroyRef).onDestroy(() => this.photos().forEach(revokeIfLocal));
  }

  protected add(event: Event): void {
    const input = event.target as HTMLInputElement;
    const picked = Array.from(input.files ?? [])
      .filter((f) => f.type.startsWith('image/'))
      .slice(0, this.remaining())
      .map((file) => ({ id: `p${++nextId}`, file, preview: URL.createObjectURL(file) }));
    this.photos.update((list) => [...list, ...picked]);
    input.value = '';
  }

  protected remove(i: number): void {
    revokeIfLocal(this.photos()[i]);
    this.photos.update((list) => list.filter((_, idx) => idx !== i));
  }

  protected makeCover(i: number): void {
    if (i === 0) return;
    this.photos.update((list) => [list[i], ...list.filter((_, idx) => idx !== i)]);
  }
}

/** Wraps already-stored photos (signed URL + key) for the picker. */
export function storedPhotos(urls: readonly string[], keys: readonly string[]): PickedPhoto[] {
  return keys.map((key, i) => ({ id: `k-${key}`, key, preview: urls[i] ?? '' }));
}

function revokeIfLocal(p: PickedPhoto | undefined): void {
  if (p?.file) URL.revokeObjectURL(p.preview);
}
