/** An animal's age as people say it, from the days the API stores:
 * "1 jour" for chicks, weeks up to three months, then months, then
 * years. Returns the translation key and its count. */
export function ageLabel(days: number): { key: string; count: number } {
  if (days < 7) return { key: 'age.days', count: days };
  if (days <= 84 && days % 7 === 0) return { key: 'age.weeks', count: days / 7 };
  if (days < 365) return { key: 'age.months', count: Math.max(1, Math.round(days / 30.4)) };
  return { key: 'age.years', count: Math.round(days / 365) };
}

/** What the sell wizard offers, in days: the first days (chicks), then
 * weeks, months and years. */
export const AGE_CHOICES_DAYS = [1, 7, 14, 21, 28, 42, 56, 70, 84, 122, 152, 182, 274, 365, 730] as const;
