import { TestBed } from '@angular/core/testing';
import { LocaleService } from '../../../core/i18n/locale.service';
import { useTestReference } from '../../../testing/reference.fixture';
import { PlaceFieldsComponent } from './place-fields.component';

describe('PlaceFieldsComponent', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({ imports: [PlaceFieldsComponent] });
    useTestReference();
    TestBed.inject(LocaleService).setLocale('en');
  });

  function render(region = '', location = '') {
    const fixture = TestBed.createComponent(PlaceFieldsComponent);
    fixture.componentRef.setInput('country', 'CM');
    fixture.componentRef.setInput('region', region);
    fixture.componentRef.setInput('location', location);
    fixture.detectChanges();
    const el: HTMLElement = fixture.nativeElement;
    const selects = () => Array.from(el.querySelectorAll('select'));
    const texts = () => Array.from(el.querySelectorAll('input:not([type=tel])')) as HTMLInputElement[];
    const options = (s: HTMLSelectElement) => Array.from(s.options).map((o) => o.textContent?.trim());
    return { fixture, selects, texts, options };
  }

  async function pick(fixture: ReturnType<typeof render>['fixture'], select: HTMLSelectElement, value: string) {
    select.value = value;
    select.dispatchEvent(new Event('change'));
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
  }

  it('names regions and towns in the reader’s language and stores their keys', async () => {
    const { fixture, selects, texts, options } = render();
    const [, region] = selects();
    expect(options(region)).toContain('Far North');
    expect(texts().length).toBe(1); // no region yet: the town is typed

    await pick(fixture, region, 'far-north');
    const town = selects()[2];
    expect(options(town)).toEqual(['Pick your town', 'Kousseri', 'Another town or village…']);

    await pick(fixture, town, 'kousseri');
    expect(fixture.componentInstance.location()).toBe('kousseri');
    expect(texts().length).toBe(0);
  });

  it('types a town the list doesn’t have under "another town"', async () => {
    const { fixture, selects, texts } = render('west', 'bafoussam');
    await pick(fixture, selects()[2], '__other__');
    expect(fixture.componentInstance.location()).toBe('');
    expect(texts().length).toBe(1);
  });

  it('opens a farm with a typed town on "another town", and a listed town fills its region', async () => {
    const typed = render('west', 'Bamendjou');
    await typed.fixture.whenStable();
    typed.fixture.detectChanges();
    expect(typed.texts().map((i) => i.value)).toEqual(['Bamendjou']);
    const { fixture } = render('', 'kousseri');
    expect(fixture.componentInstance.region()).toBe('far-north');
  });
});
