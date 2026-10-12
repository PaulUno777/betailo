import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { SellerSpaceService } from '../../../core/auth/seller-space.service';
import { ModeSwitchComponent, ModeSwitchMode } from './mode-switch.component';

describe('ModeSwitchComponent', () => {
  function render(hasFarm: boolean, mode: ModeSwitchMode): HTMLElement {
    TestBed.configureTestingModule({
      imports: [ModeSwitchComponent],
      providers: [provideRouter([]), { provide: SellerSpaceService, useValue: { hasFarm: signal(hasFarm) } }]
    });
    const fixture = TestBed.createComponent(ModeSwitchComponent);
    fixture.componentRef.setInput('mode', mode);
    fixture.detectChanges();
    return fixture.nativeElement;
  }

  it('marks the current side and links to the other one', () => {
    const el = render(true, 'buy');
    expect(el.querySelectorAll('[aria-current="page"]').length).toBe(1);
    const links = Array.from(el.querySelectorAll('a'));
    expect(links.map((a) => a.getAttribute('href'))).toEqual(['/portal']);
  });

  it('links back to the market from the seller side', () => {
    const el = render(true, 'sell');
    expect(Array.from(el.querySelectorAll('a')).map((a) => a.getAttribute('href'))).toEqual(['/']);
  });

  it('renders nothing for an account without a farm', () => {
    expect(render(false, 'buy').querySelector('[role="group"]')).toBeNull();
  });
});
