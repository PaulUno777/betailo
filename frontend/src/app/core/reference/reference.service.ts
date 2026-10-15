import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, catchError, map, of } from 'rxjs';
import { ApiClient } from '../http/api-client';
import { LocaleService } from '../i18n/locale.service';
import { Catalog, CatalogTerm, EMPTY_CATALOG, catalogLabels, refKey } from '../../domain/reference/catalog';
import { setCategories } from '../../domain/reference/species';
import { setCountries } from '../../domain/reference/countries';

const STORAGE_KEY = 'betailo.reference';

/** The reference data in force (`GET /reference`): loaded once before
 * the app first renders, its labels registered as translations and its
 * lists handed to the species and countries helpers. The last copy is
 * kept in the browser, so a failed load still shows the lists. */
@Injectable({ providedIn: 'root' })
export class ReferenceService {
  private readonly api = inject(ApiClient);
  private readonly locale = inject(LocaleService);

  readonly catalog = signal<Catalog>(EMPTY_CATALOG);

  load(): Observable<void> {
    return this.api.get<Catalog>('/reference').pipe(
      catchError(() => of(readStored())),
      map((catalog) => {
        if (catalog) this.use(catalog);
      })
    );
  }

  /** The forum categories a new topic can be filed under, as options. */
  readonly forumOptions = computed(() =>
    activeKeys(this.catalog().forumCategories).map((key) => ({ value: key, label: this.locale.t(refKey.forum(key)) }))
  );

  /** The "what you liked" chips a review can carry. */
  readonly reviewTags = computed(() => activeKeys(this.catalog().reviewTags));

  /** The label key of a stored forum category, or the text itself for a
   * topic filed before categories were a fixed list. */
  forumLabelKey(key: string): string {
    return this.catalog().forumCategories.some((t) => t.key === key) ? refKey.forum(key) : key;
  }

  reviewTagLabelKey(key: string): string {
    return refKey.reviewTag(key);
  }

  farmCheckLabelKey(key: string): string {
    return refKey.farmCheck(key);
  }

  private use(catalog: Catalog): void {
    this.catalog.set(catalog);
    setCategories(catalog.categories);
    setCountries(catalog.countries, catalog.defaultCountry);
    this.locale.setRuntimeLabels(catalogLabels(catalog));
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(catalog));
    } catch {
      /* storage unavailable: the next visit loads it again */
    }
  }
}

function readStored(): Catalog | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as Catalog) : null;
  } catch {
    return null;
  }
}

function activeKeys(list: readonly CatalogTerm[]): string[] {
  return list.filter((t) => t.active).map((t) => t.key);
}
