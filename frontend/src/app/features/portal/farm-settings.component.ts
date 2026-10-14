import { ChangeDetectionStrategy, Component, computed, effect, inject, signal, untracked } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { Farm } from '../../domain/models';
import { SPECIES_DOT, SPECIES_LABEL_KEYS, SPECIES_VALUES, SpeciesValue, isSpecies } from '../../domain/reference/species';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { PlaceFieldsComponent } from '../../shared/ui/place-fields/place-fields.component';

/** `/portal/farm` — "Ma ferme": what buyers read about the farm (name,
 * town, the number they can call, what it raises) and, at the bottom,
 * deleting it. The API refuses a delete while the farm still has
 * cycles and says so. */
@Component({
  selector: 'bt-farm-settings',
  imports: [FormsModule, ButtonComponent, DialogComponent, TextFieldComponent, PlaceFieldsComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-2xl p-4 sm:p-8">
      <h1 class="text-heading-2">{{ 'farmSettings.title' | t }}</h1>

      @if (farm(); as f) {
        <form class="mt-6 flex flex-col gap-5" (submit)="$event.preventDefault(); save(f)">
          <bt-text-field [label]="'sellerOnboarding.nameLabel' | t" name="name" [ngModel]="name()" (ngModelChange)="name.set($event)" />
          <bt-place-fields [(country)]="country" [(region)]="region" [(location)]="location" [(phone)]="phone" />
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

  protected readonly speciesValues = SPECIES_VALUES;
  protected readonly speciesLabels = SPECIES_LABEL_KEYS;
  protected readonly dots = SPECIES_DOT;

  protected readonly farm = computed(() => this.seller.currentFarm() ?? undefined);

  protected readonly name = signal('');
  protected readonly location = signal('');
  protected readonly country = signal('');
  protected readonly region = signal('');
  protected readonly phone = signal('');
  protected readonly description = signal('');
  protected readonly species = signal<SpeciesValue[]>([]);
  protected readonly saving = signal(false);
  protected readonly confirmDelete = signal(false);

  protected readonly canSave = computed(() => !this.saving() && !!this.name().trim() && !!this.location().trim());

  constructor() {
    effect(() => {
      const f = this.farm();
      if (!f) return;
      untracked(() => {
        this.name.set(f.name);
        this.location.set(f.location);
        this.country.set(f.country ?? '');
        this.region.set(f.region ?? '');
        this.phone.set(f.contactPhone ?? '');
        this.description.set(f.description ?? '');
        this.species.set((f.species ?? []).filter(isSpecies));
      });
    });
  }

  protected toggle(s: SpeciesValue): void {
    this.species.update((list) => (list.includes(s) ? list.filter((x) => x !== s) : [...list, s]));
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
        contactPhone: this.phone().trim() || undefined
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
