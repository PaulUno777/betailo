import { Component } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { LocaleService } from './locale.service';
import { TranslatePipe } from './translate.pipe';

@Component({
  selector: 'bt-test-host',
  imports: [TranslatePipe],
  template: `{{ 'nav.marketplace' | t }}`
})
class TestHostComponent {}

describe('TranslatePipe', () => {
  let fixture: ComponentFixture<TestHostComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({ imports: [TestHostComponent] });
    fixture = TestBed.createComponent(TestHostComponent);
    fixture.detectChanges();
  });

  it('renders the translation for the current locale', () => {
    TestBed.inject(LocaleService).setLocale('en');
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent.trim()).toBe('Marketplace');
  });

  it('re-renders when the locale changes later — regression test for the pure-pipe bug where switching locale silently failed to update already-rendered text', () => {
    const locale = TestBed.inject(LocaleService);
    locale.setLocale('en');
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent.trim()).toBe('Marketplace');

    locale.setLocale('fr');
    TestBed.flushEffects();
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent.trim()).toBe('Marché');
  });
});
