import { Component, inject, Injector, runInInjectionContext } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { mediaQuery } from './media-query';

describe('mediaQuery', () => {
  let listeners: ((e: MediaQueryListEvent) => void)[];
  let removed: number;

  beforeEach(() => {
    listeners = [];
    removed = 0;
    spyOn(window, 'matchMedia').and.returnValue({
      matches: true,
      addEventListener: (_: string, l: (e: MediaQueryListEvent) => void) => listeners.push(l),
      removeEventListener: () => removed++
    } as unknown as MediaQueryList);
  });

  it('starts from the current match and follows changes', () => {
    const matches = runInInjectionContext(TestBed.inject(Injector), () => mediaQuery('(min-width: 640px)'));
    expect(window.matchMedia).toHaveBeenCalledWith('(min-width: 640px)');
    expect(matches()).toBeTrue();
    listeners.forEach((l) => l({ matches: false } as MediaQueryListEvent));
    expect(matches()).toBeFalse();
  });

  it('stops listening when its owner is destroyed', () => {
    @Component({ template: '' })
    class HostComponent {
      readonly wide = mediaQuery('(min-width: 640px)');
      readonly injector = inject(Injector);
    }
    const fixture = TestBed.createComponent(HostComponent);
    fixture.destroy();
    expect(removed).toBe(1);
  });
});
