import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ListingSearchResult } from '../../domain/models';
import { ListingsApi } from '../../data-access/listings/listings.api';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { ListingCardComponent } from '../../shared/ui/listing-card/listing-card.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';

/** The buyer's saved listings, most recently saved first (the API's
 * order). A listing that's been unpublished since simply isn't returned. */
@Component({
  selector: 'bt-favorites',
  imports: [ListingCardComponent, ButtonComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-content-max px-4 pb-8 pt-5 sm:px-8 lg:px-18 lg:py-10">
      <h1 class="text-heading-1">{{ 'favorites.title' | t }}</h1>

      @if (favorites.value(); as res) {
        @if (res.items.length) {
          <p class="mt-1 text-small text-ink-muted">{{ 'favorites.count' | t: { count: res.items.length } }}</p>
          <div class="mt-5 flex flex-col gap-3 lg:hidden">
            @for (item of res.items; track item.listing.id) {
              <bt-listing-card [item]="item" layout="row" />
            }
          </div>
          <div class="mt-8 hidden gap-5 lg:grid lg:grid-cols-3 xl:grid-cols-4">
            @for (item of res.items; track item.listing.id) {
              <bt-listing-card [item]="item" />
            }
          </div>
        } @else {
          <div class="mt-10 flex flex-col items-center gap-4 rounded-xl border border-dashed border-line-strong px-6 py-14 text-center">
            <span class="flex h-14 w-14 items-center justify-center rounded-full bg-clay-soft text-clay">
              <bt-icon name="heart" [size]="26" />
            </span>
            <p class="text-heading-4">{{ 'favorites.emptyTitle' | t }}</p>
            <p class="max-w-sm text-body text-ink-muted">{{ 'favorites.emptyBody' | t }}</p>
            <bt-button href="/marketplace">{{ 'favorites.emptyCta' | t }}</bt-button>
          </div>
        }
      } @else if (favorites.isLoading()) {
        <div class="mt-5 flex flex-col gap-3" aria-busy="true">
          @for (i of [1, 2, 3]; track i) {
            <div class="h-[122px] animate-pulse rounded-lg bg-surface-sunken"></div>
          }
        </div>
      } @else if (favorites.error()) {
        <div class="mt-10 flex flex-col items-center gap-4 text-center">
          <p class="text-body">{{ 'favorites.error' | t }}</p>
          <bt-button variant="outline" (click)="favorites.reload()">{{ 'common.retry' | t }}</bt-button>
        </div>
      }
    </div>
  `
})
export class FavoritesComponent {
  private readonly listingsApi = inject(ListingsApi);
  protected readonly favorites = apiResource<ListingSearchResult>(() => this.listingsApi.favoritesUrl());
}
