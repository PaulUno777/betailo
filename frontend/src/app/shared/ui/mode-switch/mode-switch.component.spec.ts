import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';
import { Role } from '../../../domain/enums';
import { ModeSwitchComponent, ModeSwitchMode } from './mode-switch.component';

describe('ModeSwitchComponent', () => {
  function render(role: Role | null, mode: ModeSwitchMode): HTMLElement {
    TestBed.configureTestingModule({
      imports: [ModeSwitchComponent],
      providers: [provideRouter([]), { provide: AuthService, useValue: { role: signal(role) } }]
    });
    const fixture = TestBed.createComponent(ModeSwitchComponent);
    fixture.componentRef.setInput('mode', mode);
    fixture.detectChanges();
    return fixture.nativeElement;
  }

  it('marks the current side and links to the other one', () => {
    const el = render('farmer', 'buy');
    expect(el.querySelectorAll('[aria-current="page"]').length).toBe(1);
    const links = Array.from(el.querySelectorAll('a'));
    expect(links.map((a) => a.getAttribute('href'))).toEqual(['/portal']);
  });

  it('links back to the market from the seller side', () => {
    const el = render('farmer', 'sell');
    expect(Array.from(el.querySelectorAll('a')).map((a) => a.getAttribute('href'))).toEqual(['/']);
  });

  it('renders nothing for buyers and admins', () => {
    expect(render('buyer', 'buy').querySelector('[role="group"]')).toBeNull();
    TestBed.resetTestingModule();
    expect(render('admin', 'buy').querySelector('[role="group"]')).toBeNull();
  });
});
