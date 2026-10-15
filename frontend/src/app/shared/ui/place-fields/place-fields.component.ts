import { ChangeDetectionStrategy, Component, computed, effect, inject, linkedSignal, model } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { callingCodeOf, citiesOf, countryName, defaultFarmCountry, farmCountries, regionOfCity, regionsOf } from '../../../domain/reference/countries';
import { LocaleService } from '../../../core/i18n/locale.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { SelectFieldComponent, SelectOption } from '../select-field/select-field.component';
import { TextFieldComponent } from '../text-field/text-field.component';

/** Where a farm is and how to reach it: country, region, town, phone.
 * The region is a list where the API knows the country's regions
 * (Cameroon's ten) and free text anywhere else, so a farmer in any
 * country can sell, and buyers can filter by an exact region. The town
 * is picked from the region's main towns, or typed under "Autre ville";
 * `location` holds the town's key or the typed text, as the API stores
 * it. Town is what buyers see on cards; the phone shows its country's
 * prefix. */
@Component({
  selector: 'bt-place-fields',
  imports: [FormsModule, SelectFieldComponent, TextFieldComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <div class="grid gap-5 sm:grid-cols-2">
        <bt-select-field
          [label]="'placeFields.country' | t"
          [options]="countryOptions()"
          name="country"
          [ngModel]="country()"
          (ngModelChange)="setCountry($event)"
        />
        @if (regionOptions(); as options) {
          <bt-select-field
            [label]="'placeFields.region' | t"
            [options]="options"
            [placeholder]="'placeFields.regionPick' | t"
            name="region"
            [ngModel]="region()"
            (ngModelChange)="setRegion($event)"
          />
        } @else {
          <bt-text-field
            [label]="'placeFields.region' | t"
            [placeholder]="'placeFields.regionPlaceholder' | t"
            autocomplete="address-level1"
            name="region"
            [ngModel]="region()"
            (ngModelChange)="region.set($event)"
          />
        }
      </div>
      @if (townOptions().length) {
        <bt-select-field
          [label]="'sellerOnboarding.locationLabel' | t"
          [options]="townOptions()"
          [placeholder]="'placeFields.townPick' | t"
          name="town"
          [ngModel]="otherTown() ? OTHER : location()"
          (ngModelChange)="pickTown($event)"
        />
      }
      @if (!townOptions().length || otherTown()) {
        <bt-text-field
          [label]="(townOptions().length ? 'placeFields.townOther' : 'sellerOnboarding.locationLabel') | t"
          [placeholder]="'sellerOnboarding.locationPlaceholder' | t"
          [hint]="'sellerOnboarding.locationHint' | t"
          autocomplete="address-level2"
          name="location"
          [ngModel]="location()"
          (ngModelChange)="location.set($event)"
        />
      }
      <!-- The prefix follows the country: a number typed without it is
           read as a national number there (the API does the same). -->
      <bt-text-field
        [label]="'sellerOnboarding.phoneLabel' | t"
        [hint]="'sellerOnboarding.phoneHint' | t"
        type="tel"
        inputmode="tel"
        autocomplete="tel-national"
        [prefix]="phonePrefix()"
        name="phone"
        [ngModel]="phone()"
        (ngModelChange)="phone.set($event)"
      />
    </div>
  `
})
export class PlaceFieldsComponent {
  private readonly locale = inject(LocaleService);

  readonly country = model('');
  readonly region = model('');
  readonly location = model('');
  readonly phone = model('');

  protected readonly OTHER = '__other__';

  protected readonly countryOptions = computed<SelectOption[]>(() => {
    const lang = this.locale.locale();
    const known = farmCountries();
    const codes = known.includes(this.country()) || !this.country() ? known : [this.country(), ...known];
    const [first, ...rest] = codes;
    return [first, ...[...rest].sort((a, b) => countryName(a, lang).localeCompare(countryName(b, lang), lang))].map((code) => ({
      value: code,
      label: countryName(code, lang)
    }));
  });

  /** The country's region list, or null when it's typed freely. */
  protected readonly regionOptions = computed<SelectOption[] | null>(() => {
    const regions = regionsOf(this.country());
    return regions ? regions.map((r) => ({ value: r.key, label: this.locale.t(r.labelKey) })) : null;
  });

  /** The region's listed towns, then "Autre ville"; none until a region
   * with a list is picked (the town is typed then). */
  private readonly towns = computed(() => (this.region() ? citiesOf(this.country(), this.region()) : []));
  protected readonly townOptions = computed<SelectOption[]>(() => {
    const towns = this.towns();
    if (!towns.length) return [];
    return [
      ...towns.map((c) => ({ value: c.key, label: this.locale.t(c.labelKey) })),
      { value: this.OTHER, label: this.locale.t('placeFields.townOtherOption') }
    ];
  });

  /** "Autre ville" is picked: a town the list doesn't have, typed below.
   * It follows the stored town (typed text means "other") and stays on
   * once picked while the farmer types. */
  protected readonly otherTown = linkedSignal<{ location: string; keys: string[] }, boolean>({
    source: () => ({ location: this.location(), keys: this.towns().map((c) => c.key) }),
    computation: ({ location, keys }, previous) => (location ? !keys.includes(location) : (previous?.value ?? false))
  });

  /** "+233" for Ghana; none once the number carries its own "+". */
  protected readonly phonePrefix = computed(() => {
    const code = callingCodeOf(this.country());
    return code && !this.phone().trim().startsWith('+') ? `+${code}` : undefined;
  });

  constructor() {
    // A farm that doesn't say yet starts in the platform's home country.
    effect(() => {
      if (!this.country()) this.country.set(defaultFarmCountry());
    });
    // A listed town says its region: fill it in when it's missing.
    effect(() => {
      const region = regionOfCity(this.location());
      if (region && !this.region()) this.region.set(region);
    });
  }

  protected setRegion(region: string): void {
    if (region === this.region()) return;
    this.region.set(region);
    // A listed town of another region no longer fits.
    if (regionOfCity(this.location())) this.location.set('');
    this.otherTown.set(false);
  }

  protected pickTown(value: string): void {
    if (value === this.OTHER) {
      this.otherTown.set(true);
      if (regionOfCity(this.location())) this.location.set('');
      return;
    }
    this.otherTown.set(false);
    this.location.set(value);
  }

  protected setCountry(code: string): void {
    if (code === this.country()) return;
    this.country.set(code);
    this.region.set('');
  }
}
