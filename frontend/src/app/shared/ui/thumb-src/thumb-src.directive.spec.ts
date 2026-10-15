import { Component, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { ThumbSrcDirective } from './thumb-src.directive';

@Component({
  imports: [ThumbSrcDirective],
  template: `<img [btThumbSrc]="thumb()" photo="https://cdn.test/a.jpg" [sizes]="sizes()" alt="" />`
})
class HostComponent {
  readonly thumb = signal<string | undefined>('https://cdn.test/a.thumb.jpg');
  readonly sizes = signal<string | undefined>(undefined);
}

describe('ThumbSrcDirective', () => {
  function render(setup?: (host: HostComponent) => void): HTMLImageElement {
    const fixture = TestBed.createComponent(HostComponent);
    setup?.(fixture.componentInstance);
    fixture.detectChanges();
    return fixture.nativeElement.querySelector('img');
  }

  it('shows the small copy', () => {
    const img = render();
    expect(img.getAttribute('src')).toBe('https://cdn.test/a.thumb.jpg');
    expect(img.hasAttribute('srcset')).toBeFalse();
  });

  it('offers both through srcset when given sizes', () => {
    const img = render((h) => h.sizes.set('50vw'));
    expect(img.getAttribute('srcset')).toBe('https://cdn.test/a.thumb.jpg 640w, https://cdn.test/a.jpg 1600w');
    expect(img.getAttribute('sizes')).toBe('50vw');
  });

  it('falls back to the photo when there is no small copy', () => {
    const fixture = TestBed.createComponent(HostComponent);
    fixture.detectChanges();
    const img: HTMLImageElement = fixture.nativeElement.querySelector('img');

    img.dispatchEvent(new Event('error'));
    fixture.detectChanges();

    expect(img.getAttribute('src')).toBe('https://cdn.test/a.jpg');
  });
});
