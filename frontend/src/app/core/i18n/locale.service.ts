import { Injectable, computed, effect, signal } from '@angular/core';
import { en } from './translations/en';
import { fr } from './translations/fr';
import { TranslationDictionary } from './translations/fr';

export type Locale = 'en' | 'fr';

const STORAGE_KEY = 'betailo.locale';
const DICTIONARIES: Readonly<Record<Locale, TranslationDictionary>> = { en, fr };

function detectInitialLocale(): Locale {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'en' || stored === 'fr') return stored;
  } catch {
    /* localStorage unavailable (private mode, etc.) — fall through to detection */
  }
  const browserLanguages = typeof navigator !== 'undefined' ? navigator.languages ?? [navigator.language] : [];
  const isFrench = browserLanguages.some((lang) => lang?.toLowerCase().startsWith('fr'));
  return isFrench ? 'fr' : 'en';
}

function readPath(dictionary: TranslationDictionary, path: string): string | undefined {
  const value = path.split('.').reduce<unknown>((node, segment) => {
    if (node && typeof node === 'object' && segment in node) {
      return (node as Record<string, unknown>)[segment];
    }
    return undefined;
  }, dictionary);
  return typeof value === 'string' ? value : undefined;
}

function interpolate(template: string, params?: Record<string, string | number>): string {
  if (!params) return template;
  return Object.entries(params).reduce((text, [key, value]) => text.replaceAll(`{${key}}`, String(value)), template);
}

/** English is the default whenever the browser locale isn't French —
 * French only wins when actually detected (or explicitly chosen), never
 * assumed. `domain/` stays locale-agnostic: rules return translation
 * keys (+ optional params), this service is the only thing that turns a
 * key into displayed text. */
@Injectable({ providedIn: 'root' })
export class LocaleService {
  readonly locale = signal<Locale>(detectInitialLocale());

  private readonly dictionary = computed(() => DICTIONARIES[this.locale()]);

  constructor() {
    // Screen readers pick their voice from <html lang>.
    effect(() => {
      document.documentElement.lang = this.locale();
    });
  }

  setLocale(locale: Locale): void {
    this.locale.set(locale);
    try {
      localStorage.setItem(STORAGE_KEY, locale);
    } catch {
      /* ignore */
    }
  }

  /** Reads `dictionary()` internally, so any signal-tracked read of this
   * (directly, or via the `translate` pipe) re-renders on locale change. */
  t(key: string, params?: Record<string, string | number>): string {
    const template = this.pluralVariant(key, params) ?? readPath(this.dictionary(), key) ?? key;
    return interpolate(template, params);
  }

  /** With a numeric `count` param, a singular count uses the key's
   * `_one` variant when the dictionary has one ("1 vente" vs "3 ventes").
   * Intl.PluralRules decides what counts as singular per language: in
   * French 0 and 1 both do, in English only 1. */
  private pluralVariant(key: string, params?: Record<string, string | number>): string | undefined {
    const count = params?.['count'];
    if (typeof count !== 'number') return undefined;
    const category = new Intl.PluralRules(this.locale()).select(count);
    return category === 'one' ? readPath(this.dictionary(), `${key}_one`) : undefined;
  }
}
