import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { Cycle, Listing } from '../../domain/models';
import { speciesLabelKey } from '../../domain/reference/species';
import { AdminApi } from '../../data-access/admin/admin.api';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { apiResource } from '../../core/http/api-resource';

/** `/admin/listings` — per `Admin.dc.html`: platform-wide cycles and
 * listings, read-only (no per-row admin action exists on either
 * endpoint beyond what escrow/reports already cover). */
@Component({
  selector: 'bt-admin-listings',
  imports: [StatusBadgeComponent, FcfaCurrencyPipe, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="p-6 sm:p-8">
      <h1 class="text-heading-2">{{ 'adminListings.title' | t }}</h1>

      <section class="mt-6">
        <h2 class="text-heading-4">{{ 'adminListings.cyclesTitle' | t }}</h2>
        @if (cycles.isLoading()) {
          <p class="mt-3 text-body text-ink-muted">{{ 'adminListings.loading' | t }}</p>
        } @else if ((cycles.value() ?? []).length === 0) {
          <p class="mt-3 text-body text-ink-muted">{{ 'adminListings.empty' | t }}</p>
        } @else {
          <div class="mt-3 flex flex-col gap-2">
            @for (c of cycles.value()!; track c.id) {
              <div class="flex flex-wrap items-center gap-4 rounded-md border border-line bg-surface-card p-3">
                <span class="min-w-[160px] flex-1 text-body-strong">{{ speciesKey(c.species) | t }} · {{ c.type }}</span>
                <bt-status-badge kind="cycle" [status]="c.status" />
                <span class="text-small text-ink-muted">{{ c.availableQuantity }} / {{ c.capacity }}</span>
              </div>
            }
          </div>
        }
      </section>

      <section class="mt-8">
        <h2 class="text-heading-4">{{ 'adminListings.listingsTitle' | t }}</h2>
        @if (listings.isLoading()) {
          <p class="mt-3 text-body text-ink-muted">{{ 'adminListings.loading' | t }}</p>
        } @else if ((listings.value() ?? []).length === 0) {
          <p class="mt-3 text-body text-ink-muted">{{ 'adminListings.empty' | t }}</p>
        } @else {
          <div class="mt-3 flex flex-col gap-2">
            @for (l of listings.value()!; track l.id) {
              <div class="flex flex-wrap items-center gap-4 rounded-md border border-line bg-surface-card p-3">
                <span class="min-w-[160px] flex-1 text-body-strong">{{ l.title }}</span>
                <bt-status-badge kind="listing" [status]="l.status" />
                <span class="text-small">{{ l.unitPrice | fcfa }}</span>
              </div>
            }
          </div>
        }
      </section>
    </div>
  `
})
export class AdminListingsComponent {
  protected speciesKey(value: string): string {
    return speciesLabelKey(value);
  }

  private readonly adminApi = inject(AdminApi);

  protected readonly cycles = apiResource<Cycle[]>(() => this.adminApi.cyclesUrl());
  protected readonly listings = apiResource<Listing[]>(() => this.adminApi.listingsUrl());
}
