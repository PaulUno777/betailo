import { cycleTypeLabelKey } from '../../domain/reference/cycle-types';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { Cycle, Listing } from '../../domain/models';
import { speciesLabelKey } from '../../domain/reference/species';
import { AdminApi } from '../../data-access/admin/admin.api';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { apiResource } from '../../core/http/api-resource';
import { AdminPageHeaderComponent } from './ui/admin-page-header.component';

/** `/admin/listings` — per `Admin.dc.html`: platform-wide cycles and
 * listings, read-only (no per-row admin action exists on either
 * endpoint beyond what escrow/reports already cover). */
@Component({
  selector: 'bt-admin-listings',
  imports: [AdminPageHeaderComponent, StatusBadgeComponent, ButtonComponent, FcfaCurrencyPipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-[22px] p-4 sm:p-7 lg:px-9">
      <bt-admin-page-header [eyebrow]="'adminNav.group.steer' | t" [title]="'adminListings.title' | t" [subtitle]="'adminListings.subtitle' | t" />

      <section class="bo-card px-2 pt-[18px] pb-2">
        <h2 class="px-3.5 pb-3 font-display text-[19px] font-bold">{{ 'adminListings.cyclesTitle' | t }}</h2>
        @if (cycles.isLoading() && !cycles.hasValue()) {
          <div class="mx-3.5 mb-3 h-40 animate-pulse rounded-xl bg-surface-sunken" aria-hidden="true"></div>
        } @else if (cycles.error()) {
          <div class="flex flex-col items-start gap-3 px-3.5 pb-4" role="alert">
            <p class="text-body">{{ 'apiErrors.server' | t }}</p>
            <bt-button variant="outline" (click)="cycles.reload()">{{ 'common.retry' | t }}</bt-button>
          </div>
        } @else if ((cycles.value() ?? []).length === 0) {
          <p class="px-3.5 pb-5 text-body text-ink-muted">{{ 'adminListings.emptyCycles' | t }}</p>
        } @else {
          <div class="overflow-x-auto">
            <table class="bo-table min-w-[640px]">
              <thead>
                <tr>
                  <th class="bo-th" scope="col">{{ 'adminListings.colCycle' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminListings.colType' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminListings.colStatus' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminListings.colFill' | t }}</th>
                </tr>
              </thead>
              <tbody>
                @for (c of cycles.value()!; track c.id) {
                  <tr>
                    <td class="bo-td">
                      <div class="flex flex-col">
                        <b>{{ speciesKey(c.species) | t }}</b>
                        <span class="font-mono text-[11.5px] text-ink-muted">{{ c.reference || c.id.slice(-6) }}</span>
                      </div>
                    </td>
                    <td class="bo-td">{{ typeKey(c.type) | t }}</td>
                    <td class="bo-td"><bt-status-badge kind="cycle" [status]="c.status" /></td>
                    <td class="bo-td font-mono !text-[13px] tabular-nums">{{ c.capacity - c.availableQuantity }} / {{ c.capacity }}</td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
        }
      </section>

      <section class="bo-card px-2 pt-[18px] pb-2">
        <h2 class="px-3.5 pb-3 font-display text-[19px] font-bold">{{ 'adminListings.listingsTitle' | t }}</h2>
        @if (listings.isLoading() && !listings.hasValue()) {
          <div class="mx-3.5 mb-3 h-40 animate-pulse rounded-xl bg-surface-sunken" aria-hidden="true"></div>
        } @else if (listings.error()) {
          <div class="flex flex-col items-start gap-3 px-3.5 pb-4" role="alert">
            <p class="text-body">{{ 'apiErrors.server' | t }}</p>
            <bt-button variant="outline" (click)="listings.reload()">{{ 'common.retry' | t }}</bt-button>
          </div>
        } @else if ((listings.value() ?? []).length === 0) {
          <p class="px-3.5 pb-5 text-body text-ink-muted">{{ 'adminListings.empty' | t }}</p>
        } @else {
          <div class="overflow-x-auto">
            <table class="bo-table min-w-[560px]">
              <thead>
                <tr>
                  <th class="bo-th" scope="col">{{ 'adminListings.colListing' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminListings.colStatus' | t }}</th>
                  <th class="bo-th" scope="col">{{ 'adminListings.colPrice' | t }}</th>
                </tr>
              </thead>
              <tbody>
                @for (l of listings.value()!; track l.id) {
                  <tr>
                    <td class="bo-td wrap-anywhere"><b>{{ l.title }}</b></td>
                    <td class="bo-td"><bt-status-badge kind="listing" [status]="l.status" /></td>
                    <td class="bo-td font-mono !text-[13px] whitespace-nowrap tabular-nums">{{ l.unitPrice | fcfa }}</td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
        }
      </section>
    </div>
  `
})
export class AdminListingsComponent {
  protected typeKey(type: string): string {
    return cycleTypeLabelKey(type);
  }

  protected speciesKey(value: string): string {
    return speciesLabelKey(value);
  }

  private readonly adminApi = inject(AdminApi);

  protected readonly cycles = apiResource<Cycle[]>(() => this.adminApi.cyclesUrl());
  protected readonly listings = apiResource<Listing[]>(() => this.adminApi.listingsUrl());
}
