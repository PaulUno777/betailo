import { ChangeDetectionStrategy, Component, DestroyRef, computed, inject, output, signal } from '@angular/core';
import { Observable, catchError, concat, defer, map, of, switchMap, tap, toArray } from 'rxjs';
import { FormsModule } from '@angular/forms';
import { Farm, FarmContact } from '../../domain/models';
import { FarmImage, FarmInput, FarmsApi } from '../../data-access/farms/farms.api';
import { PhotoUploadService } from '../../core/http/photo-upload.service';
import { ToastService } from '../../core/notifications/toast.service';
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
import { FarmCoverComponent } from '../../shared/ui/farm-cover/farm-cover.component';
import { ContactsFieldComponent, filledContacts } from '../../shared/ui/contacts-field/contacts-field.component';

/** Opening a farm, in two steps: present it (its logo and banner, where
 * it is, the numbers buyers reach it on, what it raises), then submit the proof
 * staff validate it from (an identity document and photos of the farm).
 * Any account can do this, up to its farm limit. The second step can be
 * left for later: the farm then stays a draft whose cycles and listings
 * can be prepared but not published. Emits the farm once the owner is
 * done with both steps. */
@Component({
  selector: 'bt-seller-onboarding',
  imports: [
    SpeciesPickerComponent,
    FormsModule,
    ButtonComponent,
    IconComponent,
    NoticeComponent,
    TextFieldComponent,
    PlaceFieldsComponent,
    FarmProofFormComponent,
    FarmCoverComponent,
    ContactsFieldComponent,
    TranslatePipe
  ],
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

      <!-- Picked images wait here, previewed, until the farm exists:
           their storage keys are minted under its id. -->
      <bt-farm-cover [name]="name()" [logoUrl]="logo()?.url" [bannerUrl]="banner()?.url" editable (picked)="pickImage($event.image, $event.file)" (removed)="dropImage($event)" />

      <bt-text-field
        [label]="'sellerOnboarding.nameLabel' | t"
        [placeholder]="'sellerOnboarding.namePlaceholder' | t"
        autocomplete="organization"
        name="farmName"
        [ngModel]="name()"
        (ngModelChange)="name.set($event)"
      />
      <bt-place-fields [(country)]="country" [(region)]="region" [(location)]="location" />
      <bt-contacts-field [(contacts)]="contacts" [country]="country()" [limit]="seller.contactLimit()" />

      <fieldset class="flex flex-col gap-3">
        <legend class="mb-1 text-small font-bold text-ink">{{ 'sellerOnboarding.speciesLabel' | t }} <span aria-hidden="true" class="text-alert">*</span></legend>
        <p class="mb-2 text-small text-ink-muted">{{ 'sellerOnboarding.speciesHint' | t }}</p>
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
  private readonly uploads = inject(PhotoUploadService);
  private readonly toast = inject(ToastService);

  protected readonly steps = [{ key: 'sellerOnboarding.stepFarm' }, { key: 'sellerOnboarding.stepProof' }];
  /** Set once step 1 created the farm: step 2 works on it. */
  protected readonly farm = signal<Farm | null>(null);

  readonly created = output<Farm>();

  protected readonly perks = ['sellerOnboarding.perk1', 'sellerOnboarding.perk2', 'sellerOnboarding.perk3'];

  protected readonly name = signal('');
  protected readonly location = signal('');
  protected readonly country = signal('');
  protected readonly region = signal('');
  protected readonly contacts = signal<FarmContact[]>([]);
  /** Images picked before the farm exists, with a local preview. */
  protected readonly logo = signal<PickedImage | null>(null);
  protected readonly banner = signal<PickedImage | null>(null);
  protected readonly species = signal<string[]>([]);
  protected readonly saving = signal(false);
  protected readonly error = signal<string | null>(null);

  protected readonly canSubmit = computed(
    () => !this.saving() && !!this.name().trim() && !!this.location().trim() && this.species().length > 0
  );

  constructor() {
    inject(DestroyRef).onDestroy(() => {
      this.dropImage('logo');
      this.dropImage('banner');
    });
  }

  protected pickImage(image: FarmImage, file: File): void {
    this.dropImage(image);
    this.slot(image).set({ file, url: URL.createObjectURL(file) });
  }

  protected dropImage(image: FarmImage): void {
    const picked = this.slot(image)();
    if (picked) URL.revokeObjectURL(picked.url);
    this.slot(image).set(null);
  }

  private slot(image: FarmImage) {
    return image === 'logo' ? this.logo : this.banner;
  }

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
      contacts: filledContacts(this.contacts())
    };
    this.farmsApi
      .create(input)
      .pipe(switchMap((farm) => this.uploadImages(farm)))
      .subscribe({
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

  /** Uploads the picked logo and banner onto the new farm, one after
   * the other. The farm exists by now: an image that fails doesn't undo
   * it, the owner is told and can add the image from the settings. */
  private uploadImages(farm: Farm): Observable<Farm> {
    let failed = false;
    const steps = (['logo', 'banner'] as const).flatMap((image) => {
      const picked = this.slot(image)();
      if (!picked) return [];
      return [
        defer(() =>
          this.uploads
            .upload(picked.file, (sizes) => this.farmsApi.presignImage(farm.id, sizes), { thumb: true })
            .pipe(switchMap((key) => this.farmsApi.setImage(farm.id, image, key)))
        ).pipe(
          catchError(() => {
            failed = true;
            return of(null);
          })
        )
      ];
    });
    if (!steps.length) return of(farm);
    return concat(...steps).pipe(
      toArray(),
      tap(() => {
        if (failed) this.toast.show({ tone: 'warning', title: 'farmCover.failedAfterCreate' });
      }),
      map((saved) => saved.filter((f): f is Farm => !!f).at(-1) ?? farm)
    );
  }

  /** Step 2 is over, submitted or left for later. */
  protected finish(farm: Farm): void {
    this.seller.refresh();
    this.created.emit(farm);
  }
}

interface PickedImage {
  readonly file: File;
  /** An object URL for the preview; revoked once dropped. */
  readonly url: string;
}
