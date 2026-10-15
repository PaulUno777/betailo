import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { InfiniteScrollDirective } from './infinite-scroll.directive';

@Component({
  imports: [InfiniteScrollDirective],
  template: `<div btInfiniteScroll margin="300px" (reached)="count = count + 1"></div>`
})
class HostComponent {
  count = 0;
}

describe('InfiniteScrollDirective', () => {
  let callback: IntersectionObserverCallback;
  let options: IntersectionObserverInit | undefined;
  let disconnected: boolean;
  const original = window.IntersectionObserver;

  beforeEach(() => {
    disconnected = false;
    window.IntersectionObserver = class {
      constructor(cb: IntersectionObserverCallback, opts?: IntersectionObserverInit) {
        callback = cb;
        options = opts;
      }
      observe(): void {
        // The test fires the callback itself.
      }
      disconnect(): void {
        disconnected = true;
      }
    } as unknown as typeof IntersectionObserver;
  });

  afterEach(() => (window.IntersectionObserver = original));

  it('emits when the sentinel comes into reach, not when it leaves', async () => {
    const fixture = TestBed.createComponent(HostComponent);
    await fixture.whenStable();
    expect(options?.rootMargin).toBe('300px 0px');
    const fire = (isIntersecting: boolean) => callback([{ isIntersecting } as IntersectionObserverEntry], {} as IntersectionObserver);
    fire(true);
    fire(false);
    fire(true);
    expect(fixture.componentInstance.count).toBe(2);
    fixture.destroy();
    expect(disconnected).toBeTrue();
  });
});
