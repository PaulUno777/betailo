import { Catalog, CatalogPart } from '../../../domain/reference/catalog';

/** The kinds of reference entry, as the API names them. */
export type Entity = 'category' | 'subcategory' | 'country' | 'region' | 'city' | 'term';
export type TermList = 'forumCategories' | 'reviewTags' | 'reportReasons' | 'farmReviewChecks';
export const TERM_LISTS: readonly TermList[] = ['forumCategories', 'reviewTags', 'reportReasons', 'farmReviewChecks'];

/** Names one entry: its key (a country's code) and where it lives;
 * `parent` is, for an add or a move, the category of a kind or the
 * region of a town. */
export interface CatalogRef {
  readonly entity: Entity;
  readonly key: string;
  readonly country?: string;
  readonly list?: TermList;
  readonly parent?: string;
}

/** One change: add a whole entry, update some of its fields, or move
 * it (in its list, or to another parent). */
export interface CatalogOp {
  readonly kind: 'add' | 'update' | 'move';
  readonly ref: CatalogRef;
  readonly value?: Record<string, unknown>;
  readonly position?: number;
}

export interface Change {
  readonly id: string;
  readonly op: CatalogOp;
  /** For an update: the values it replaces. */
  readonly before?: Record<string, unknown>;
  readonly by: { readonly id: string; readonly name: string };
  readonly at: string;
  /** Set when the change no longer applies to the version in force. */
  readonly error?: { readonly code: string; readonly params?: Record<string, unknown> };
}

/** The shared pending changes, replayed onto the version in force. */
export interface Changeset {
  readonly rev: number;
  /** The version in force of each reference. */
  readonly baseVersions: Readonly<Partial<Record<CatalogPart, number>>>;
  /** The references the changes act on: a publish versions those. */
  readonly touched: readonly CatalogPart[];
  readonly ready: boolean;
  readonly problems: readonly string[];
  readonly changes: readonly Change[];
  readonly draft: Catalog;
}

export interface CatalogVersion {
  readonly name: CatalogPart;
  readonly version: number;
  readonly note?: string;
  readonly publishedAt: string;
  readonly fromSeed: boolean;
}

/** How many farms, cycles, posts… use each entry ("species:chicks"). */
export type Usage = Readonly<Record<string, number>>;

/** Where an entry stands in the draft. */
export type Mark = 'added' | 'edited' | 'moved';

/** What the drawer is opened for: a new entry under a parent, or one to
 * edit. */
export interface EntryRequest {
  readonly mode: 'add' | 'edit';
  readonly entity: Entity;
  /** Edit: the entry. Add: where it goes (country, list, parent). */
  readonly ref: Omit<CatalogRef, 'key'> & { readonly key?: string };
}

/** Where pasted entries go: towns of a region, regions of a country,
 * kinds of a category, or entries of a list. */
export interface BulkTarget {
  readonly entity: Extract<Entity, 'subcategory' | 'region' | 'city' | 'term'>;
  readonly country?: string;
  readonly parent?: string;
  readonly list?: TermList;
  /** Shown in the title: "Ouest", "Volaille". */
  readonly where: string;
}

/** The reference an entry belongs to. */
export function partOf(ref: CatalogRef): CatalogPart {
  switch (ref.entity) {
    case 'category':
    case 'subcategory':
      return 'animals';
    case 'country':
    case 'region':
    case 'city':
      return 'places';
    default:
      return ref.list as CatalogPart;
  }
}
