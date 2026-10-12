import { cycleTypeLabelKey } from '../../domain/reference/cycle-types';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { Observable, map } from 'rxjs';
import { CycleDetail } from '../../domain/models';
import { CycleStatus } from '../../domain/enums';
import { speciesLabelKey } from '../../domain/reference/species';
import { CyclesApi } from '../../data-access/cycles/cycles.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { BottomSheetComponent } from '../../shared/ui/bottom-sheet/bottom-sheet.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { CapacityBarComponent } from '../../shared/ui/capacity-bar/capacity-bar.component';
import { CycleTimelineComponent } from '../../shared/ui/cycle-timeline/cycle-timeline.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { StatusBadgeComponent } from '../../shared/ui/status-badge/status-badge.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';

type Sheet = 'health' | 'edit' | null;

const STATUS_ACTION_KEYS: Readonly<Record<CycleStatus, string>> = {
  planned: 'cycleDetail.action.planned',
  in_progress: 'cycleDetail.action.in_progress',
  done: 'cycleDetail.action.done',
  cancelled: 'cycleDetail.action.cancelled'
};

function today(): string {
  return new Date().toISOString().slice(0, 10);
}

/** `/portal/cycles/:id` — one batch of animals: how full it is, its
 * carnet de santé as a timeline, "Ajouter un relevé" (the thing a farmer
 * does most, so it's the big button), and the status changes the API
 * says are possible next. Capacity and dates are edited in a sheet. */
