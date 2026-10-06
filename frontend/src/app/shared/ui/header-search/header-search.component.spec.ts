import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { Component } from '@angular/core';
import { HeaderSearchComponent } from './header-search.component';

@Component({ template: '' })
class BlankComponent {}

describe('HeaderSearchComponent', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HeaderSearchComponent],
      providers: [provideRouter([{ path: '**', component: BlankComponent }])]
    });
  });

  it('submits the trimmed query to the marketplace', async () => {
    const router = TestBed.inject(Router);
    const fixture = TestBed.createComponent(HeaderSearchComponent);
    fixture.detectChanges();
    const el: HTMLElement = fixture.nativeElement;
    const input = el.querySelector('input')!;
    input.value = '  poussins Bafoussam ';
    input.dispatchEvent(new Event('input'));
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    await fixture.whenStable();
    expect(router.url).toBe('/marketplace?q=poussins%20Bafoussam');
  });

  it('shows the marketplace query and clears it on other pages', async () => {
    const router = TestBed.inject(Router);
    const fixture = TestBed.createComponent(HeaderSearchComponent);
    fixture.detectChanges();
    const input = (fixture.nativeElement as HTMLElement).querySelector('input')!;

    await router.navigateByUrl('/marketplace?q=chevres');
    fixture.detectChanges();
    expect(input.value).toBe('chevres');

    await router.navigateByUrl('/forum');
    fixture.detectChanges();
    expect(input.value).toBe('');
  });
});
