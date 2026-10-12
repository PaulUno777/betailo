import { TestBed } from '@angular/core/testing';
import { ThemeService } from './theme.service';

describe('ThemeService', () => {
  beforeEach(() => {
    localStorage.removeItem('betailo.theme');
    document.documentElement.removeAttribute('data-theme');
    TestBed.configureTestingModule({});
  });

  afterEach(() => {
    localStorage.removeItem('betailo.theme');
    document.documentElement.removeAttribute('data-theme');
  });

  it('defaults to system, with no [data-theme] attribute set', () => {
    const theme = TestBed.inject(ThemeService);
    expect(theme.preference()).toBe('system');
    expect(document.documentElement.hasAttribute('data-theme')).toBeFalse();
  });

  it('setPreference("dark") sets [data-theme="dark"] and persists it', () => {
    const theme = TestBed.inject(ThemeService);
    theme.setPreference('dark');
    TestBed.flushEffects();
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
    expect(theme.isDark()).toBeTrue();
    expect(localStorage.getItem('betailo.theme')).toBe('dark');
  });

  it('setPreference("light") sets [data-theme="light"]', () => {
    const theme = TestBed.inject(ThemeService);
    theme.setPreference('light');
    TestBed.flushEffects();
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
    expect(theme.isDark()).toBeFalse();
  });

  it('restores a persisted preference on a fresh instance', () => {
    TestBed.inject(ThemeService).setPreference('dark');
    TestBed.resetTestingModule();
    TestBed.configureTestingModule({});
    const restored = TestBed.inject(ThemeService);
    expect(restored.preference()).toBe('dark');
  });

  it('toggle flips between light and dark', () => {
    const theme = TestBed.inject(ThemeService);
    theme.setPreference('light');
    theme.toggle();
    expect(theme.preference()).toBe('dark');
    theme.toggle();
    expect(theme.preference()).toBe('light');
  });
});
