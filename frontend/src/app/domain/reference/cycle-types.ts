/** The cycle types the sell wizard creates (derived from the species
 * server-side, see domain/species.go), with their label keys. Older
 * cycles may carry other spellings; they're shown as typed. */
export const CYCLE_TYPE_LABEL_KEYS: Readonly<Record<string, string>> = {
  hatching: 'cycleTypes.hatching',
  fattening: 'cycleTypes.fattening',
  laying: 'cycleTypes.laying',
  breeding: 'cycleTypes.breeding',
  rearing: 'cycleTypes.rearing'
};

export function cycleTypeLabelKey(type: string): string {
  return CYCLE_TYPE_LABEL_KEYS[type.toLowerCase()] ?? type;
}
