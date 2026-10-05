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
