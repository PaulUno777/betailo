import { CatalogCountry, refKey } from './catalog';

/** The countries farms can be in, from the reference data (see
 * species.ts for how it is filled). Country names come from the browser
 * in the user's language (Intl.DisplayNames); region and town names
 * come from the catalog, in both languages. */
let countries: readonly CatalogCountry[] = [];
let defaultCountry = 'CM';
/** Where each stored region or town key belongs, for showing it. */
let places = new Map<string, { country: string; region: string; city?: string }>();

export function setCountries(list: readonly CatalogCountry[], fallback: string): void {
  countries = list;
  defaultCountry = fallback;
  places = new Map();
  for (const c of list) {
    for (const r of c.regions ?? []) {
      places.set(`region:${r.key}`, { country: c.code, region: r.key });
      for (const city of r.cities ?? []) places.set(`city:${city.key}`, { country: c.code, region: r.key, city: city.key });
    }
  }
}

/** A region or town as offered in a picker: its key and label key. */
export interface PlaceOption {
  readonly key: string;
  readonly labelKey: string;
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
export function regionsOf(code: string): PlaceOption[] | null {
  const regions = countries.find((c) => c.code === code)?.regions;
  return regions?.length ? regions.filter((r) => r.active).map((r) => ({ key: r.key, labelKey: refKey.region(code, r.key) })) : null;
}

/** The listed towns of a region (empty when it lists none). */
export function citiesOf(code: string, region: string): PlaceOption[] {
  const r = countries.find((c) => c.code === code)?.regions?.find((x) => x.key === region);
  return (r?.cities ?? []).filter((c) => c.active).map((c) => ({ key: c.key, labelKey: refKey.city(code, c.key) }));
}

/** The region a listed town is in, or null for a town typed freely. */
export function regionOfCity(city: string): string | null {
  return places.get(`city:${city}`)?.region ?? null;
}

/** The label key of a stored town or region: the catalog's name for a
 * key it lists, the text itself for a place typed freely. */
export function placeLabelKey(value: string): string {
  const city = places.get(`city:${value}`);
  if (city) return refKey.city(city.country, value);
  const region = places.get(`region:${value}`);
  return region ? refKey.region(region.country, value) : value;
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
