import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { firstValueFrom } from 'rxjs';
import { ReferenceService } from './reference.service';
import { LocaleService } from '../i18n/locale.service';
import { API_BASE_URL } from '../config/api.config';
import { TEST_CATALOG } from '../../testing/reference.fixture';
import { categoryOf, isSpecies, speciesCycleType, speciesDot, speciesGroups, speciesLabelKey, speciesUnitKey } from '../../domain/reference/species';
import { callingCodeOf, farmCountries, regionsOf } from '../../domain/reference/countries';

describe('ReferenceService', () => {
  let http: HttpTestingController;

  beforeEach(() => {
    localStorage.removeItem('betailo.reference');
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), { provide: API_BASE_URL, useValue: 'https://api.test' }]
    });
    http = TestBed.inject(HttpTestingController);
    TestBed.inject(LocaleService).setLocale('fr');
  });

  afterEach(() => http.verify());

  it('hands the catalog to the helpers and registers its labels', async () => {
    const loaded = firstValueFrom(TestBed.inject(ReferenceService).load(), { defaultValue: undefined });
    http.expectOne('https://api.test/reference').flush(TEST_CATALOG);
    await loaded;

    // Only active categories with active sub-categories are offered.
    expect(speciesGroups().map((g) => [g.key, g.species])).toEqual([
      ['volaille', ['poussins', 'dindes']],
      ['ovins', ['agneaux']]
    ]);
    expect(isSpecies('cailles')).toBeTrue(); // inactive, but older listings keep their label
    expect(categoryOf('agneaux')).toBe('ovins');
    expect(speciesDot('dindes')).toBe('bg-clay');
    expect(speciesCycleType('poussins')).toBe('couvee');

    const locale = TestBed.inject(LocaleService);
    expect(locale.t(speciesLabelKey('dindes'))).toBe('Dindes');
    expect(locale.t(speciesUnitKey('agneaux'))).toBe('agneau');
    expect(speciesLabelKey('licornes')).toBe('licornes'); // unknown: shown as stored
    locale.setLocale('en');
    expect(locale.t(speciesLabelKey('dindes'))).toBe('Turkeys');

    expect(farmCountries()).toEqual(['CM', 'GH']);
    expect(regionsOf('CM')).toEqual(['Ouest', 'Littoral']);
    expect(regionsOf('GH')).toBeNull();
    expect(callingCodeOf('GH')).toBe('233');
  });

  it('falls back to the copy from the last visit when the API fails', async () => {
    localStorage.setItem('betailo.reference', JSON.stringify(TEST_CATALOG));
    const service = TestBed.inject(ReferenceService);
    const loaded = firstValueFrom(service.load(), { defaultValue: undefined });
    http.expectOne('https://api.test/reference').flush('down', { status: 503, statusText: 'Unavailable' });
    await loaded;
    expect(service.catalog().version).toBe(3);
    expect(service.forumOptions()).toEqual([{ value: 'sante', label: 'Santé animale' }]);
  });
});
