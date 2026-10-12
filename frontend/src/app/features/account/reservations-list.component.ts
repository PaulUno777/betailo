import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map, Observable } from 'rxjs';
import { PaymentMethod } from '../../domain/enums';
import { ReservationAction, ReservationGroup, ReservationList, ReservationView } from '../../domain/models';
import { PAYMENT_METHOD_LABEL_KEYS, PAYMENT_METHOD_VALUES } from '../../domain/rules/payment-method';
import { ReservationsApi } from '../../data-access/reservations/reservations.api';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { BottomSheetComponent } from '../../shared/ui/bottom-sheet/bottom-sheet.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { ChoiceCardComponent } from '../../shared/ui/choice-card/choice-card.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { ReservationCardComponent } from './reservation-card.component';

const TABS: readonly { id: ReservationGroup; labelKey: string }[] = [
  { id: 'active', labelKey: 'reservations.tabActive' },
  { id: 'done', labelKey: 'reservations.tabCompleted' },
  { id: 'cancelled', labelKey: 'reservations.tabCancelled' }
];

type Sheet = { kind: 'pay' | 'dispute' | 'confirm' | 'cancel'; view: ReservationView } | null;

/** "Mes réservations", per the mobile design: tabs with the API's
 * counts, the money held in escrow, and one card per reservation. What
 * each card can do comes from the API (`actions`); the sheets here only
 * collect what an action needs (a payment number, a dispute reason, a
 * rating) and call it. */
