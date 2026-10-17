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
  cycleTypes: ['hatching', 'fattening', 'laying', 'breeding', 'rearing'],
  categories: [
    {
      key: 'poultry',
      label: { fr: 'Volaille', en: 'Poultry' },
      icon: 'hen',
      color: 'clay',
      active: true,
      subcategories: [
        { key: 'chicks', label: { fr: 'Poussins', en: 'Chicks' }, cycleType: 'hatching', icon: 'chick', unit: { key: 'chick', label: { fr: 'chick', en: 'chick' } }, active: true },
        { key: 'turkeys', label: { fr: 'Dindes', en: 'Turkeys' }, cycleType: 'fattening', icon: 'turkey', unit: { key: 'turkey', label: { fr: 'turkey', en: 'turkey' } }, active: true },
        { key: 'quails', label: { fr: 'Cailles', en: 'Quails' }, cycleType: 'rearing', icon: 'hen', unit: { key: 'quail', label: { fr: 'quail', en: 'quail' } }, active: false }
      ]
    },
    {
      key: 'ovine',
      label: { fr: 'Ovins', en: 'Sheep' },
      icon: 'sheep',
      color: 'stone',
      active: true,
      subcategories: [{ key: 'lambs', label: { fr: 'Agneaux', en: 'Lambs' }, cycleType: 'rearing', icon: 'sheep', unit: { key: 'lamb', label: { fr: 'lamb', en: 'lamb' } }, active: true }]
    },
    {
      key: 'cattle',
      label: { fr: 'Bovins', en: 'Cattle' },
      icon: 'cow',
      color: 'forest',
      active: false,
      subcategories: [{ key: 'calves', label: { fr: 'Veaux', en: 'Calves' }, cycleType: 'rearing', icon: 'cow', unit: { key: 'calf', label: { fr: 'calf', en: 'calf' } }, active: true }]
    }
  ],
  countries: [
    {
      code: 'CM',
      callingCode: '237',
      active: true,
      regions: [
        {
          key: 'west',
          label: { fr: 'Ouest', en: 'West' },
          active: true,
          cities: [{ key: 'bafoussam', label: { fr: 'Bafoussam', en: 'Bafoussam' }, active: true }]
        },
        {
          key: 'far-north',
          label: { fr: 'Extrême-Nord', en: 'Far North' },
          active: true,
          cities: [{ key: 'kousseri', label: { fr: 'Kousséri', en: 'Kousseri' }, active: true }]
        }
      ]
    },
    { code: 'GH', callingCode: '233', active: true }
  ],
  forumCategories: [{ key: 'health', label: { fr: 'Santé animale', en: 'Animal health' }, active: true }],
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
