import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { HttpClient, httpResource } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { forkJoin, map, switchMap } from 'rxjs';
import { Cycle, Farm } from '../../domain/models';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { CyclesApi } from '../../data-access/cycles/cycles.api';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { SelectFieldComponent } from '../../shared/ui/select-field/select-field.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';

/** `/portal/listings/new` — per `ListingForm.dc.html`, including the
 * presign → PUT-to-storage → register-photos flow the plan called out
 * as the one non-trivial write path. Cycle choice comes pre-filled when
 * reached from a cycle row's "Publier une annonce" action. */
@Component({
  selector: 'bt-listing-form',
  imports: [FormsModule, TextFieldComponent, SelectFieldComponent, ButtonComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-lg p-6 sm:p-8">
      <h1 class="text-heading-2">{{ 'listingForm.title' | t }}</h1>

      @if (cycles.isLoading()) {
        <p class="mt-6 text-body text-ink-muted">{{ 'listingForm.cycleLoading' | t }}</p>
      } @else if ((cycles.value() ?? []).length === 0) {
        <p class="mt-6 text-body text-ink-muted">{{ 'listingForm.noCycles' | t }}</p>
      } @else {
        <div class="mt-6 flex flex-col gap-4">
          <bt-select-field
            [label]="'listingForm.cycleLabel' | t"
            [options]="cycleOptions()"
            [ngModel]="cycleId()"
            (ngModelChange)="cycleId.set($event)"
          />
          <bt-text-field [label]="'listingForm.titleLabel' | t" [ngModel]="title()" (ngModelChange)="title.set($event)" />
          <bt-text-field
            [label]="'listingForm.descriptionLabel' | t"
            multiline
            [ngModel]="description()"
            (ngModelChange)="description.set($event)"
          />
          <bt-text-field
            [label]="'listingForm.unitPriceLabel' | t"
            type="number"
            [ngModel]="unitPrice()"
            (ngModelChange)="unitPrice.set($event)"
          />
          <bt-text-field [label]="'listingForm.unitLabel' | t" [ngModel]="unit()" (ngModelChange)="unit.set($event)" />

          <div class="flex flex-col gap-2">
            <span class="text-small text-ink-muted">{{ 'listingForm.photosLabel' | t }}</span>
            <label
              class="flex cursor-pointer flex-col items-center gap-2 rounded-lg border border-dashed border-line-strong bg-surface-card px-4 py-6 text-center hover:border-forest"
            >
              <input type="file" accept="image/*" multiple (change)="onFilesSelected($event)" class="hidden" />
              <span class="text-small text-ink-muted">
                @if (photoFiles().length > 0) {
                  {{ photoFiles().length }} · {{ photoNames() }}
                } @else {
                  {{ 'listingForm.addPhotoButton' | t }}
                }
              </span>
            </label>
          </div>

          <bt-button [disabled]="submitting() || !canSubmit()" class="mt-2" (click)="submit()">
            {{ (submitting() ? (uploading() ? 'listingForm.uploadingPhotos' : 'listingForm.submitting') : 'listingForm.submitButton') | t }}
          </bt-button>
        </div>
      }
    </div>
  `
})
export class ListingFormComponent {
  private readonly http = inject(HttpClient);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly farmsApi = inject(FarmsApi);
  private readonly cyclesApi = inject(CyclesApi);
  private readonly listingsApi = inject(ListingsApi);
  private readonly toast = inject(ToastService);

  private readonly initialCycleId = toSignal(this.route.queryParamMap.pipe(map((p) => p.get('cycleId') ?? '')), {
    initialValue: ''
  });

  private readonly farms = httpResource<Farm[]>(() => this.farmsApi.mineUrl());
  private readonly myFarm = computed(() => this.farms.value()?.[0]);

  protected readonly cycles = httpResource<Cycle[]>(() => {
    const farmId = this.myFarm()?.id;
    return farmId ? this.cyclesApi.byFarmUrl(farmId) : undefined;
  });

  protected readonly cycleOptions = computed(() =>
    (this.cycles.value() ?? []).map((c) => ({ value: c.id, label: `${c.species} · ${c.type} (${c.id.slice(-6)})` }))
  );

  protected readonly cycleId = signal(this.initialCycleId());
  protected readonly title = signal('');
  protected readonly description = signal('');
  protected readonly unitPrice = signal('');
  protected readonly unit = signal('');
  protected readonly photoFiles = signal<File[]>([]);
  protected readonly submitting = signal(false);
  protected readonly uploading = signal(false);

  protected readonly canSubmit = computed(() => !!this.cycleId() && !!this.title() && Number(this.unitPrice()) > 0);

  protected photoNames(): string {
    return this.photoFiles()
      .map((f) => f.name)
      .join(', ');
  }

  protected onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.photoFiles.set(input.files ? Array.from(input.files) : []);
  }

  protected submit(): void {
    if (this.submitting() || !this.canSubmit()) return;
    this.submitting.set(true);
    this.listingsApi
      .create({
        cycleId: this.cycleId(),
        title: this.title(),
        description: this.description() || undefined,
        unitPrice: Number(this.unitPrice()),
        unit: this.unit() || undefined
      })
      .pipe(
        switchMap((listing) => {
          if (this.photoFiles().length === 0) {
            return [listing];
          }
          this.uploading.set(true);
          const uploads = this.photoFiles().map((file) =>
            this.listingsApi.presignPhoto(listing.id).pipe(
              switchMap((presigned) =>
                this.http.put(presigned.uploadUrl, file, { headers: { 'Content-Type': file.type } }).pipe(map(() => presigned.key))
              )
            )
          );
          return forkJoin(uploads).pipe(switchMap((keys) => this.listingsApi.setPhotos(listing.id, keys)));
        })
      )
      .subscribe({
        next: () => {
          this.submitting.set(false);
          this.uploading.set(false);
          this.toast.show({ tone: 'success', title: 'listingForm.submitSuccessTitle', detail: 'listingForm.submitSuccessDetail' });
          this.router.navigateByUrl('/portal/cycles');
        },
        error: (err: AppError) => {
          this.submitting.set(false);
          this.uploading.set(false);
          this.toast.show({ tone: 'danger', title: 'listingForm.submitFailedTitle', detail: err.message });
        }
      });
  }
}