@Component({
  selector: 'bt-reservations-list',
  imports: [
    FormsModule,
    ReservationCardComponent,
    BottomSheetComponent,
    ButtonComponent,
    ChoiceCardComponent,
    IconComponent,
    TextFieldComponent,
    FcfaCurrencyPipe,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-3xl px-4 pb-8 pt-5 sm:px-8 lg:py-10">
      <h1 class="text-heading-1">{{ 'reservations.title' | t }}</h1>

      <div class="mt-4 grid grid-cols-3 gap-1 rounded-full bg-surface-sunken p-1" role="tablist">
        @for (tab of tabs; track tab.id) {
          <button
            type="button"
            role="tab"
            [attr.aria-selected]="group() === tab.id"
            (click)="group.set(tab.id)"
            class="h-11 rounded-full text-small font-semibold transition"
            [class]="group() === tab.id ? 'bg-surface-card text-ink shadow-sm' : 'text-ink-muted'"
          >
            {{ tab.labelKey | t }} · {{ list.value()?.counts?.[tab.id] ?? 0 }}
          </button>
        }
      </div>

      @if ((list.value()?.escrowTotal ?? 0) > 0) {
        <div class="mt-4 flex items-center gap-3 rounded-xl bg-forest-deep px-5 py-4 text-on-forest-deep">
          <bt-icon name="lock" [size]="22" class="text-clay" />
          <div class="flex flex-col">
            <span class="text-heading-3">{{ list.value()!.escrowTotal | fcfa }}</span>
            <span class="text-small text-on-forest-deep/75">{{ 'reservations.escrowBanner' | t }}</span>
          </div>
        </div>
      }

      <div class="mt-5 flex flex-col gap-3" aria-live="polite">
        @if (list.value(); as l) {
          @for (v of l.items; track v.id) {
            <bt-reservation-card
              [view]="v"
              [highlighted]="v.id === highlight()"
              [whatsappMessage]="'reservations.whatsappMessage' | t: { ref: v.reference, title: v.listing.title }"
              (action)="onAction($event, v)"
            />
          } @empty {
            <div class="flex flex-col items-center gap-3 rounded-xl border border-dashed border-line-strong px-6 py-12 text-center">
              <bt-icon name="calendar" [size]="30" class="text-ink-muted" />
              <p class="text-body text-ink-muted">{{ 'reservations.empty' | t }}</p>
              @if (group() === 'active') {
                <bt-button href="/marketplace">{{ 'favorites.emptyCta' | t }}</bt-button>
              }
            </div>
          }
        } @else if (list.isLoading()) {
          @for (i of [1, 2]; track i) {
            <div class="h-44 animate-pulse rounded-xl bg-surface-sunken"></div>
          }
        } @else if (list.error()) {
          <div class="flex flex-col items-center gap-3 py-10 text-center">
            <p class="text-body">{{ 'reservations.loadError' | t }}</p>
            <bt-button variant="outline" (click)="list.reload()">{{ 'common.retry' | t }}</bt-button>
          </div>
        }
      </div>
    </div>

    <!-- Pay a reserved-but-unpaid reservation. -->
    <bt-bottom-sheet [open]="sheet()?.kind === 'pay'" [title]="'reservations.paySheetTitle' | t" (closed)="close()">
      <div class="flex flex-col gap-3" role="radiogroup">
        @for (m of methods; track m) {
          <bt-choice-card name="pay-method" [value]="m" [title]="methodLabels[m] | t" [checked]="payMethod() === m" (selected)="payMethod.set(m)" />
        }
        @if (payMethod() !== 'card') {
          <bt-text-field
            [label]="'checkout.momoNumberLabel' | t"
            type="tel"
            inputmode="tel"
            autocomplete="tel-national"
            prefix="+237"
            [ngModel]="payPhone()"
            (ngModelChange)="payPhone.set($event)"
          />
        }
      </div>
      <div sheetFooter class="border-t border-line px-5 pt-3" style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))">
        <bt-button block size="lg" [disabled]="busy() || (payMethod() !== 'card' && !payPhone().trim())" (click)="pay()">
          {{ 'checkout.payAmount' | t: { amount: (sheet()?.view?.totalAmount ?? 0 | fcfa) } }}
        </bt-button>
      </div>
    </bt-bottom-sheet>

    <!-- Confirm reception: releases the escrow, so ask once. -->
    <bt-bottom-sheet [open]="sheet()?.kind === 'confirm'" [title]="'reservations.confirmSheetTitle' | t" (closed)="close()">
      <p class="text-body text-ink-muted">{{ 'reservations.confirmSheetBody' | t: { amount: (sheet()?.view?.totalAmount ?? 0 | fcfa) } }}</p>
      <div sheetFooter class="flex gap-3 border-t border-line px-5 pt-3" style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))">
        <bt-button variant="outline" (click)="close()">{{ 'common.cancel' | t }}</bt-button>
        <bt-button class="flex-1" variant="solid" block [disabled]="busy()" (click)="confirm()">{{ 'reservations.action.confirm' | t }}</bt-button>
      </div>
    </bt-bottom-sheet>

    <bt-bottom-sheet [open]="sheet()?.kind === 'cancel'" [title]="'reservations.cancelSheetTitle' | t" (closed)="close()">
      <p class="text-body text-ink-muted">{{ 'reservations.cancelSheetBody' | t }}</p>
      <div sheetFooter class="flex gap-3 border-t border-line px-5 pt-3" style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))">
        <bt-button variant="outline" (click)="close()">{{ 'common.back' | t }}</bt-button>
        <bt-button class="flex-1" variant="danger" block [disabled]="busy()" (click)="cancel()">{{ 'reservations.action.cancel' | t }}</bt-button>
      </div>
    </bt-bottom-sheet>

    <bt-bottom-sheet [open]="sheet()?.kind === 'dispute'" [title]="'reservations.disputeTitle' | t" (closed)="close()">
      <div class="flex flex-col gap-3">
        <p class="text-small text-ink-muted">{{ 'reservations.disputeIntro' | t }}</p>
        <bt-text-field [label]="'reservations.disputeReasonLabel' | t" multiline [ngModel]="reason()" (ngModelChange)="reason.set($event)" />
      </div>
      <div sheetFooter class="border-t border-line px-5 pt-3" style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))">
        <bt-button block size="lg" variant="danger" [disabled]="busy() || !reason().trim()" (click)="dispute()">
          {{ 'reservations.disputeSubmit' | t }}
        </bt-button>
      </div>
    </bt-bottom-sheet>

  `
})
export class ReservationsListComponent {
  private readonly reservationsApi = inject(ReservationsApi);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  protected readonly tabs = TABS;
  protected readonly methods = PAYMENT_METHOD_VALUES;
  protected readonly methodLabels = PAYMENT_METHOD_LABEL_KEYS;

  protected readonly group = signal<ReservationGroup>('active');
  protected readonly highlight = toSignal(this.route.queryParamMap.pipe(map((p) => p.get('highlight'))), { initialValue: null });
  protected readonly list = apiResource<ReservationList>(() => this.reservationsApi.mineUrl(this.group()));

  protected readonly sheet = signal<Sheet>(null);
  protected readonly busy = signal(false);
  protected readonly payMethod = signal<PaymentMethod>('mtn_momo');
  protected readonly payPhone = signal('');
  protected readonly reason = signal('');

  private readonly current = computed(() => this.sheet()?.view);

  protected onAction(action: ReservationAction, view: ReservationView): void {
    if (action === 'review') {
      this.router.navigate(['/account/reservations', view.id, 'review']);
      return;
    }
    if (action === 'pay' || action === 'dispute' || action === 'confirm' || action === 'cancel') {
      this.reason.set('');
      this.sheet.set({ kind: action, view });
    }
  }

  protected close(): void {
    this.sheet.set(null);
  }

  protected pay(): void {
    const v = this.current();
    if (!v) return;
    const phone = this.payMethod() === 'card' ? undefined : this.payPhone();
    this.run(this.reservationsApi.pay(v.id, this.payMethod(), phone), 'reservations.paidToast');
  }

  protected confirm(): void {
    const v = this.current();
    if (v) this.run(this.reservationsApi.confirmDelivery(v.id), 'reservations.confirmedToast');
  }

  protected cancel(): void {
    const v = this.current();
    if (v) this.run(this.reservationsApi.cancel(v.id), 'reservations.cancelledToast');
  }

  protected dispute(): void {
    const v = this.current();
    if (v) this.run(this.reservationsApi.dispute(v.id, this.reason().trim()), 'reservations.disputedToast');
  }

  /** Every action: call it, close the sheet, refresh the list (the API
   * recomputes steps, actions and counts), and say what happened. */
  private run(request: Observable<unknown>, successTitle: string): void {
    this.busy.set(true);
    request.subscribe({
      next: () => {
        this.busy.set(false);
        this.close();
        this.list.reload();
        this.toast.show({ tone: 'success', title: successTitle });
      },
      error: (err: AppError) => {
        this.busy.set(false);
        this.toast.show({ tone: 'danger', title: 'reservations.actionFailed', detail: err.message });
      }
    });
  }
}
