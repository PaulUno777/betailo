import { CatalogCategory, CatalogSubcategory, refKey } from './catalog';

/** The animal categories and their sub-categories (the "species" cycles
 * and listings store), from the reference data. ReferenceService fills
 * this before the app first renders; these helpers read it so every
 * screen agrees with the version the API validates against. */
let categories: readonly CatalogCategory[] = [];

/** Each catalog tone as a class written out in full: Tailwind only
 * generates the classes it can read in the source, never ones built
 * from strings at run time. */
const DOT_CLASSES: Readonly<Record<string, string>> = {
  clay: 'bg-clay',
  forest: 'bg-forest',
  amber: 'bg-amber',
  sky: 'bg-sky',
  alert: 'bg-alert',
  plum: 'bg-plum',
  stone: 'bg-stone'
};

/** The tones a category can wear, as the API accepts them. */
export const CATEGORY_TONES: readonly string[] = Object.keys(DOT_CLASSES);

export function dotClass(color: string | undefined): string {
  return (color && DOT_CLASSES[color]) || 'bg-stone';
}
let bySpecies = new Map<string, { sub: CatalogSubcategory; category: CatalogCategory }>();

export function setCategories(list: readonly CatalogCategory[]): void {
  categories = list;
  bySpecies = new Map(list.flatMap((category) => category.subcategories.map((sub) => [sub.key, { sub, category }] as const)));
}

/** A category with only its sub-categories open to new listings. */
export interface SpeciesGroup {
  readonly key: string;
  readonly labelKey: string;
  readonly dot: string;
  /** The animal drawing the API gives the category. */
  readonly icon: string;
  readonly species: readonly string[];
}

/** The active categories, each with its active sub-categories, in
 * catalog order: what the sell flow and filters offer. */
export function speciesGroups(): SpeciesGroup[] {
  return categories
    .filter((c) => c.active)
    .map((c) => ({
      key: c.key,
      labelKey: refKey.category(c.key),
      dot: dotClass(c.color),
      icon: c.icon,
      species: c.subcategories.filter((s) => s.active).map((s) => s.key)
    }))
    .filter((g) => g.species.length > 0);
}

/** The groups narrowed to what a farm declared it raises: the only
 * species the API lets it open batches and listings for. */
export function farmSpeciesGroups(declared: readonly string[] | undefined): SpeciesGroup[] {
  const own = new Set(declared ?? []);
  return speciesGroups()
    .map((g) => ({ ...g, species: g.species.filter((s) => own.has(s)) }))
    .filter((g) => g.species.length > 0);
}

/** Every sub-category open to new listings, across categories. */
export function speciesValues(): string[] {
  return speciesGroups().flatMap((g) => g.species);
}

/** Whether the catalog knows this sub-category, active or not (older
 * data keeps its label). */
export function isSpecies(value: string): boolean {
  return bySpecies.has(value);
}

/** The label key for a stored species, or the raw value for one the
 * catalog doesn't know: never a blank. */
export function speciesLabelKey(value: string): string {
  return isSpecies(value) ? refKey.species(value) : value;
}

/** The category tone a species wears on chips and choice cards. */
export function speciesDot(value: string): string {
  return dotClass(bySpecies.get(value)?.category.color);
}

/** The label key of what one animal of this species is called. */
export function speciesUnitKey(value: string): string {
  const unit = bySpecies.get(value)?.sub.unit.key;
  return unit ? refKey.unit(unit) : '';
}

/** The animal drawing the API gives a species ('' for one it doesn't know). */
export function speciesIcon(value: string): string {
  return bySpecies.get(value)?.sub.icon ?? '';
}

/** The animal drawing the API gives a category. */
export function categoryIcon(value: string): string {
  return categories.find((c) => c.key === value)?.icon ?? '';
}

/** The category a species belongs to. */
export function categoryOf(value: string): string | undefined {
  return bySpecies.get(value)?.category.key;
}

/** Whether the catalog has this category. */
export function isCategory(value: string): boolean {
  return categories.some((c) => c.key === value);
}

export function categoryLabelKey(value: string): string {
  return isCategory(value) ? refKey.category(value) : value;
}

/** The sub-categories of a category, active or not. */
export function speciesOf(category: string): string[] {
  return categories.find((c) => c.key === category)?.subcategories.map((s) => s.key) ?? [];
}

/** The cycle type a species is raised in (the API sets it from this). */
export function speciesCycleType(value: string): string {
  return bySpecies.get(value)?.sub.cycleType ?? '';
}
