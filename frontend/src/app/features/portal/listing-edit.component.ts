import { NgTemplateOutlet } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, effect, inject, signal, untracked } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { map, switchMap } from 'rxjs';
import { FarmListing } from '../../domain/models';
import { speciesLabelKey } from '../../domain/reference/species';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { ListingPhotosService } from '../../data-access/listings/listing-photos.service';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { PhotoPickerComponent, PickedPhoto, storedPhotos } from '../../shared/ui/photo-picker/photo-picker.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { StickyActionBarComponent } from '../../shared/ui/sticky-action-bar/sticky-action-bar.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';

interface EditForm {
  title: string;
  description: string;
  unitPrice: number | null;
  breed: string;
  delivery: boolean;
}

/** `/portal/listings/:id` — one listing, owner side: its photos (add,
 * remove, pick the cover), the words and price buyers see, and the
 * publish / unpublish switch. Read from the farm's listings (the owner
 * view, with photo keys), since that's the one read that carries them. */
@Component({
  selector: 'bt-listing-edit',
  imports: [
    FormsModule,
    NgTemplateOutlet,
    RouterLink,
    ButtonComponent,
    IconComponent,
    NoticeComponent,
    PhotoPickerComponent,
    StatusBadgeComponent,
    StickyActionBarComponent,
    TextFieldComponent,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-2xl px-4 pb-36 pt-4 sm:px-8 lg:pb-16 lg:pt-8">
      <a routerLink="/portal/listings" class="inline-flex h-11 items-center gap-1 text-body-strong text-forest">
        <bt-icon name="chevron-left" [size]="18" />
        {{ 'portalListings.title' | t }}
      </a>

      @if (rows.isLoading() && !row()) {
        <div class="mt-4 h-64 animate-pulse rounded-lg bg-surface-sunken" aria-busy="true"></div>
      } @else if (!row()) {
        <bt-notice class="mt-4 block" tone="warning">{{ 'listingEdit.notFound' | t }}</bt-notice>
      } @else {
        @let r = row()!;
        <div class="mt-2 flex items-start justify-between gap-3">
          <h1 class="text-heading-2">{{ 'listingEdit.title' | t }}</h1>
          <bt-status-badge kind="listing" [status]="r.listing.status" />
        </div>
        <p class="mt-1 text-small text-ink-muted">
          {{ speciesKey(r.cycle.species) | t }} · {{ r.cycle.reference || r.cycle.id.slice(-6) }} ·
          <a [routerLink]="['/portal/cycles', r.cycle.id]" class="font-semibold text-forest">{{ 'listingEdit.seeCycle' | t }}</a>
        </p>

        <form class="mt-6 flex flex-col gap-8" (submit)="$event.preventDefault(); save()">
          <section class="flex flex-col gap-3">
            <h2 class="text-heading-4">{{ 'sell.photosTitle' | t }}</h2>
            <bt-photo-picker [(photos)]="photos" [max]="12" />
          </section>

          <section class="flex flex-col gap-4">
            <bt-text-field [label]="'listingEdit.titleLabel' | t" name="title" [ngModel]="form().title" (ngModelChange)="patch({ title: $event })" />
            <bt-text-field
              [label]="'sell.priceLabel' | t"
              inputmode="numeric"
              suffix="FCFA"
              name="price"
              [ngModel]="priceText()"
              (ngModelChange)="setPrice($event)"
            />
            <bt-text-field [label]="'sell.breedLabel' | t" name="breed" [ngModel]="form().breed" (ngModelChange)="patch({ breed: $event })" />
            <label class="flex min-h-12 cursor-pointer items-center gap-3">
              <input
                type="checkbox"
                [checked]="form().delivery"
                (change)="patch({ delivery: !form().delivery })"
                class="h-6 w-6 shrink-0 cursor-pointer rounded accent-forest"
              />
              <span class="text-body">{{ 'sell.deliveryLabel' | t }}</span>
            </label>
            <bt-text-field
              [label]="'sell.descriptionLabel' | t"
              multiline
              name="description"
              [ngModel]="form().description"
              (ngModelChange)="patch({ description: $event })"
            />
          </section>

          <div class="hidden gap-3 lg:flex">
            <ng-container *ngTemplateOutlet="actions" />
          </div>
        </form>

        <bt-sticky-action-bar>
          <ng-container *ngTemplateOutlet="actions" />
        </bt-sticky-action-bar>

        <ng-template #actions>
          @if (r.listing.status === 'published') {
            <bt-button variant="outline" size="lg" [disabled]="busy()" (click)="unpublish()">{{ 'portalListings.unpublish' | t }}</bt-button>
          } @else {
            <bt-button variant="outline" size="lg" [disabled]="busy()" (click)="save(true)">{{ 'portalListings.publish' | t }}</bt-button>
          }
          <bt-button class="flex-1" size="lg" block [disabled]="busy()" (click)="save()">
            {{ (busy() ? 'listingEdit.saving' : 'listingEdit.save') | t }}
          </bt-button>
        </ng-template>
      }
    </div>
  `
})
export class ListingEditComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  protected readonly seller = inject(SellerSpaceService);
  private readonly listingsApi = inject(ListingsApi);
  private readonly listingPhotos = inject(ListingPhotosService);
  private readonly toast = inject(ToastService);

  private readonly id = toSignal(this.route.paramMap.pipe(map((p) => p.get('id') ?? '')), { initialValue: '' });
  protected readonly rows = apiResource<FarmListing[]>(() => {
    const farm = this.seller.currentFarm() ?? undefined;
    return farm ? this.listingsApi.farmListingsUrl(farm.id) : undefined;
  });
  protected readonly row = computed(() => this.rows.value()?.find((r) => r.listing.id === this.id()));

  protected readonly form = signal<EditForm>({ title: '', description: '', unitPrice: null, breed: '', delivery: false });
  protected readonly photos = signal<PickedPhoto[]>([]);
  protected readonly busy = signal(false);

  protected readonly priceText = computed(() => {
    const p = this.form().unitPrice;
    return p === null ? '' : new Intl.NumberFormat('fr-FR').format(p);
  });

  constructor() {
    // Fill the form once per listing loaded (and again after a save
    // reloads it), never while the farmer is typing.
    effect(() => {
      const r = this.row();
      if (!r) return;
      untracked(() => {
        const l = r.listing;
        this.form.set({
          title: l.title,
          description: l.description ?? '',
          unitPrice: l.unitPrice,
          breed: l.breed ?? '',
          delivery: !!l.deliveryAvailable
        });
        this.photos.set(storedPhotos(l.photos ?? [], r.photoKeys));
      });
    });
  }

  protected speciesKey(value: string): string {
    return speciesLabelKey(value);
  }

  protected patch(change: Partial<EditForm>): void {
    this.form.update((f) => ({ ...f, ...change }));
  }

  protected setPrice(text: string): void {
    const digits = text.replace(/\D/g, '').slice(0, 9);
    this.patch({ unitPrice: digits ? parseInt(digits, 10) : null });
  }

  /** Saves words, price and photos; with `publish`, then asks the API to
   * put the listing live (it says what's missing if it can't). */
  protected save(publish = false): void {
    const r = this.row();
    const f = this.form();
    if (!r || this.busy()) return;
    if (!f.title.trim() || !f.unitPrice) {
      this.toast.show({ tone: 'warning', title: 'listingEdit.missing' });
      return;
    }
    this.busy.set(true);
    this.listingsApi
      .update(r.listing.id, {
        title: f.title.trim(),
        description: f.description.trim() || undefined,
        unitPrice: f.unitPrice,
        unit: r.listing.unit,
        breed: f.breed.trim(),
        ageDays: r.listing.ageDays,
        deliveryAvailable: f.delivery
      })
      .pipe(
        switchMap(() => this.listingPhotos.save(r.listing.id, this.photos())),
        switchMap((l) => (publish ? this.listingsApi.publish(r.listing.id) : [l]))
      )
      .subscribe({
        next: () => {
          this.busy.set(false);
          this.toast.show({ tone: 'success', title: publish ? 'portalListings.publishedToast' : 'listingEdit.savedToast' });
          this.rows.reload();
        },
        error: (err: AppError) => {
          this.busy.set(false);
          this.rows.reload();
          this.toast.show({ tone: 'danger', title: 'portalListings.actionFailed', detail: err.message });
        }
      });
  }

  protected unpublish(): void {
    const r = this.row();
    if (!r || this.busy()) return;
    this.busy.set(true);
    this.listingsApi.unpublish(r.listing.id).subscribe({
      next: () => {
        this.busy.set(false);
        this.toast.show({ tone: 'success', title: 'portalListings.unpublishedToast' });
        this.router.navigateByUrl('/portal/listings');
      },
      error: (err: AppError) => {
        this.busy.set(false);
        this.toast.show({ tone: 'danger', title: 'portalListings.actionFailed', detail: err.message });
      }
    });
  }
}
