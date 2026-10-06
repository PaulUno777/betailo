import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { Cycle, Farm } from '../../domain/models';
import { speciesLabelKey } from '../../domain/reference/species';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { CyclesApi } from '../../data-access/cycles/cycles.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { CapacityBarComponent } from '../../shared/ui/capacity-bar/capacity-bar.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { apiResource } from '../../core/http/api-resource';

/** `/portal/cycles` — per `Portal.dc.html`'s cycle table, its own page:
 * list + delete, backed by `GET/DELETE /cycles`. Status changes live on
 * each cycle's page, where the API lists the ones allowed next. */
@Component({
  selector: 'bt-portal-cycles-list',
  imports: [RouterLink, StatusBadgeComponent, CapacityBarComponent, ButtonComponent, DialogComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="p-6 sm:p-8">
      <div class="flex items-center justify-between gap-4">
        <h1 class="text-heading-2">{{ 'portalCycles.title' | t }}</h1>
        <bt-button variant="solid" href="/portal/cycles/new">{{ 'portalCycles.newCycleButton' | t }}</bt-button>
      </div>

      @if (cycles.isLoading()) {
        <p class="mt-6 text-body text-ink-muted">{{ 'portalCycles.loading' | t }}</p>
      } @else if ((cycles.value() ?? []).length === 0) {
        <p class="mt-6 text-body text-ink-muted">{{ 'portalCycles.empty' | t }}</p>
      } @else {
        <div class="mt-6 flex flex-col gap-3">
          @for (c of cycles.value()!; track c.id) {
            <div class="flex flex-wrap items-center gap-4 rounded-lg border border-line bg-surface-card p-4">
              <a [routerLink]="['/portal/cycles', c.id]" class="flex min-w-[160px] flex-col">
                <span class="text-body-strong">{{ speciesKey(c.species) | t }} · {{ c.type }}</span>
                <span class="text-caption text-ink-muted">{{ c.reference || c.id.slice(-6) }}</span>
              </a>
              <bt-status-badge kind="cycle" [status]="c.status" />
              <div class="min-w-[160px] flex-1">
                <bt-capacity-bar [capacity]="c.capacity" [availableQuantity]="c.availableQuantity" />
              </div>
              <div class="flex flex-wrap gap-2">
                <bt-button variant="ghost" size="sm" href="/portal/listings/new" [queryParams]="{ cycleId: c.id }">
                  {{ 'portalCycles.publishListingButton' | t }}
                </bt-button>
                <bt-button variant="outline" size="sm" [href]="'/portal/cycles/' + c.id">{{ 'portalCycles.open' | t }}</bt-button>
                <bt-button variant="ghost" size="sm" (click)="confirmDeleteId.set(c.id)">{{ 'portalCycles.deleteButton' | t }}</bt-button>
              </div>
            </div>
          }
        </div>
      }
    </div>

    <bt-dialog
      [open]="confirmDeleteId() !== null"
      tone="alert"
      [title]="'portalCycles.deleteConfirmTitle' | t"
      [primaryLabel]="'portalCycles.deleteConfirmAction' | t"
      primaryVariant="danger"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="deleteCycle()"
      (secondaryClick)="confirmDeleteId.set(null)"
      (closed)="confirmDeleteId.set(null)"
    >
      {{ 'portalCycles.deleteConfirmBody' | t }}
    </bt-dialog>
  `
})
export class PortalCyclesListComponent {
  private readonly farmsApi = inject(FarmsApi);
  private readonly cyclesApi = inject(CyclesApi);
  private readonly toast = inject(ToastService);

  protected readonly farms = apiResource<Farm[]>(() => this.farmsApi.mineUrl());
  protected readonly myFarm = computed(() => this.farms.value()?.[0]);

  protected readonly cycles = apiResource<Cycle[]>(() => {
    const farmId = this.myFarm()?.id;
    return farmId ? this.cyclesApi.byFarmUrl(farmId) : undefined;
  });

  protected readonly confirmDeleteId = signal<string | null>(null);

  protected speciesKey(value: string): string {
    return speciesLabelKey(value);
  }

  protected deleteCycle(): void {
    const id = this.confirmDeleteId();
    if (!id) return;
    this.cyclesApi.delete(id).subscribe({
      next: () => {
        this.confirmDeleteId.set(null);
        this.cycles.reload();
      },
      error: (err: AppError) => {
        this.confirmDeleteId.set(null);
        this.toast.show({ tone: 'danger', title: 'portalCycles.actionFailedTitle', detail: err.message });
      }
    });
  }
}