@Component({
  selector: 'bt-cycle-detail',
  imports: [
    FormsModule,
    RouterLink,
    BottomSheetComponent,
    ButtonComponent,
    CapacityBarComponent,
    CycleTimelineComponent,
    IconComponent,
    NoticeComponent,
    StatusBadgeComponent,
    TextFieldComponent,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-2xl p-4 sm:p-8">
      <a routerLink="/portal/cycles" class="inline-flex h-11 items-center gap-1 text-body-strong text-forest">
        <bt-icon name="chevron-left" [size]="18" />
        {{ 'portalCycles.title' | t }}
      </a>

      @if (cycle.isLoading() && !cycle.hasValue()) {
        <div class="mt-4 h-64 animate-pulse rounded-lg bg-surface-sunken" aria-busy="true"></div>
      } @else if (cycle.value(); as c) {
        <div class="mt-2 flex items-start justify-between gap-3">
          <div class="flex flex-col">
            <h1 class="text-heading-2">{{ speciesKey(c.species) | t }}</h1>
            <span class="text-small text-ink-muted">{{ c.reference || c.id.slice(-6) }} · {{ typeKey(c.type) | t }}</span>
          </div>
          <bt-status-badge kind="cycle" [status]="c.status" />
        </div>

        <div class="mt-4 rounded-lg border border-line bg-surface-card p-4">
          <bt-capacity-bar [capacity]="c.capacity" [availableQuantity]="c.availableQuantity" />
        </div>

        <div class="mt-4 grid grid-cols-2 gap-2">
          <bt-button size="lg" block (click)="openHealth()">
            <bt-icon name="plus" [size]="18" />
            {{ 'cycleDetail.addHealth' | t }}
          </bt-button>
          <bt-button variant="outline" size="lg" block (click)="openEdit(c)">{{ 'cycleDetail.edit' | t }}</bt-button>
        </div>

        <section class="mt-8">
          <h2 class="text-heading-4">{{ 'cycleDetail.timelineTitle' | t }}</h2>
          <div class="mt-4">
            <bt-cycle-timeline [steps]="c.timeline" [cancelled]="c.status === 'cancelled'" />
          </div>
        </section>

        @if (c.nextStatuses.length) {
          <section class="mt-8 flex flex-col gap-2">
            <h2 class="text-heading-4">{{ 'cycleDetail.statusTitle' | t }}</h2>
            <div class="flex flex-wrap gap-2">
              @for (s of c.nextStatuses; track s) {
                <bt-button [variant]="s === 'cancelled' ? 'ghost' : 'outline'" [disabled]="busy()" (click)="setStatus(s)">
                  {{ statusKeys[s] | t }}
                </bt-button>
              }
            </div>
          </section>
        }
      } @else {
        <bt-notice class="mt-4 block" tone="warning">{{ 'cycleDetail.notFound' | t }}</bt-notice>
      }
    </div>

    <bt-bottom-sheet [open]="sheet() === 'health'" [title]="'cycleDetail.addHealth' | t" (closed)="sheet.set(null)">
      <form class="flex flex-col gap-4" (submit)="$event.preventDefault(); saveHealth()">
        <bt-text-field
          [label]="'cycleDetail.healthTitleLabel' | t"
          [placeholder]="'cycleDetail.healthTitlePlaceholder' | t"
          name="title"
          [ngModel]="healthTitle()"
          (ngModelChange)="healthTitle.set($event)"
        />
        <bt-text-field
          [label]="'cycleDetail.healthDetailLabel' | t"
          multiline
          name="detail"
          [ngModel]="healthDetail()"
          (ngModelChange)="healthDetail.set($event)"
        />
        <label class="flex flex-col gap-2">
          <span class="text-small font-bold text-ink">{{ 'cycleDetail.dateLabel' | t }}</span>
          <input
            type="date"
            [max]="todayValue"
            [value]="healthDate()"
            (change)="healthDate.set($any($event.target).value)"
            class="h-14 rounded-md border border-line-strong bg-surface-card px-4 text-[16px] text-ink outline-none focus:border-forest"
          />
        </label>
        <bt-button type="submit" size="lg" block [disabled]="busy() || !healthTitle().trim()">{{ 'cycleDetail.save' | t }}</bt-button>
      </form>
    </bt-bottom-sheet>

    <bt-bottom-sheet [open]="sheet() === 'edit'" [title]="'cycleDetail.edit' | t" (closed)="sheet.set(null)">
      <form class="flex flex-col gap-4" (submit)="$event.preventDefault(); saveEdit()">
        <bt-text-field
          [label]="'cycleDetail.capacityLabel' | t"
          [hint]="'cycleDetail.capacityHint' | t"
          inputmode="numeric"
          name="capacity"
          [ngModel]="'' + capacity()"
          (ngModelChange)="capacity.set(toInt($event))"
        />
        <div class="grid grid-cols-2 gap-3">
          <bt-text-field
            [label]="'cycleDetail.minLabel' | t"
            inputmode="numeric"
            name="min"
            [ngModel]="'' + minPer()"
            (ngModelChange)="minPer.set(toInt($event))"
          />
          <bt-text-field
            [label]="'cycleDetail.maxLabel' | t"
            inputmode="numeric"
            name="max"
            [ngModel]="'' + maxPer()"
            (ngModelChange)="maxPer.set(toInt($event))"
          />
        </div>
        <label class="flex flex-col gap-2">
          <span class="text-small font-bold text-ink">{{ 'cycleDetail.startsLabel' | t }}</span>
          <input
            type="date"
            [value]="startsAt()"
            (change)="startsAt.set($any($event.target).value)"
            class="h-14 rounded-md border border-line-strong bg-surface-card px-4 text-[16px] text-ink outline-none focus:border-forest"
          />
        </label>
        <bt-button type="submit" size="lg" block [disabled]="busy() || !startsAt() || capacity() < 1">{{ 'cycleDetail.save' | t }}</bt-button>
      </form>
    </bt-bottom-sheet>
  `
})
export class CycleDetailComponent {
  protected typeKey(type: string): string {
    return cycleTypeLabelKey(type);
  }

  private readonly route = inject(ActivatedRoute);
  private readonly cyclesApi = inject(CyclesApi);
  private readonly toast = inject(ToastService);

  private readonly id = toSignal(this.route.paramMap.pipe(map((p) => p.get('id') ?? '')), { initialValue: '' });
  protected readonly cycle = apiResource<CycleDetail>(() => (this.id() ? this.cyclesApi.detailUrl(this.id()) : undefined));

  protected readonly statusKeys = STATUS_ACTION_KEYS;
  protected readonly todayValue = today();
  protected readonly sheet = signal<Sheet>(null);
  protected readonly busy = signal(false);

  protected readonly healthTitle = signal('');
  protected readonly healthDetail = signal('');
  protected readonly healthDate = signal(today());

  protected readonly capacity = signal(0);
  protected readonly minPer = signal(0);
  protected readonly maxPer = signal(0);
  protected readonly startsAt = signal('');
  private readonly endsAt = signal<string | undefined>(undefined);

  protected readonly current = computed(() => this.cycle.value());

  protected speciesKey(value: string): string {
    return speciesLabelKey(value);
  }

  protected toInt(text: string): number {
    const digits = String(text).replace(/\D/g, '').slice(0, 7);
    return digits ? parseInt(digits, 10) : 0;
  }

  protected openHealth(): void {
    this.healthTitle.set('');
    this.healthDetail.set('');
    this.healthDate.set(today());
    this.sheet.set('health');
  }

  protected openEdit(c: CycleDetail): void {
    this.capacity.set(c.capacity);
    this.minPer.set(c.minPerReservation ?? 0);
    this.maxPer.set(c.maxPerReservation ?? 0);
    this.startsAt.set(c.startsAt.slice(0, 10));
    this.endsAt.set(c.endsAt);
    this.sheet.set('edit');
  }

  protected saveHealth(): void {
    const c = this.current();
    if (!c || !this.healthTitle().trim()) return;
    this.run(
      this.cyclesApi.addHealthEvent(c.id, {
        title: this.healthTitle().trim(),
        detail: this.healthDetail().trim() || undefined,
        date: new Date(this.healthDate() + 'T12:00:00').toISOString()
      }),
      'cycleDetail.healthSaved'
    );
  }

  /** The API refuses a capacity below what's already reserved and says
   * so; that message is what the farmer sees. */
  protected saveEdit(): void {
    const c = this.current();
    if (!c) return;
    this.run(
      this.cyclesApi.update(c.id, {
        startsAt: new Date(this.startsAt() + 'T00:00:00').toISOString(),
        endsAt: this.endsAt(),
        capacity: this.capacity(),
        minPerReservation: this.minPer(),
        maxPerReservation: this.maxPer()
      }),
      'cycleDetail.saved'
    );
  }

  protected setStatus(status: CycleStatus): void {
    const c = this.current();
    if (!c) return;
    this.run(this.cyclesApi.updateStatus(c.id, status), 'cycleDetail.saved');
  }

  private run(request: Observable<unknown>, doneKey: string): void {
    if (this.busy()) return;
    this.busy.set(true);
    request.subscribe({
      next: () => {
        this.busy.set(false);
        this.sheet.set(null);
        this.toast.show({ tone: 'success', title: doneKey });
        this.cycle.reload();
      },
      error: (err: AppError) => {
        this.busy.set(false);
        this.toast.show({ tone: 'danger', title: 'portalCycles.actionFailedTitle', detail: err.message });
      }
    });
  }
}
