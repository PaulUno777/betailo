import { DestroyRef, Directive, ElementRef, afterNextRender, inject, input, output } from '@angular/core';

/** Emits `reached` when its element comes within `margin` of the
 * viewport: put it on an empty sentinel after a list to load the next
 * page before the reader hits the end. Emits again each time it comes
 * back into reach, so the parent decides whether a load is due (more to
 * load, none in flight, no error).
 *
 *   <div btInfiniteScroll (reached)="loadMore()"></div> */
@Directive({ selector: '[btInfiniteScroll]' })
export class InfiniteScrollDirective {
  /** How far ahead of the viewport to start loading. */
  readonly margin = input('600px');
  readonly reached = output<void>();

  constructor() {
    const el = inject<ElementRef<HTMLElement>>(ElementRef).nativeElement;
    const destroyRef = inject(DestroyRef);
    afterNextRender(() => {
      if (typeof IntersectionObserver === 'undefined') return;
      const observer = new IntersectionObserver(
        (entries) => {
          if (entries.some((e) => e.isIntersecting)) this.reached.emit();
        },
        { rootMargin: `${this.margin()} 0px` }
      );
      observer.observe(el);
      destroyRef.onDestroy(() => observer.disconnect());
    });
  }
}
