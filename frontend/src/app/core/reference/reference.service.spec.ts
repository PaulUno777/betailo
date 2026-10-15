import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { firstValueFrom } from 'rxjs';
import { ReferenceService } from './reference.service';
import { LocaleService } from '../i18n/locale.service';
import { API_BASE_URL } from '../config/api.config';
import { testPart } from '../../testing/reference.fixture';
import { CATALOG_PARTS } from '../../domain/reference/catalog';
import { categoryOf, isSpecies, speciesCycleType, speciesDot, speciesGroups, speciesLabelKey, speciesUnitKey } from '../../domain/reference/species';
import { callingCodeOf, citiesOf, farmCountries, placeLabelKey, regionOfCity, regionsOf } from '../../domain/reference/countries';

describe('ReferenceService', () => {
  let http: HttpTestingController;

  beforeEach(() => {
    CATALOG_PARTS.forEach((name) => localStorage.removeItem(`betailo.reference.${name}`));
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), { provide: API_BASE_URL, useValue: 'https://api.test' }]
    });
    http = TestBed.inject(HttpTestingController);
    TestBed.inject(LocaleService).setLocale('fr');
  });

  afterEach(() => http.verify());

  it('hands the catalog to the helpers and registers its labels', async () => {
    const loaded = firstValueFrom(TestBed.inject(ReferenceService).load(), { defaultValue: undefined });
    CATALOG_PARTS.forEach((name) => http.expectOne(`https://api.test/reference/${name}`).flush(testPart(name)));
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
    expect(regionsOf('CM')?.map((r) => r.key)).toEqual(['ouest', 'extreme-nord']);
    expect(citiesOf('CM', 'extreme-nord').map((c) => c.key)).toEqual(['kousseri']);
    expect(regionOfCity('kousseri')).toBe('extreme-nord');
    // Stored keys read in the reader's language; typed text as typed.
    expect(locale.t(placeLabelKey('extreme-nord'))).toBe('Far North');
    expect(locale.t(placeLabelKey('kousseri'))).toBe('Kousseri');
    expect(locale.t(placeLabelKey('Bamendjou'))).toBe('Bamendjou');
    locale.setLocale('fr');
    expect(locale.t(placeLabelKey('kousseri'))).toBe('Kousséri');
    expect(regionsOf('GH')).toBeNull();
    expect(callingCodeOf('GH')).toBe('233');
  });

  it('falls back to the copy from the last visit when the API fails', async () => {
    // The places failed to load: the copy from the last visit stands in.
    localStorage.setItem('betailo.reference.places', JSON.stringify(testPart('places')));
    const service = TestBed.inject(ReferenceService);
    const loaded = firstValueFrom(service.load(), { defaultValue: undefined });
    for (const name of CATALOG_PARTS) {
      const req = http.expectOne(`https://api.test/reference/${name}`);
      if (name === 'places') req.flush('down', { status: 503, statusText: 'Unavailable' });
      else req.flush(testPart(name));
    }
    await loaded;
    expect(service.catalog().versions.places).toBe(3);
    expect(service.catalog().countries.length).toBeGreaterThan(0);
    expect(service.forumOptions()).toEqual([{ value: 'sante', label: 'Santé animale' }]);
  });
});
