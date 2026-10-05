import { TestBed } from '@angular/core/testing';
import { LocaleService } from './locale.service';

describe('LocaleService', () => {
  const originalLanguages = Object.getOwnPropertyDescriptor(window.navigator, 'languages');

  function setBrowserLanguages(languages: readonly string[]): void {
    Object.defineProperty(window.navigator, 'languages', { value: languages, configurable: true });
  }

  beforeEach(() => {
    TestBed.configureTestingModule({});
  });

  afterEach(() => {
    localStorage.removeItem('betailo.locale');
    if (originalLanguages) {
      Object.defineProperty(window.navigator, 'languages', originalLanguages);
    }
  });

  it('detects French from the browser language', () => {
    localStorage.removeItem('betailo.locale');
    setBrowserLanguages(['fr-FR', 'en-US']);
    const locale = TestBed.inject(LocaleService);
    expect(locale.locale()).toBe('fr');
  });

  it('defaults to English when the browser isn’t French', () => {
    localStorage.removeItem('betailo.locale');
    setBrowserLanguages(['de-DE', 'en-US']);
    const locale = TestBed.inject(LocaleService);
    expect(locale.locale()).toBe('en');
  });

  it('prefers a stored preference over browser detection', () => {
    localStorage.setItem('betailo.locale', 'fr');
    setBrowserLanguages(['en-US']);
    const locale = TestBed.inject(LocaleService);
    expect(locale.locale()).toBe('fr');
  });

  it('translates a known key and persists the choice on setLocale', () => {
    const locale = TestBed.inject(LocaleService);
    locale.setLocale('en');
    expect(locale.t('nav.marketplace')).toBe('Marketplace');
    locale.setLocale('fr');
    expect(locale.t('nav.marketplace')).toBe('Marché');
    expect(localStorage.getItem('betailo.locale')).toBe('fr');
  });

  it('interpolates {param} placeholders', () => {
    const locale = TestBed.inject(LocaleService);
    locale.setLocale('en');
    expect(locale.t('capacity.remaining', { count: 5 })).toBe('5 spots left');
  });

  it('passes an unknown key through unchanged, so a raw backend message displays as-is', () => {
    const locale = TestBed.inject(LocaleService);
    expect(locale.t('insufficient capacity')).toBe('insufficient capacity');
  });
});
