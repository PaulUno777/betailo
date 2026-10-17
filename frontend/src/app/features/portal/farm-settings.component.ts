import { ChangeDetectionStrategy, Component, computed, effect, inject, signal, untracked } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { switchMap } from 'rxjs';
import { Farm, FarmContact } from '../../domain/models';
import { FarmImage, FarmsApi } from '../../data-access/farms/farms.api';
import { PhotoUploadService } from '../../core/http/photo-upload.service';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { isSpecies } from '../../domain/reference/species';
import { SpeciesPickerComponent } from '../../shared/ui/species-picker/species-picker.component';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { PlaceFieldsComponent } from '../../shared/ui/place-fields/place-fields.component';
import { FarmCoverComponent } from '../../shared/ui/farm-cover/farm-cover.component';
import { ContactsFieldComponent, filledContacts } from '../../shared/ui/contacts-field/contacts-field.component';

/** `/portal/farm` — "Ma ferme": what buyers read about the farm (its
 * logo and banner, saved as soon as picked; name, town, the numbers
 * they can reach, what it raises) and, at the bottom,
 * deleting it. The API refuses a delete while the farm still has
 * cycles and says so. */
@Component({
  selector: 'bt-farm-settings',
  imports: [
    SpeciesPickerComponent,
    FormsModule,
    ButtonComponent,
    DialogComponent,
    TextFieldComponent,
    PlaceFieldsComponent,
    FarmCoverComponent,
    ContactsFieldComponent,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-2xl p-4 sm:p-8">
      <h1 class="text-heading-2">{{ 'farmSettings.title' | t }}</h1>

      @if (farm(); as f) {
        <bt-farm-cover
          class="mt-6 block"
          [name]="f.name"
          [logoUrl]="f.logoUrl"
          [bannerUrl]="f.bannerUrl"
          editable
          [busy]="imageBusy()"
          (picked)="uploadImage(f, $event.image, $event.file)"
          (removed)="removeImage(f, $event)"
        />
        <form class="mt-6 flex flex-col gap-5" (submit)="$event.preventDefault(); save(f)">
          <bt-text-field [label]="'sellerOnboarding.nameLabel' | t" name="name" [ngModel]="name()" (ngModelChange)="name.set($event)" />
          <bt-place-fields [(country)]="country" [(region)]="region" [(location)]="location" />
          <bt-contacts-field [(contacts)]="contacts" [country]="country()" [limit]="seller.contactLimit()" />
          <fieldset class="flex flex-col gap-3">
            <legend class="mb-1 text-small font-bold text-ink">{{ 'sellerOnboarding.speciesLabel' | t }}</legend>
            <p class="mb-2 text-small text-ink-muted">{{ 'sellerOnboarding.speciesHint' | t }}</p>
            <bt-species-picker [(value)]="species" />
          </fieldset>
          <bt-text-field
            [label]="'farmSettings.descriptionLabel' | t"
            multiline
            name="description"
            [ngModel]="description()"
            (ngModelChange)="description.set($event)"
          />
          <bt-button type="submit" size="lg" block [disabled]="!canSave()">
            {{ (saving() ? 'sellerOnboarding.saving' : 'farmSettings.save') | t }}
          </bt-button>
        </form>

        <section class="mt-12 rounded-lg border border-alert/30 p-4">
          <h2 class="text-heading-4 text-alert">{{ 'farmSettings.deleteTitle' | t }}</h2>
          <p class="mt-1 text-small text-ink-muted">{{ 'farmSettings.deleteBody' | t }}</p>
          <bt-button class="mt-3 inline-block" variant="danger" (click)="confirmDelete.set(true)">{{ 'farmSettings.delete' | t }}</bt-button>
        </section>
      } @else if (!seller.loaded()) {
        <div class="mt-6 h-64 animate-pulse rounded-lg bg-surface-sunken" aria-busy="true"></div>
      }
    </div>

    <bt-dialog
      [open]="confirmDelete()"
      tone="alert"
      [title]="'farmSettings.deleteConfirmTitle' | t"
      [primaryLabel]="'farmSettings.delete' | t"
      primaryVariant="danger"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="remove()"
      (secondaryClick)="confirmDelete.set(false)"
      (closed)="confirmDelete.set(false)"
    >
      {{ 'farmSettings.deleteBody' | t }}
    </bt-dialog>
  `
})
export class FarmSettingsComponent {
  private readonly farmsApi = inject(FarmsApi);
  protected readonly seller = inject(SellerSpaceService);
  private readonly toast = inject(ToastService);
  private readonly router = inject(Router);
  private readonly uploads = inject(PhotoUploadService);


  protected readonly farm = computed(() => this.seller.currentFarm() ?? undefined);

  protected readonly name = signal('');
  protected readonly location = signal('');
  protected readonly country = signal('');
  protected readonly region = signal('');
  protected readonly contacts = signal<FarmContact[]>([]);
  protected readonly imageBusy = signal<FarmImage | null>(null);
  protected readonly description = signal('');
  protected readonly species = signal<string[]>([]);
  protected readonly saving = signal(false);
  protected readonly confirmDelete = signal(false);

  protected readonly canSave = computed(() => !this.saving() && !!this.name().trim() && !!this.location().trim() && this.species().length > 0);

  /** The form loads once per farm: a refresh after an image change must
   * not wipe what the owner is typing. */
  private readonly farmId = computed(() => this.farm()?.id);

  constructor() {
    effect(() => {
      if (!this.farmId()) return;
      const f = untracked(() => this.farm());
      if (!f) return;
      untracked(() => {
        this.name.set(f.name);
        this.location.set(f.location);
        this.country.set(f.country ?? '');
        this.region.set(f.region ?? '');
        this.contacts.set(f.contacts ?? []);
        this.description.set(f.description ?? '');
        this.species.set((f.species ?? []).filter(isSpecies));
      });
    });
  }


  protected save(f: Farm): void {
    if (!this.canSave()) return;
    this.saving.set(true);
    this.farmsApi
      .update(f.id, {
        name: this.name().trim(),
        location: this.location().trim(),
        country: this.country() || undefined,
        region: this.region().trim() || undefined,
        species: this.species(),
        description: this.description().trim() || undefined,
        contacts: filledContacts(this.contacts())
      })
      .subscribe({
        next: () => {
          this.saving.set(false);
          this.toast.show({ tone: 'success', title: 'farmSettings.savedToast' });
          this.seller.refresh();
        },
        error: (err: AppError) => {
          this.saving.set(false);
          this.toast.show({ tone: 'danger', title: 'portal.farmSaveFailedTitle', detail: err.message });
        }
      });
  }

  protected uploadImage(f: Farm, image: FarmImage, file: File): void {
    if (this.imageBusy()) return;
    this.imageBusy.set(image);
    this.uploads
      .upload(file, (sizes) => this.farmsApi.presignImage(f.id, sizes), { thumb: true })
      .pipe(switchMap((key) => this.farmsApi.setImage(f.id, image, key)))
      .subscribe({
        next: () => this.imageSaved('farmCover.saved'),
        error: (err: AppError) => this.imageFailed(err)
      });
  }

  protected removeImage(f: Farm, image: FarmImage): void {
    if (this.imageBusy()) return;
    this.imageBusy.set(image);
    this.farmsApi.setImage(f.id, image, '').subscribe({
      next: () => this.imageSaved('farmCover.removed'),
      error: (err: AppError) => this.imageFailed(err)
    });
  }

  private imageSaved(title: string): void {
    this.imageBusy.set(null);
    this.toast.show({ tone: 'success', title });
    this.seller.refresh();
  }

  private imageFailed(err: AppError): void {
    this.imageBusy.set(null);
    this.toast.show({ tone: 'danger', title: 'farmCover.failed', detail: err.message });
  }

  protected remove(): void {
    const f = this.farm();
    this.confirmDelete.set(false);
    if (!f) return;
    this.farmsApi.delete(f.id).subscribe({
      next: () => {
        this.toast.show({ tone: 'success', title: 'farmSettings.deletedToast' });
        this.router.navigateByUrl('/portal');
      },
      error: (err: AppError) => this.toast.show({ tone: 'danger', title: 'portalCycles.actionFailedTitle', detail: err.message })
    });
  }
}
