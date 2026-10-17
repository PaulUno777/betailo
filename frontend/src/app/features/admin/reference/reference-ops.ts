import { Catalog, CatalogCategory, CatalogCity, CatalogCountry, CatalogRegion, CatalogSubcategory, CatalogTerm, Label } from '../../../domain/reference/catalog';
import { CatalogRef, Change, Entity, Mark, TermList } from './reference.types';

export type Entry = CatalogCategory | CatalogSubcategory | CatalogCountry | CatalogRegion | CatalogCity | CatalogTerm;

/** The children an update never carries: they are entries of their own. */
const CHILDREN: Partial<Record<Entity, string>> = { category: 'subcategories', country: 'regions', region: 'cities' };

/** "Poules d’ornement" → "poules-d-ornement": the API's key shape. */
export function slugify(text: string): string {
  return text
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

/** The entry ref names in catalog c, if it is there. */
export function entryOf(c: Catalog | undefined, ref: CatalogRef): Entry | undefined {
  if (!c) return undefined;
  switch (ref.entity) {
    case 'category':
      return c.categories.find((x) => x.key === ref.key);
    case 'subcategory':
      return c.categories.flatMap((x) => x.subcategories).find((x) => x.key === ref.key);
    case 'country':
      return c.countries.find((x) => x.code === ref.key);
    case 'region':
      return c.countries.find((x) => x.code === ref.country)?.regions?.find((x) => x.key === ref.key);
    case 'city':
      return c.countries
        .find((x) => x.code === ref.country)
        ?.regions?.flatMap((r) => r.cities ?? [])
        .find((x) => x.key === ref.key);
    case 'term':
      return (c[ref.list as TermList] ?? []).find((x) => x.key === ref.key);
  }
}

/** An entry's own fields: no key, no children. */
export function ownFields(entity: Entity, entry: object): Record<string, unknown> {
  const out: Record<string, unknown> = { ...(entry as Record<string, unknown>) };
  delete out['key'];
  delete out['code'];
  const children = CHILDREN[entity];
  if (children) delete out[children];
  return out;
}

/** The fields of edited that differ from original: an update's value
 * (a partial edit), or null when nothing changed. */
export function changedFields(original: Record<string, unknown>, edited: Record<string, unknown>): Record<string, unknown> | null {
  const out: Record<string, unknown> = {};
  for (const [name, value] of Object.entries(edited)) {
    if (JSON.stringify(value) !== JSON.stringify(original[name])) out[name] = value;
  }
  return Object.keys(out).length ? out : null;
}

/** The stable id of an entry, for marks and usage: "subcategory:chicks",
 * "city:CM:bafoussam", "term:reviewTags:on_time". */
export function refId(ref: CatalogRef): string {
  switch (ref.entity) {
    case 'region':
    case 'city':
      return `${ref.entity}:${ref.country}:${ref.key}`;
    case 'term':
      return `term:${ref.list}:${ref.key}`;
    default:
      return `${ref.entity}:${ref.key}`;
  }
}

/** What the pending changes did to each entry. An entry added and then
 * edited stays "added". */
export function marksOf(changes: readonly Change[]): ReadonlyMap<string, Mark> {
  const marks = new Map<string, Mark>();
  for (const c of changes) {
    const id = refId(c.op.ref);
    const now: Mark = c.op.kind === 'add' ? 'added' : c.op.kind === 'update' ? 'edited' : 'moved';
    const before = marks.get(id);
    if (before === 'added' || (before === 'edited' && now === 'moved')) continue;
    marks.set(id, now);
  }
  return marks;
}

/** The usage key the API counts an entry under. */
export function usageKey(ref: CatalogRef): string {
  switch (ref.entity) {
    case 'subcategory':
      return `species:${ref.key}`;
    case 'country':
      return `country:${ref.key}`;
    case 'region':
      return `region:${ref.country}:${ref.key}`;
    case 'city':
      return `city:${ref.country}:${ref.key}`;
    case 'term':
      return `${ref.list}:${ref.key}`;
    default:
      return '';
  }
}

/** An entry's name in the reader's language (a country's from Intl). */
export function nameOf(entry: Entry | undefined, lang: 'fr' | 'en', ref?: CatalogRef): string {
  if (!entry) return ref?.key ?? '';
  if ('label' in entry) return (entry.label as Label)[lang] || (entry.label as Label).fr;
  try {
    return new Intl.DisplayNames([lang], { type: 'region' }).of((entry as CatalogCountry).code) ?? (entry as CatalogCountry).code;
  } catch {
    return (entry as CatalogCountry).code;
  }
}

/** One line of a bulk paste: "Nom FR ; Name EN", "key;fr;en", tabs or
 * semicolons; the English name defaults to the French one. */
export interface PastedEntry {
  readonly key: string;
  readonly fr: string;
  readonly en: string;
}

export function parsePaste(text: string): PastedEntry[] {
  return text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => line.split(/\t|;/).map((p) => p.trim()))
    .filter((parts) => !(parts.length >= 2 && /^(key|cl[ée])$/i.test(parts[0]))) // a CSV header
    .map((parts) => {
      if (parts.length >= 3) return { key: slugify(parts[0]), fr: parts[1], en: parts[2] || parts[1] };
      return { key: slugify(parts[0]), fr: parts[0], en: parts[1] || parts[0] };
    });
}

