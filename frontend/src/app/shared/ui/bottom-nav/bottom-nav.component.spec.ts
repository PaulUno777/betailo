import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { BottomNavComponent, BottomNavItem } from './bottom-nav.component';

const ITEMS: readonly BottomNavItem[] = [
  { kind: 'link', labelKey: 'a', icon: 'home', route: '/a', exact: true },
  { kind: 'link', labelKey: 'b', icon: 'search', route: '/b' },
  { kind: 'link', labelKey: 'c', icon: 'calendar', route: '/c' },
  { kind: 'more', labelKey: 'd', icon: 'grid' }
];

describe('BottomNavComponent', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({ imports: [BottomNavComponent], providers: [provideRouter([])] });
  });

  function render(withCenter: boolean) {
    const fixture = TestBed.createComponent(BottomNavComponent);
    fixture.componentRef.setInput('items', ITEMS);
    if (withCenter) {
      fixture.componentRef.setInput('center', { labelKey: 'publish', icon: 'plus', route: '/sell' });
    }
    fixture.detectChanges();
    return fixture;
  }

  it('puts the raised center action between the two halves of the tabs', () => {
    const el: HTMLElement = render(true).nativeElement;
    const slots = Array.from(el.querySelectorAll('nav > div > *')).map(
      (n) => n.getAttribute('href') ?? (n.querySelector('a') ? 'center' : 'button')
    );
    expect(slots).toEqual(['/a', '/b', 'center', '/c', 'button']);
  });

  it('renders every tab in order when there is no center action', () => {
    const el: HTMLElement = render(false).nativeElement;
    expect(el.querySelectorAll('nav a').length).toBe(3);
    expect(el.querySelectorAll('nav button').length).toBe(1);
  });

  it('emits `more` from the overflow tab instead of navigating', () => {
    const fixture = render(false);
    let emitted = 0;
    fixture.componentInstance.more.subscribe(() => emitted++);
    (fixture.nativeElement as HTMLElement).querySelector<HTMLButtonElement>('nav button')!.click();
    expect(emitted).toBe(1);
  });

  it('shows a badge only on tabs with something waiting', () => {
    const fixture = TestBed.createComponent(BottomNavComponent);
    fixture.componentRef.setInput('items', [
      { kind: 'link', labelKey: 'a', icon: 'home', route: '/a', badge: 0 },
      { kind: 'link', labelKey: 'b', icon: 'calendar', route: '/b', badge: 12 }
    ] satisfies BottomNavItem[]);
    fixture.detectChanges();
    const tabs = (fixture.nativeElement as HTMLElement).querySelectorAll('nav a');
    expect(tabs[0].querySelector('.bg-clay')).toBeNull();
    expect(tabs[1].querySelector('.bg-clay')?.textContent?.trim()).toBe('9+');
  });
});
