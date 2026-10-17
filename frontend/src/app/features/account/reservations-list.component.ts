import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map, Observable } from 'rxjs';
import { PaymentMethod } from '../../domain/enums';
import { AppNotification, NotificationList, ReservationAction, ReservationGroup, ReservationList, ReservationView } from '../../domain/models';
import { PAYMENT_METHOD_LABEL_KEYS, PAYMENT_METHOD_VALUES } from '../../domain/rules/payment-method';
import { NotificationsApi } from '../../data-access/notifications/notifications.api';
import { ReservationsApi } from '../../data-access/reservations/reservations.api';
import { SellerSpaceService } from '../../core/auth/seller-space.service';
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
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';
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
    RouterLink,
    LocalDatePipe,
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
    <div class="mx-auto grid max-w-content-max gap-8 px-4 pb-8 pt-5 sm:px-8 lg:grid-cols-[minmax(0,1fr)_360px] lg:px-[72px] lg:py-10">
     <div class="min-w-0">
      <div class="flex flex-wrap items-end justify-between gap-x-6 gap-y-4">
        <h1 class="font-display text-[30px] leading-none font-extrabold tracking-[-0.03em] lg:text-[44px]">{{ 'reservations.title' | t }}</h1>
        <div class="grid w-full grid-cols-3 gap-1 rounded-full bg-surface-sunken p-1 lg:w-auto" role="tablist">
          @for (tab of tabs; track tab.id) {
            <button
              type="button"
              role="tab"
              [attr.aria-selected]="group() === tab.id"
              (click)="group.set(tab.id)"
              class="h-11 rounded-full px-4 text-small font-semibold whitespace-nowrap transition lg:h-10"
              [class]="group() === tab.id ? 'bg-surface-card text-ink shadow-sm' : 'text-ink-muted'"
            >
              {{ tab.labelKey | t }} · {{ list.value()?.counts?.[tab.id] ?? 0 }}
            </button>
          }
        </div>
      </div>

      @if ((list.value()?.escrowTotal ?? 0) > 0) {
        <div class="panel-dark mt-4 flex items-center gap-3 rounded-xl bg-forest-deep px-5 py-4 text-on-forest-deep lg:hidden">
          <bt-icon name="lock" [size]="22" class="text-clay" />
          <div class="flex flex-col">
            <span class="text-heading-3">{{ list.value()!.escrowTotal | fcfa }}</span>
            <span class="text-small text-on-forest-deep/75">{{ 'reservations.escrowBanner' | t }}</span>
          </div>
        </div>
      }

      <div class="mt-5 flex flex-col gap-3" [attr.aria-busy]="list.isLoading()">
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

      <aside class="hidden flex-col gap-5 lg:flex" [attr.aria-label]="'reservations.asideLabel' | t">
        <div class="panel-dark flex flex-col gap-3.5 rounded-[22px] bg-forest-deep p-6 text-on-forest-deep">
          <span class="text-eyebrow text-on-forest-deep/70">{{ 'reservations.fundsTitle' | t }}</span>
          <span class="font-display text-[40px] leading-none font-extrabold tracking-[-0.02em]">
            {{ plain(active.value()?.escrowTotal ?? 0) }} <span class="text-[18px] text-on-forest-deep/70">FCFA</span>
          </span>
          <span class="text-small text-on-forest-deep/80">{{ 'reservations.fundsBody' | t: { count: heldCount() } }}</span>
        </div>

        <section class="flex flex-col gap-3.5 rounded-[20px] border border-line bg-surface-raised p-[22px]">
          <div class="flex items-center justify-between gap-3">
            <h2 class="font-display text-[20px] font-bold">{{ 'nav.notifications' | t }}</h2>
            <a routerLink="/notifications" class="text-small font-semibold text-forest underline-offset-2 hover:underline">{{ 'portal.seeAll' | t }}</a>
          </div>
          @for (n of latestNotifications(); track n.id) {
            <a [routerLink]="n.link || '/notifications'" class="flex gap-3 text-ink">
              <span class="mt-[7px] h-2 w-2 shrink-0 rounded-full" [class]="n.read ? 'bg-line-strong' : 'bg-clay'" aria-hidden="true"></span>
              <span class="flex min-w-0 flex-col gap-0.5">
                <b class="text-small" [class.text-ink-muted]="n.read">{{ notificationText(n) }}</b>
                <span class="text-[13px] text-ink-muted">{{ n.createdAt | localDate: 'dayMonthTime' }}</span>
              </span>
            </a>
          } @empty {
            <p class="text-small text-ink-muted">{{ 'notifications.empty' | t }}</p>
          }
        </section>

        @if (toReview().length > 0) {
          <section class="flex flex-col gap-3 rounded-[20px] border border-line bg-surface-raised p-[22px]">
            <h2 class="font-display text-[18px] font-bold">{{ 'reservations.toReviewTitle' | t }}</h2>
            @for (v of toReview(); track v.id) {
              <div class="flex items-center gap-3">
                <div class="h-11 w-11 shrink-0 overflow-hidden rounded-xl bg-surface-sunken">
                  @if (v.listing.photo; as photo) {
                    <img [src]="photo" alt="" loading="lazy" class="h-full w-full object-cover" />
                  }
                </div>
                <div class="flex min-w-0 flex-1 flex-col">
                  <b class="truncate text-small">{{ v.listing.title }}</b>
                  <span class="truncate text-[12.5px] text-ink-muted">{{ v.farm.name }}</span>
                </div>
                <bt-button size="sm" [href]="'/account/reservations/' + v.id + '/review'">{{ 'reservations.action.review' | t }}</bt-button>
              </div>
            }
          </section>
        }

        @if (seller.loaded() && !seller.hasFarm()) {
          <section class="flex flex-col gap-2.5 rounded-[22px] bg-clay p-[22px] text-on-clay">
            <div class="flex items-center gap-2.5">
              <bt-icon name="leaf" [size]="22" />
              <b class="font-display text-[19px]">{{ 'reservations.sellTitle' | t }}</b>
            </div>
            <p class="text-small leading-normal">{{ 'reservations.sellBody' | t }}</p>
            <a routerLink="/sell" class="inline-flex h-11 items-center self-start rounded-full bg-ink px-5 text-body-strong text-surface transition hover:brightness-125">{{ 'reservations.sellCta' | t }}</a>
          </section>
        }
      </aside>
    </div>

    <!-- Pay a reserved-but-unpaid reservation. -->
    <bt-bottom-sheet [open]="sheet()?.kind === 'pay'" [title]="'reservations.paySheetTitle' | t" (closed)="close()">
      <div class="flex flex-col gap-3" role="radiogroup" [attr.aria-label]="'checkout.paymentMethodLabel' | t">
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
  private readonly notificationsApi = inject(NotificationsApi);
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

  // The right column, whatever tab is open: what is held, what the
  // buyer was last told, and the purchases still to rate.
  protected readonly seller = inject(SellerSpaceService);
  protected readonly active = apiResource<ReservationList>(() => this.reservationsApi.mineUrl('active'));
  private readonly done = apiResource<ReservationList>(() => this.reservationsApi.mineUrl('done'));
  private readonly inbox = apiResource<NotificationList>(() => this.notificationsApi.listUrl());
  protected readonly heldCount = computed(() => (this.active.value()?.items ?? []).filter((v) => v.payment?.status === 'held').length);
  protected readonly toReview = computed(() => (this.done.value()?.items ?? []).filter((v) => v.actions.includes('review')).slice(0, 3));
  protected readonly latestNotifications = computed(() => (this.inbox.value()?.items ?? []).slice(0, 3));

  protected plain(value: number): string {
    return Math.round(value).toLocaleString('fr-FR').replace(/\s/g, ' ');
  }

  /** A localized template per type when this client knows it, else the
   * server's own message. */
  protected notificationText(n: AppNotification): string {
    const key = `notifications.types.${n.type}`;
    const text = this.locale.t(key, n.params ?? {});
    return text === key ? n.message : text;
  }

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

  /** Confirming releases the escrow: the order leaves "En cours" for
   * "Terminées", so the list follows it there, highlighted, with its
   * review button in reach. */
  protected confirm(): void {
    const v = this.current();
    if (v) this.run(this.reservationsApi.confirmDelivery(v.id), 'reservations.confirmedToast', v.id);
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
  private run(request: Observable<unknown>, successTitle: string, completedId?: string): void {
    this.busy.set(true);
    request.subscribe({
      next: () => {
        this.busy.set(false);
        this.close();
        if (completedId) {
          this.group.set('done');
          this.router.navigate([], { queryParams: { highlight: completedId }, queryParamsHandling: 'merge', replaceUrl: true });
        }
        this.list.reload();
        this.active.reload();
        this.done.reload();
        this.toast.show({ tone: 'success', title: successTitle, detail: completedId ? 'reservations.confirmedToastDetail' : undefined });
      },
      error: (err: AppError) => {
        this.busy.set(false);
        this.toast.show({ tone: 'danger', title: 'reservations.actionFailed', detail: err.message });
      }
    });
  }
}
