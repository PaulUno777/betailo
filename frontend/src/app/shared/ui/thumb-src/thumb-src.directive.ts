import { Directive, computed, input, signal } from '@angular/core';

/** Shows a photo's small copy instead of the photo itself. With `sizes`,
 * both go in a srcset and the browser picks by displayed width, so a
 * wide card on a sharp screen still gets the full photo. A photo from
 * before small copies existed has a thumb URL that 404s: the image then
 * falls back to the photo alone.
 *
 *   <img [btThumbSrc]="listing.thumbs?.[0]" [photo]="listing.photos[0]" alt="" /> */
@Directive({
  selector: 'img[btThumbSrc]',
  host: {
    '[src]': 'src()',
    '[attr.srcset]': 'srcset()',
    '[attr.sizes]': 'srcset() ? sizes() : null',
    '(error)': 'onError()'
  }
})
export class ThumbSrcDirective {
  readonly btThumbSrc = input<string | undefined>();
  readonly photo = input.required<string>();
  /** The `sizes` attribute; without it the thumb is always used. */
  readonly sizes = input<string>();

  /** The thumb URL that failed to load, if any. */
  private readonly failed = signal<string | null>(null);
  private readonly thumb = computed(() => {
    const t = this.btThumbSrc();
    return t && t !== this.failed() ? t : null;
  });

  protected readonly src = computed(() => (this.thumb() && !this.sizes() ? this.thumb()! : this.photo()));
  /** 640w/1600w match the widths PhotoUploadService encodes. */
  protected readonly srcset = computed(() => (this.thumb() && this.sizes() ? `${this.thumb()} 640w, ${this.photo()} 1600w` : null));

  protected onError(): void {
    const t = this.thumb();
    if (t) this.failed.set(t);
  }
}
