import { TestBed } from '@angular/core/testing';
import { LocaleService } from '../core/i18n/locale.service';
import { CATALOG_PARTS, Catalog, CatalogPartResponse, catalogLabels, partContent } from '../domain/reference/catalog';
import { setCategories } from '../domain/reference/species';
import { setCountries } from '../domain/reference/countries';

/** A small reference catalog for specs: the shape `GET /reference`
 * serves, with a few entries of each list. */
export const TEST_CATALOG: Catalog = {
  versions: { animals: 3, places: 3, forumCategories: 3, reviewTags: 3, reportReasons: 3, farmReviewChecks: 3 },
  defaultCountry: 'CM',
  cycleTypes: ['couvee', 'engraissement', 'ponte', 'reproduction', 'elevage'],
  categories: [
    {
      key: 'volaille',
      label: { fr: 'Volaille', en: 'Poultry' },
      icon: 'poule',
      color: 'clay',
      active: true,
      subcategories: [
        { key: 'poussins', label: { fr: 'Poussins et jeunes', en: 'Chicks and young birds' }, cycleType: 'couvee', icon: 'poussin', unit: { key: 'sujet', label: { fr: 'sujet', en: 'bird' } }, active: true },
        { key: 'dindes', label: { fr: 'Dindes', en: 'Turkeys' }, cycleType: 'engraissement', icon: 'dinde', unit: { key: 'dinde', label: { fr: 'dinde', en: 'turkey' } }, active: true },
        { key: 'cailles', label: { fr: 'Cailles', en: 'Quails' }, cycleType: 'elevage', icon: 'poule', unit: { key: 'caille', label: { fr: 'caille', en: 'quail' } }, active: false }
      ]
    },
    {
      key: 'ovins',
      label: { fr: 'Ovins', en: 'Sheep' },
      icon: 'mouton',
      color: 'stone',
      active: true,
      subcategories: [{ key: 'agneaux', label: { fr: 'Agneaux', en: 'Lambs' }, cycleType: 'elevage', icon: 'mouton', unit: { key: 'agneau', label: { fr: 'agneau', en: 'lamb' } }, active: true }]
    },
    {
      key: 'bovins',
      label: { fr: 'Bovins', en: 'Cattle' },
      icon: 'vache',
      color: 'forest',
      active: false,
      subcategories: [{ key: 'veaux', label: { fr: 'Veaux', en: 'Calves' }, cycleType: 'elevage', icon: 'vache', unit: { key: 'veau', label: { fr: 'veau', en: 'calf' } }, active: true }]
    }
  ],
  countries: [
    {
      code: 'CM',
      callingCode: '237',
      active: true,
      regions: [
        {
          key: 'ouest',
          label: { fr: 'Ouest', en: 'West' },
          active: true,
          cities: [{ key: 'bafoussam', label: { fr: 'Bafoussam', en: 'Bafoussam' }, active: true }]
        },
        {
          key: 'extreme-nord',
          label: { fr: 'Extrême-Nord', en: 'Far North' },
          active: true,
          cities: [{ key: 'kousseri', label: { fr: 'Kousséri', en: 'Kousseri' }, active: true }]
        }
      ]
    },
    { code: 'GH', callingCode: '233', active: true }
  ],
  forumCategories: [{ key: 'sante', label: { fr: 'Santé animale', en: 'Animal health' }, active: true }],
  reviewTags: [{ key: 'responsive_farmer', label: { fr: 'Éleveur réactif', en: 'Responsive farmer' }, active: true }],
  reportReasons: [{ key: 'other', label: { fr: 'Autre', en: 'Other' }, active: true }],
  farmReviewChecks: [{ key: 'name_matches', label: { fr: 'Le nom correspond', en: 'The name matches' }, active: true }]
};

/** Loads TEST_CATALOG the way ReferenceService does, for specs that
 * render reference labels or lists. Call after configureTestingModule. */
export function useTestReference(): void {
  setCategories(TEST_CATALOG.categories);
  setCountries(TEST_CATALOG.countries, TEST_CATALOG.defaultCountry);
  TestBed.inject(LocaleService).setRuntimeLabels(catalogLabels(TEST_CATALOG));
}

/** TEST_CATALOG's reference name, as `GET /reference/{name}` serves it. */
export function testPart(name: (typeof CATALOG_PARTS)[number]): CatalogPartResponse {
  const extra = name === 'animals' ? { cycleTypes: TEST_CATALOG.cycleTypes } : name === 'places' ? { defaultCountry: TEST_CATALOG.defaultCountry } : {};
  return { name, version: 3, ...partContent(TEST_CATALOG, name), ...extra } as CatalogPartResponse;
}
