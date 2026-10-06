/** The species filter/category vocabulary — shared by Home's hero search
 * and the Marketplace filter sidebar so the two can't drift apart. Values
 * match what `GET /listings`'s `species` query param expects; labels are
 * translation keys, resolved by whoever renders them. */
export const SPECIES_VALUES = ['poussins', 'jeunes-poulets', 'pondeuses', 'reproducteurs', 'porcs', 'chevres'] as const;

export type SpeciesValue = (typeof SPECIES_VALUES)[number];

export const SPECIES_LABEL_KEYS: Readonly<Record<SpeciesValue, string>> = {
  poussins: 'species.poussins',
  'jeunes-poulets': 'species.jeunesPoulets',
  pondeuses: 'species.pondeuses',
  reproducteurs: 'species.reproducteurs',
  porcs: 'species.porcs',
  chevres: 'species.chevres'
};

/** The dot colour each species wears on chips and choice cards. */
export const SPECIES_DOT: Readonly<Record<SpeciesValue, string>> = {
  poussins: 'bg-clay',
  'jeunes-poulets': 'bg-forest',
  pondeuses: 'bg-amber',
  reproducteurs: 'bg-sky',
  porcs: 'bg-alert',
  chevres: 'bg-plum'
};

/** What one unit is called, for display before the API has created the
 * listing (the sell wizard's preview). The API derives the stored unit
 * itself from the species; this only mirrors it as a translation key. */
export const SPECIES_UNIT_KEYS: Readonly<Record<SpeciesValue, string>> = {
  poussins: 'units.poussin',
  'jeunes-poulets': 'units.sujet',
  pondeuses: 'units.pondeuse',
  reproducteurs: 'units.sujet',
  porcs: 'units.porcelet',
  chevres: 'units.chevre'
};

export function isSpecies(value: string): value is SpeciesValue {
  return (SPECIES_VALUES as readonly string[]).includes(value);
}

/** The label key for a stored species, or the raw value for one this
 * client doesn't know (older data): never a blank. */
export function speciesLabelKey(value: string): string {
  return isSpecies(value) ? SPECIES_LABEL_KEYS[value] : value;
}
