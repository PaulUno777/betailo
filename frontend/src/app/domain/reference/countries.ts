/** The countries offered when a farm says where it is: Africa first
 * (where Betailo works), as ISO 3166-1 alpha-2 codes. Names come from
 * the browser in the user's language (Intl.DisplayNames), so adding a
 * country is one code here and nothing to translate. */
export const FARM_COUNTRIES: readonly string[] = [
  'CM', 'BJ', 'BF', 'BI', 'CF', 'CG', 'CD', 'CI', 'DZ', 'EG', 'ET', 'GA', 'GH', 'GN', 'GQ', 'KE', 'MA', 'MG',
  'ML', 'MR', 'MW', 'MZ', 'NE', 'NG', 'RW', 'SN', 'TD', 'TG', 'TN', 'TZ', 'UG', 'ZA', 'ZM', 'ZW', 'BE', 'CA', 'FR', 'US'
];

/** "Cameroun" for "CM" in French, "Cameroon" in English. */
export function countryName(code: string, locale: string): string {
  try {
    return new Intl.DisplayNames([locale], { type: 'region' }).of(code) ?? code;
  } catch {
    return code;
  }
}
