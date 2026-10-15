import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { CyclesApi } from '../../data-access/cycles/cycles.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { SelectFieldComponent } from '../../shared/ui/select-field/select-field.component';
import { LocaleService } from '../../core/i18n/locale.service';
import { speciesCycleType, speciesGroups, speciesLabelKey } from '../../domain/reference/species';
import { cycleTypeLabelKey } from '../../domain/reference/cycle-types';
import { ButtonComponent } from '../../shared/ui/button/button.component';

/** `/portal/cycles/new` — per `CycleForm.dc.html`, simplified to the
 * fields `POST /cycles` actually accepts (species, type, capacity,
 * dates) — `minPerReservation`/`maxPerReservation` exist in the domain
 * but aren't in the create request body, so the backend just applies
 * its "no bound" default rather than this form inventing a value for a
 * field it can't send. */
@Component({
  selector: 'bt-cycle-form',
  imports: [FormsModule, SelectFieldComponent, TextFieldComponent, ButtonComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-2xl p-6 sm:p-8">
      <h1 class="text-heading-1">{{ 'cycleForm.title' | t }}</h1>
      <div class="mt-8 flex flex-col gap-8">
        <div class="grid grid-cols-[44px_1fr] gap-5">
          <span class="flex h-11 w-11 items-center justify-center rounded-full bg-forest text-body-strong text-on-forest">1</span>
          <div class="flex flex-col gap-4">
            <h2 class="text-heading-4">{{ 'cycleForm.sectionSpeciesTitle' | t }}</h2>
            <bt-select-field
              [label]="'cycleForm.speciesLabel' | t"
              [placeholder]="'cycleForm.speciesPlaceholder' | t"
              [options]="speciesOptions()"
              [ngModel]="species()"
              (ngModelChange)="species.set($event)"
            />
            @if (type()) {
              <!-- The API raises a species in the cycle type the catalog gives it. -->
              <p class="text-small text-ink-muted">{{ 'cycleForm.typeFromSpecies' | t: { type: (typeLabel(type()) | t) } }}</p>
            }
          </div>
        </div>
        <div class="grid grid-cols-[44px_1fr] gap-5">
          <span class="flex h-11 w-11 items-center justify-center rounded-full bg-clay text-body-strong text-on-clay">2</span>
          <div class="flex flex-col gap-4">
            <h2 class="text-heading-4">{{ 'cycleForm.sectionScheduleTitle' | t }}</h2>
            <div class="grid gap-4 sm:grid-cols-2">
              <bt-text-field [label]="'cycleForm.startsAtLabel' | t" type="text" [ngModel]="startsAt()" (ngModelChange)="startsAt.set($event)" hint="AAAA-MM-JJ" />
              <bt-text-field [label]="'cycleForm.endsAtLabel' | t" type="text" [ngModel]="endsAt()" (ngModelChange)="endsAt.set($event)" hint="AAAA-MM-JJ" />
            </div>
          </div>
        </div>
        <div class="grid grid-cols-[44px_1fr] gap-5">
          <span class="flex h-11 w-11 items-center justify-center rounded-full border border-line-strong text-body-strong text-ink-muted">3</span>
          <div class="flex flex-col gap-4">
            <h2 class="text-heading-4">{{ 'cycleForm.sectionCapacityTitle' | t }}</h2>
            <bt-text-field [label]="'cycleForm.capacityLabel' | t" type="number" [ngModel]="capacity()" (ngModelChange)="capacity.set($event)" />
          </div>
        </div>
        <div class="flex gap-3 border-t border-line pt-6">
          <bt-button variant="outline" (click)="cancel()">{{ 'cycleForm.cancelButton' | t }}</bt-button>
          <bt-button [disabled]="submitting() || !canSubmit()" (click)="submit()">
            {{ (submitting() ? 'cycleForm.submitting' : 'cycleForm.submitButton') | t }}
          </bt-button>
        </div>
      </div>
    </div>
  `
})
export class CycleFormComponent {
  protected readonly seller = inject(SellerSpaceService);
  private readonly cyclesApi = inject(CyclesApi);
  private readonly router = inject(Router);
  private readonly toast = inject(ToastService);

  private readonly myFarm = computed(() => this.seller.currentFarm() ?? undefined);

  private readonly locale = inject(LocaleService);
  protected readonly species = signal('');
  protected readonly type = computed(() => speciesCycleType(this.species()));
  protected readonly typeLabel = cycleTypeLabelKey;
  /** Every species open to sale, named with its category. */
  protected readonly speciesOptions = computed(() =>
    speciesGroups().flatMap((g) =>
      g.species.map((s) => ({ value: s, label: `${this.locale.t(g.labelKey)} · ${this.locale.t(speciesLabelKey(s))}` }))
    )
  );
  protected readonly capacity = signal('');
  protected readonly startsAt = signal('');
  protected readonly endsAt = signal('');
  protected readonly submitting = signal(false);

  protected readonly canSubmit = computed(
    () => !!this.myFarm() && !!this.species() && !!this.type() && Number(this.capacity()) > 0 && !!this.startsAt()
  );

  protected submit(): void {
    const farmId = this.myFarm()?.id;
    if (this.submitting() || !this.canSubmit() || !farmId) return;
    this.submitting.set(true);
    this.cyclesApi
      .create({
        farmId,
        species: this.species(),
        type: this.type(),
        capacity: Number(this.capacity()),
        startsAt: new Date(this.startsAt()).toISOString(),
        endsAt: this.endsAt() ? new Date(this.endsAt()).toISOString() : undefined
      })
      .subscribe({
        next: () => this.router.navigateByUrl('/portal/cycles'),
        error: (err: AppError) => {
          this.submitting.set(false);
          this.toast.show({ tone: 'danger', title: 'cycleForm.submitFailedTitle', detail: err.message });
        }
      });
  }

  protected cancel(): void {
    this.router.navigateByUrl('/portal/cycles');
  }
}
