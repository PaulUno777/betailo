import { ChangeDetectionStrategy, Component, computed, inject, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Farm } from '../../domain/models';
import { FarmInput, FarmsApi } from '../../data-access/farms/farms.api';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { AppError } from '../../core/http/app-error';
import { SpeciesPickerComponent } from '../../shared/ui/species-picker/species-picker.component';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { PlaceFieldsComponent } from '../../shared/ui/place-fields/place-fields.component';
import { FarmProofFormComponent } from './farm-proof-form.component';

/** Opening a farm, in two steps: present it, then submit the proof
 * staff validate it from (an identity document and photos of the farm).
 * Any account can do this, up to its farm limit. The second step can be
 * left for later: the farm then stays a draft whose cycles and listings
 * can be prepared but not published. Emits the farm once the owner is
 * done with both steps. */
@Component({
  selector: 'bt-seller-onboarding',
  imports: [SpeciesPickerComponent, FormsModule, ButtonComponent, IconComponent, NoticeComponent, TextFieldComponent, PlaceFieldsComponent, FarmProofFormComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <ol class="mb-5 grid grid-cols-2 gap-2" [attr.aria-label]="'sellerOnboarding.stepsLabel' | t">
      @for (s of steps; track s.key; let i = $index) {
        <li class="flex flex-col gap-1.5" [attr.aria-current]="(farm() ? 1 : 0) === i ? 'step' : null">
          <span class="h-1.5 rounded-full" [class]="i <= (farm() ? 1 : 0) ? 'bg-forest' : 'bg-line'"></span>
          <span class="text-caption" [class]="(farm() ? 1 : 0) === i ? 'text-ink' : 'text-ink-muted'">{{ i + 1 }}. {{ s.key | t }}</span>
        </li>
      }
    </ol>
    @if (farm(); as created) {
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <h2 class="text-heading-3">{{ 'farmProof.title' | t: { farm: created.name } }}</h2>
          <p class="text-body text-ink-muted">{{ 'farmProof.intro' | t }}</p>
        </div>
        <bt-farm-proof-form [farm]="created" [skippable]="true" (submitted)="finish($event)" (skipped)="finish(created)" />
      </div>
    } @else {
    <form class="flex flex-col gap-6" (submit)="$event.preventDefault(); submit()">
      <div class="flex flex-col items-start gap-3 rounded-xl bg-forest-deep p-5 text-on-forest-deep">
        <span class="flex h-12 w-12 items-center justify-center rounded-full bg-on-forest-deep/10">
          <bt-icon name="leaf" [size]="24" class="text-clay" />
        </span>
        <h2 class="text-heading-3">{{ (seller.hasFarm() ? 'sellerOnboarding.titleAnother' : 'sellerOnboarding.titleBuyer') | t }}</h2>
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
      <bt-place-fields [(country)]="country" [(region)]="region" [(location)]="location" [(phone)]="phone" />

      <fieldset class="flex flex-col gap-3">
        <legend class="mb-3 text-small font-bold text-ink">{{ 'sellerOnboarding.speciesLabel' | t }}</legend>
          <bt-species-picker [(value)]="species" />
      </fieldset>

      <bt-button type="submit" size="lg" block [disabled]="!canSubmit()">
        {{ (saving() ? 'sellerOnboarding.saving' : 'sellerOnboarding.submit') | t }}
      </bt-button>
    </form>
    }
  `
})
export class SellerOnboardingComponent {
  protected readonly seller = inject(SellerSpaceService);
  private readonly farmsApi = inject(FarmsApi);

  protected readonly steps = [{ key: 'sellerOnboarding.stepFarm' }, { key: 'sellerOnboarding.stepProof' }];
  /** Set once step 1 created the farm: step 2 works on it. */
  protected readonly farm = signal<Farm | null>(null);

  readonly created = output<Farm>();

  protected readonly perks = ['sellerOnboarding.perk1', 'sellerOnboarding.perk2', 'sellerOnboarding.perk3'];

  protected readonly name = signal('');
  protected readonly location = signal('');
  protected readonly country = signal('');
  protected readonly region = signal('');
  protected readonly phone = signal('');
  protected readonly species = signal<string[]>([]);
  protected readonly saving = signal(false);
  protected readonly error = signal<string | null>(null);

  protected readonly canSubmit = computed(() => !this.saving() && !!this.name().trim() && !!this.location().trim());


  protected submit(): void {
    if (!this.canSubmit()) return;
    this.saving.set(true);
    this.error.set(null);
    const input: FarmInput = {
      name: this.name().trim(),
      location: this.location().trim(),
      country: this.country() || undefined,
      region: this.region().trim() || undefined,
      species: this.species(),
      contactPhone: this.phone().trim() || undefined
    };
    this.farmsApi.create(input).subscribe({
      next: (farm) => {
        this.saving.set(false);
        this.farm.set(farm);
        // The portal and the menus now have a farm to show.
        this.seller.select(farm.id);
        this.seller.refresh();
      },
      error: (err: AppError) => {
        this.saving.set(false);
        this.error.set(err.message);
      }
    });
  }

  /** Step 2 is over, submitted or left for later. */
  protected finish(farm: Farm): void {
    this.seller.refresh();
    this.created.emit(farm);
  }
}
