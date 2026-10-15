import { TestBed } from '@angular/core/testing';
import { LocaleService } from '../core/i18n/locale.service';
import { Catalog, catalogLabels } from '../domain/reference/catalog';
import { setCategories } from '../domain/reference/species';
import { setCountries } from '../domain/reference/countries';

/** A small reference catalog for specs: the shape `GET /reference`
 * serves, with a few entries of each list. */
export const TEST_CATALOG: Catalog = {
  version: 3,
  publishedAt: '2026-10-15T08:00:00Z',
  defaultCountry: 'CM',
  cycleTypes: ['couvee', 'engraissement', 'ponte', 'reproduction', 'elevage'],
  categories: [
    {
      key: 'volaille',
      label: { fr: 'Volaille', en: 'Poultry' },
      color: 'clay',
      active: true,
      subcategories: [
        { key: 'poussins', label: { fr: 'Poussins d’un jour', en: 'Day-old chicks' }, cycleType: 'couvee', unit: { key: 'poussin', label: { fr: 'poussin', en: 'chick' } }, active: true },
        { key: 'dindes', label: { fr: 'Dindes', en: 'Turkeys' }, cycleType: 'engraissement', unit: { key: 'dinde', label: { fr: 'dinde', en: 'turkey' } }, active: true },
        { key: 'cailles', label: { fr: 'Cailles', en: 'Quails' }, cycleType: 'elevage', unit: { key: 'caille', label: { fr: 'caille', en: 'quail' } }, active: false }
      ]
    },
    {
      key: 'ovins',
      label: { fr: 'Ovins', en: 'Sheep' },
      color: 'stone',
      active: true,
      subcategories: [{ key: 'agneaux', label: { fr: 'Agneaux', en: 'Lambs' }, cycleType: 'elevage', unit: { key: 'agneau', label: { fr: 'agneau', en: 'lamb' } }, active: true }]
    },
    {
      key: 'bovins',
      label: { fr: 'Bovins', en: 'Cattle' },
      color: 'forest',
      active: false,
      subcategories: [{ key: 'veaux', label: { fr: 'Veaux', en: 'Calves' }, cycleType: 'elevage', unit: { key: 'veau', label: { fr: 'veau', en: 'calf' } }, active: true }]
    }
  ],
  countries: [
    { code: 'CM', callingCode: '237', active: true, regions: [{ key: 'ouest', label: 'Ouest', active: true }, { key: 'littoral', label: 'Littoral', active: true }] },
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
