import { CatalogCountry } from './catalog';

/** The countries farms can be in, from the reference data (see
 * species.ts for how it is filled). Names come from the browser in the
 * user's language (Intl.DisplayNames), so the catalog only holds codes. */
let countries: readonly CatalogCountry[] = [];
let defaultCountry = 'CM';

export function setCountries(list: readonly CatalogCountry[], fallback: string): void {
  countries = list;
  defaultCountry = fallback;
}

/** Active country codes, in catalog order (the default country first). */
export function farmCountries(): string[] {
  return countries.filter((c) => c.active).map((c) => c.code);
}

export function defaultFarmCountry(): string {
  return defaultCountry;
}

/** The active regions of a country, or null when its region is typed
 * freely. */
export function regionsOf(code: string): string[] | null {
  const regions = countries.find((c) => c.code === code)?.regions;
  return regions?.length ? regions.filter((r) => r.active).map((r) => r.label) : null;
}

/** "233" for Ghana; undefined for a country the catalog doesn't know. */
export function callingCodeOf(code: string): string | undefined {
  return countries.find((c) => c.code === code)?.callingCode;
}

/** "Cameroun" for "CM" in French, "Cameroon" in English. */
export function countryName(code: string, locale: string): string {
  try {
    return new Intl.DisplayNames([locale], { type: 'region' }).of(code) ?? code;
  } catch {
    return code;
  }
}
