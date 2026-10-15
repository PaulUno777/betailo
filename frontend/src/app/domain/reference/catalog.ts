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

export interface CatalogSubcategory {
  readonly key: string;
  readonly label: Label;
  readonly cycleType: string;
  readonly unit: { readonly key: string; readonly label: Label };
  readonly active: boolean;
}

export interface CatalogCategory {
  readonly key: string;
  readonly label: Label;
  /** A design-system tone: clay, forest, amber, sky, alert, plum, stone. */
  readonly color: string;
  readonly active: boolean;
  readonly subcategories: readonly CatalogSubcategory[];
}

export interface CatalogRegion {
  readonly key: string;
  readonly label: string;
  readonly active: boolean;
}

export interface CatalogCountry {
  readonly code: string;
  readonly callingCode: string;
  readonly active: boolean;
  readonly regions?: readonly CatalogRegion[];
}

export interface Catalog {
  readonly version: number;
  readonly note?: string;
  readonly publishedAt: string;
  readonly defaultCountry: string;
  readonly cycleTypes: readonly string[];
  readonly categories: readonly CatalogCategory[];
  readonly countries: readonly CatalogCountry[];
  readonly forumCategories: readonly CatalogTerm[];
  readonly reviewTags: readonly CatalogTerm[];
  readonly reportReasons: readonly CatalogTerm[];
  readonly farmReviewChecks: readonly CatalogTerm[];
}

/** An empty catalog: what the app holds until `GET /reference` answers. */
export const EMPTY_CATALOG: Catalog = {
  version: 0,
  publishedAt: '',
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
  farmCheck: (key: string) => `ref.farmCheck.${key}`
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
  return out;
}
