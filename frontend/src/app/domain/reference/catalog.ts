/** The platform's reference data as `GET /reference` serves it: the
 * versioned catalog the super admin maintains. The API validates
 * against the same version, so the choices offered here are exactly
 * the ones it accepts. */
export interface Label {
  readonly fr: string;
  readonly en: string;
}

export interface CatalogTerm {
  readonly key: string;
  readonly label: Label;
  readonly active: boolean;
}

/** One of the brand's animal drawings, named by the API (see
 * shared/ui/animal-icon): the reference data decides which one a
 * category or sub-category wears. */
export type AnimalIconKey = string;

export interface CatalogSubcategory {
  readonly key: string;
  readonly label: Label;
  readonly cycleType: string;
  readonly icon: AnimalIconKey;
  readonly unit: { readonly key: string; readonly label: Label };
  readonly active: boolean;
}

export interface CatalogCategory {
  readonly key: string;
  readonly label: Label;
  readonly icon: AnimalIconKey;
  /** A design-system tone: clay, forest, amber, sky, alert, plum, stone. */
  readonly color: string;
  readonly active: boolean;
  readonly subcategories: readonly CatalogSubcategory[];
}

/** A town farms can pick. Its key is what a farm stores; a town the
 * catalog doesn't list is stored as typed. */
export interface CatalogCity {
  readonly key: string;
  readonly label: Label;
  readonly active: boolean;
}

export interface CatalogRegion {
  readonly key: string;
  readonly label: Label;
  readonly active: boolean;
  readonly cities?: readonly CatalogCity[];
}

export interface CatalogCountry {
  readonly code: string;
  readonly callingCode: string;
  readonly active: boolean;
  readonly regions?: readonly CatalogRegion[];
}

/** The references the reference data is stored as, each with its own
 * schema, versions and endpoint (`GET /reference/{name}`). */
export const CATALOG_PARTS = ['animals', 'places', 'forumCategories', 'reviewTags', 'reportReasons', 'farmReviewChecks'] as const;
export type CatalogPart = (typeof CATALOG_PARTS)[number];

export interface Catalog {
  /** The version in force of each reference ("animals": 4). */
  readonly versions: Readonly<Partial<Record<CatalogPart, number>>>;
  readonly defaultCountry: string;
  readonly cycleTypes: readonly string[];
  readonly categories: readonly CatalogCategory[];
  readonly countries: readonly CatalogCountry[];
  readonly forumCategories: readonly CatalogTerm[];
  readonly reviewTags: readonly CatalogTerm[];
  readonly reportReasons: readonly CatalogTerm[];
  readonly farmReviewChecks: readonly CatalogTerm[];
}

/** The lists a super admin edits and publishes: the catalog without
 * its version data. */
export type CatalogLists = Pick<Catalog, 'categories' | 'countries' | 'forumCategories' | 'reviewTags' | 'reportReasons' | 'farmReviewChecks'>;

/** An empty catalog: what the app holds until `GET /reference` answers. */
export const EMPTY_CATALOG: Catalog = {
  versions: {},
  defaultCountry: 'CM',
  cycleTypes: [],
  categories: [],
  countries: [],
  forumCategories: [],
  reviewTags: [],
  reportReasons: [],
  farmReviewChecks: []
};

/** Translation keys the catalog's labels are registered under. */
export const refKey = {
  category: (key: string) => `ref.category.${key}`,
  species: (key: string) => `ref.species.${key}`,
  unit: (key: string) => `ref.unit.${key}`,
  forum: (key: string) => `ref.forum.${key}`,
  reviewTag: (key: string) => `ref.reviewTag.${key}`,
  reportReason: (key: string) => `ref.reportReason.${key}`,
  farmCheck: (key: string) => `ref.farmCheck.${key}`,
  region: (country: string, key: string) => `ref.region.${country}.${key}`,
  city: (country: string, key: string) => `ref.city.${country}.${key}`
};

/** Every label of the catalog, by locale and translation key. */
export function catalogLabels(c: Catalog): Record<'fr' | 'en', Record<string, string>> {
  const out: Record<'fr' | 'en', Record<string, string>> = { fr: {}, en: {} };
  const put = (key: string, label: Label) => {
    out.fr[key] = label.fr;
    out.en[key] = label.en;
  };
  for (const cat of c.categories) {
    put(refKey.category(cat.key), cat.label);
    for (const s of cat.subcategories) {
      put(refKey.species(s.key), s.label);
      put(refKey.unit(s.unit.key), s.unit.label);
    }
  }
  c.forumCategories.forEach((t) => put(refKey.forum(t.key), t.label));
  c.reviewTags.forEach((t) => put(refKey.reviewTag(t.key), t.label));
  c.reportReasons.forEach((t) => put(refKey.reportReason(t.key), t.label));
  c.farmReviewChecks.forEach((t) => put(refKey.farmCheck(t.key), t.label));
  for (const country of c.countries) {
    for (const r of country.regions ?? []) {
      put(refKey.region(country.code, r.key), r.label);
      (r.cities ?? []).forEach((city) => put(refKey.city(country.code, city.key), city.label));
    }
  }
  return out;
}

/** One reference's content, as `GET /reference/{name}` and the
 * reference's schema have it. */
export function partContent(c: CatalogLists, name: CatalogPart): Record<string, unknown> {
  switch (name) {
    case 'animals':
      return { categories: c.categories };
    case 'places':
      return { countries: c.countries };
    default:
      return { entries: c[name] };
  }
}

/** One reference as `GET /reference/{name}` serves it. */
export interface CatalogPartResponse {
  readonly name: CatalogPart;
  readonly version: number;
  readonly categories?: Catalog['categories'];
  readonly cycleTypes?: Catalog['cycleTypes'];
  readonly countries?: Catalog['countries'];
  readonly defaultCountry?: string;
  readonly entries?: readonly CatalogTerm[];
}

/** Puts the references together; a missing one stays empty. */
export function assembleCatalog(parts: readonly (CatalogPartResponse | null)[]): Catalog {
  let c: Catalog = EMPTY_CATALOG;
  for (const p of parts) {
    if (!p) continue;
    c = { ...c, versions: { ...c.versions, [p.name]: p.version } };
    if (p.name === 'animals') c = { ...c, categories: p.categories ?? [], cycleTypes: p.cycleTypes ?? c.cycleTypes };
    else if (p.name === 'places') c = { ...c, countries: p.countries ?? [], defaultCountry: p.defaultCountry ?? c.defaultCountry };
    else c = { ...c, [p.name]: p.entries ?? [] };
  }
  return c;
}
