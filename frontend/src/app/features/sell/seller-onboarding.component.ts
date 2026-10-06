import { ChangeDetectionStrategy, Component, computed, inject, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Observable, map, switchMap } from 'rxjs';
import { Farm } from '../../domain/models';
import { SPECIES_DOT, SPECIES_LABEL_KEYS, SPECIES_VALUES, SpeciesValue } from '../../domain/reference/species';
import { AccountApi } from '../../data-access/account/account.api';
import { FarmInput, FarmsApi } from '../../data-access/farms/farms.api';
import { AuthService } from '../../core/auth/auth.service';
import { AppError } from '../../core/http/app-error';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';

/** The one step between "I want to sell" and the sell wizard: present
 * your farm. A buyer becomes a seller (the API creates the farm and
 * switches the role together, then the session is refreshed so the new
 * role is in the token); a farmer account without a farm just creates
 * one. Emits the farm once it exists. */
@Component({
  selector: 'bt-seller-onboarding',
  imports: [FormsModule, ButtonComponent, IconComponent, NoticeComponent, TextFieldComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form class="flex flex-col gap-6" (submit)="$event.preventDefault(); submit()">
      <div class="flex flex-col items-start gap-3 rounded-xl bg-forest-deep p-5 text-on-forest-deep">
        <span class="flex h-12 w-12 items-center justify-center rounded-full bg-on-forest-deep/10">
          <bt-icon name="leaf" [size]="24" class="text-clay" />
        </span>
        <h2 class="text-heading-3">{{ (isBuyer() ? 'sellerOnboarding.titleBuyer' : 'sellerOnboarding.titleFarmer') | t }}</h2>
        <p class="text-body text-on-forest-deep/80">{{ 'sellerOnboarding.intro' | t }}</p>
        <ul class="flex flex-col gap-1.5 text-small text-on-forest-deep/85">
          @for (k of perks; track k) {
            <li class="flex items-center gap-2"><bt-icon name="check" [size]="16" class="text-clay" />{{ k | t }}</li>
          }
        </ul>
      </div>

      @if (error(); as message) {
        <bt-notice tone="danger">{{ message }}</bt-notice>
      }

      <bt-text-field
        [label]="'sellerOnboarding.nameLabel' | t"
        [placeholder]="'sellerOnboarding.namePlaceholder' | t"
        autocomplete="organization"
        name="farmName"
        [ngModel]="name()"
        (ngModelChange)="name.set($event)"
      />
      <bt-text-field
        [label]="'sellerOnboarding.locationLabel' | t"
        [placeholder]="'sellerOnboarding.locationPlaceholder' | t"
        [hint]="'sellerOnboarding.locationHint' | t"
        autocomplete="address-level2"
        name="location"
        [ngModel]="location()"
        (ngModelChange)="location.set($event)"
      />
      <bt-text-field
        [label]="'sellerOnboarding.phoneLabel' | t"
        [hint]="'sellerOnboarding.phoneHint' | t"
        type="tel"
        inputmode="tel"
        autocomplete="tel-national"
        prefix="+237"
        name="phone"
        [ngModel]="phone()"
        (ngModelChange)="phone.set($event)"
      />

      <fieldset class="flex flex-col gap-3">
        <legend class="mb-3 text-small font-bold text-ink">{{ 'sellerOnboarding.speciesLabel' | t }}</legend>
        <div class="flex flex-wrap gap-2">
          @for (s of speciesValues; track s) {
            <button
              type="button"
              (click)="toggle(s)"
              [attr.aria-pressed]="species().includes(s)"
              class="flex h-11 items-center gap-2 rounded-full border px-4 text-body-strong transition"
              [class]="species().includes(s) ? 'border-forest bg-forest-soft text-forest-deep' : 'border-line-strong bg-surface-card text-ink'"
            >
              <span class="h-2.5 w-2.5 rounded-full" [class]="dots[s]"></span>
              {{ speciesLabels[s] | t }}
            </button>
          }
        </div>
      </fieldset>

      <bt-button type="submit" size="lg" block [disabled]="!canSubmit()">
        {{ (saving() ? 'sellerOnboarding.saving' : 'sellerOnboarding.submit') | t }}
      </bt-button>
    </form>
  `
})
export class SellerOnboardingComponent {
  private readonly auth = inject(AuthService);
  private readonly accountApi = inject(AccountApi);
  private readonly farmsApi = inject(FarmsApi);

  readonly created = output<Farm>();

  protected readonly speciesValues = SPECIES_VALUES;
  protected readonly speciesLabels = SPECIES_LABEL_KEYS;
  protected readonly dots = SPECIES_DOT;
  protected readonly perks = ['sellerOnboarding.perk1', 'sellerOnboarding.perk2', 'sellerOnboarding.perk3'];

  protected readonly isBuyer = computed(() => this.auth.role() !== 'farmer');
  protected readonly name = signal('');
  protected readonly location = signal('');
  protected readonly phone = signal('');
  protected readonly species = signal<SpeciesValue[]>([]);
  protected readonly saving = signal(false);
  protected readonly error = signal<string | null>(null);

  protected readonly canSubmit = computed(() => !this.saving() && !!this.name().trim() && !!this.location().trim());

  protected toggle(s: SpeciesValue): void {
    this.species.update((list) => (list.includes(s) ? list.filter((x) => x !== s) : [...list, s]));
  }

  protected submit(): void {
    if (!this.canSubmit()) return;
    this.saving.set(true);
    this.error.set(null);
    const input: FarmInput = {
      name: this.name().trim(),
      location: this.location().trim(),
      species: this.species(),
      contactPhone: this.phone().trim() || undefined
    };
    // A buyer's new role only reaches the API's checks once a new token
    // carries it, so the session is reloaded before going on.
    const request: Observable<Farm> = this.isBuyer()
      ? this.accountApi.becomeSeller(input).pipe(switchMap((farm) => this.auth.reloadSession().pipe(map(() => farm))))
      : this.farmsApi.create(input);
    request.subscribe({
      next: (farm) => {
        this.saving.set(false);
        this.created.emit(farm);
      },
      error: (err: AppError) => {
        this.saving.set(false);
        this.error.set(err.message);
      }
    });
  }
}
