import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { Observable } from 'rxjs';
import { FarmListing } from '../../domain/models';
import { speciesLabelKey } from '../../domain/reference/species';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';

/** `/portal/listings` — "Mes annonces": every listing of the farm, any
 * status, as one card each (cover, price, what's left, reservations),
 * with the one action that matters for its status: publish a draft,
 * take a live one off the market. Editing opens the listing's page. */
@Component({
  selector: 'bt-portal-listings',
  imports: [RouterLink, ButtonComponent, IconComponent, StatusBadgeComponent, FcfaCurrencyPipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-4xl p-4 sm:p-8">
      <div class="flex items-center justify-between gap-4">
        <h1 class="text-heading-2">{{ 'portalListings.title' | t }}</h1>
        <bt-button variant="solid" href="/sell">
          <bt-icon name="plus" [size]="18" />
          {{ 'portalListings.new' | t }}
        </bt-button>
      </div>

      @if (rows.isLoading() && !rows.hasValue()) {
        <div class="mt-6 flex flex-col gap-3" aria-busy="true">
          @for (i of [1, 2, 3]; track i) {
            <div class="h-28 animate-pulse rounded-lg bg-surface-sunken"></div>
          }
        </div>
      } @else if (list().length === 0) {
        <div class="mt-10 flex flex-col items-center gap-4 rounded-xl border border-dashed border-line-strong p-8 text-center">
          <bt-icon name="camera" [size]="32" class="text-ink-muted" />
          <p class="text-body text-ink-muted">{{ 'portalListings.empty' | t }}</p>
          <bt-button size="lg" href="/sell">{{ 'portalListings.emptyCta' | t }}</bt-button>
        </div>
      } @else {
        <ul class="mt-6 flex flex-col gap-3">
          @for (row of list(); track row.listing.id) {
            <li class="flex gap-3 rounded-lg border border-line bg-surface-card p-3">
              <a [routerLink]="['/portal/listings', row.listing.id]" class="h-24 w-24 shrink-0 overflow-hidden rounded-md bg-surface-sunken">
                @if (row.listing.photos?.[0]; as cover) {
                  <img [src]="cover" alt="" loading="lazy" class="h-full w-full object-cover" />
                } @else {
                  <span class="flex h-full w-full items-center justify-center text-ink-muted"><bt-icon name="camera" [size]="22" /></span>
                }
              </a>
              <div class="flex min-w-0 flex-1 flex-col gap-1">
                <div class="flex items-start justify-between gap-2">
                  <a [routerLink]="['/portal/listings', row.listing.id]" class="line-clamp-2 text-body-strong">{{ row.listing.title }}</a>
                  <bt-status-badge kind="listing" [status]="row.listing.status" />
                </div>
                <span class="text-small text-ink-muted">
                  {{ speciesKey(row.cycle.species) | t }} · {{ 'portalListings.left' | t: { count: row.cycle.availableQuantity } }}
                  @if (row.activeReservations) {
                    · {{ 'portalListings.reservations' | t: { count: row.activeReservations } }}
                  }
                </span>
                <div class="mt-auto flex flex-wrap items-center justify-between gap-2">
                  <strong class="text-heading-4">{{ row.listing.unitPrice | fcfa }}</strong>
                  <div class="flex gap-2">
                    <bt-button variant="ghost" size="sm" [href]="'/portal/listings/' + row.listing.id">{{ 'portalListings.edit' | t }}</bt-button>
                    @if (row.listing.status === 'published') {
                      <bt-button variant="outline" size="sm" [disabled]="acting() === row.listing.id" (click)="unpublish(row)">
                        {{ 'portalListings.unpublish' | t }}
                      </bt-button>
                    } @else {
                      <bt-button size="sm" [disabled]="acting() === row.listing.id" (click)="publish(row)">
                        {{ 'portalListings.publish' | t }}
                      </bt-button>
                    }
                  </div>
                </div>
              </div>
            </li>
          }
        </ul>
      }
    </div>
  `
})
export class PortalListingsComponent {
  protected readonly seller = inject(SellerSpaceService);
  private readonly listingsApi = inject(ListingsApi);
  private readonly toast = inject(ToastService);

  protected readonly rows = apiResource<FarmListing[]>(() => {
    const farm = this.seller.currentFarm() ?? undefined;
    return farm ? this.listingsApi.farmListingsUrl(farm.id) : undefined;
  });
  protected readonly list = computed(() => this.rows.value() ?? []);
  protected readonly acting = signal<string | null>(null);

  protected speciesKey(value: string): string {
    return speciesLabelKey(value);
  }

  /** The API checks what publishing needs (a photo, an open cycle) and
   * says what's missing; that message is shown as is. */
  protected publish(row: FarmListing): void {
    this.act(row, this.listingsApi.publish(row.listing.id), 'portalListings.publishedToast');
  }

  protected unpublish(row: FarmListing): void {
    this.act(row, this.listingsApi.unpublish(row.listing.id), 'portalListings.unpublishedToast');
  }

  private act(row: FarmListing, request: Observable<unknown>, doneKey: string): void {
    this.acting.set(row.listing.id);
    request.subscribe({
      next: () => {
        this.acting.set(null);
        this.toast.show({ tone: 'success', title: doneKey });
        this.rows.reload();
      },
      error: (err: AppError) => {
        this.acting.set(null);
        if (err.code === 'farm.not_approved') {
          this.toast.show({ tone: 'info', title: 'portalListings.notApprovedTitle', detail: err.message });
        } else {
          this.toast.show({ tone: 'danger', title: 'portalListings.actionFailed', detail: err.message });
        }
      }
    });
  }
}
