import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, catchError, forkJoin, map, of } from 'rxjs';
import { ApiClient } from '../http/api-client';
import { LocaleService } from '../i18n/locale.service';
import { CATALOG_PARTS, Catalog, CatalogPart, CatalogPartResponse, CatalogTerm, EMPTY_CATALOG, assembleCatalog, catalogLabels, refKey } from '../../domain/reference/catalog';
import { setCategories } from '../../domain/reference/species';
import { setCountries } from '../../domain/reference/countries';

const STORAGE_KEY = 'betailo.reference';

/** The reference data in force, one reference at a time
 * (`GET /reference/{name}`: animals, places and each vocabulary), loaded
 * in parallel before the app first renders; its labels are registered
 * as translations and its lists handed to the species and countries
 * helpers. Each reference's last copy is kept in the browser on its
 * own, so a failed load still shows it. */
@Injectable({ providedIn: 'root' })
export class ReferenceService {
  private readonly api = inject(ApiClient);
  private readonly locale = inject(LocaleService);

  readonly catalog = signal<Catalog>(EMPTY_CATALOG);

  load(): Observable<void> {
    return forkJoin(
      CATALOG_PARTS.map((name) =>
        this.api.get<CatalogPartResponse>(`/reference/${name}`).pipe(
          map((part) => {
            store(name, part);
            return part;
          }),
          catchError(() => of(readStored(name)))
        )
      )
    ).pipe(
      map((parts) => {
        if (parts.some(Boolean)) this.use(assembleCatalog(parts));
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

  /** A version just published from the back office: the whole app
   * switches to it without reloading. */
  apply(catalog: Catalog): void {
    this.use(catalog);
  }

  private use(catalog: Catalog): void {
    this.catalog.set(catalog);
    setCategories(catalog.categories);
    setCountries(catalog.countries, catalog.defaultCountry);
    this.locale.setRuntimeLabels(catalogLabels(catalog));
  }
}

function store(name: CatalogPart, part: CatalogPartResponse): void {
  try {
    localStorage.setItem(`${STORAGE_KEY}.${name}`, JSON.stringify(part));
  } catch {
    /* storage unavailable: the next visit loads it again */
  }
}

function readStored(name: CatalogPart): CatalogPartResponse | null {
  try {
    const raw = localStorage.getItem(`${STORAGE_KEY}.${name}`);
    return raw ? (JSON.parse(raw) as CatalogPartResponse) : null;
  } catch {
    return null;
  }
}

function activeKeys(list: readonly CatalogTerm[]): string[] {
  return list.filter((t) => t.active).map((t) => t.key);
}
