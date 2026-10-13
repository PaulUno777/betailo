import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';
import { ReservationAction, ReservationView } from '../../domain/models';
import { whatsappUrl } from '../../domain/rules/contact';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { CountdownComponent } from '../../shared/ui/countdown/countdown.component';
import { FcfaCurrencyPipe } from '../../shared/pipes/fcfa-currency.pipe';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';

/** The status line under the progress bar, picked from the server's own
 * state (group, steps, dispute) rather than recomputed rules. */
interface StatusLine { key: string; tone: 'clay' | 'forest' | 'muted' | 'alert'; countdown?: string; countdownKey?: string }

/** One "Mes réservations" card, per the mobile design: what was reserved
 * and from whom, the escrow progress bar, a status line (with a live
 * countdown to auto-release), and exactly the
 * buttons the API lists in `actions`. Purely presentational: it emits
 * the chosen action and the page carries it out. */
@Component({
  selector: 'bt-reservation-card',
  imports: [TranslatePipe, ButtonComponent, IconComponent, CountdownComponent, FcfaCurrencyPipe, LocalDatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <article
      class="flex flex-col gap-4 rounded-xl border bg-surface-card p-4 transition"
      [class]="needsAttention() ? 'border-2 border-clay' : 'border-line'"
      [class.ring-4]="highlighted()"
      [class.ring-forest-soft]="highlighted()"
    >
      <div class="flex gap-3">
        <div class="h-16 w-16 shrink-0 overflow-hidden rounded-md bg-surface-sunken sm:h-20 sm:w-20">
          @if (view().listing.photo; as photo) {
            <img [src]="photo" alt="" loading="lazy" class="h-full w-full object-cover" />
          }
        </div>
        <div class="flex min-w-0 flex-1 flex-col">
          <div class="flex items-baseline justify-between gap-2">
            <span class="text-mono-id text-ink-muted">
              {{ view().reference }}
              <span class="hidden sm:inline"> · {{ 'reservations.reservedOn' | t: { date: (view().createdAt | localDate: 'dayMonth') } }}</span>
            </span>
            <span class="shrink-0 text-amount text-body-strong">{{ view().totalAmount | fcfa }}</span>
          </div>
          <span class="line-clamp-2 text-body-strong">{{ view().quantity }} × {{ view().listing.title }}</span>
          @if (perspective() === 'buyer') {
            <span class="truncate text-small text-ink-muted">{{ view().farm.name }} · {{ view().farm.location }}</span>
          } @else {
            <span class="truncate text-small text-ink-muted">{{ view().cycle.reference }} · {{ view().payment?.method ? ('paymentMethod.' + methodKey() | t) : ('reservations.status.unpaidShort' | t) }}</span>
          }
        </div>
      </div>

      @if (view().group !== 'cancelled') {
        <div class="flex flex-col gap-2">
          <div class="grid grid-cols-4 gap-1.5" role="img" [attr.aria-label]="'reservations.progressLabel' | t">
            @for (step of view().steps; track step.key) {
              <span
                class="h-1.5 rounded-full"
                [class]="step.state === 'done' ? 'bg-forest' : step.state === 'current' ? 'animate-pulse bg-clay-soft ring-1 ring-inset ring-clay' : 'bg-surface-sunken'"
              ></span>
            }
          </div>
          <div class="hidden grid-cols-4 gap-1.5 sm:grid">
            @for (step of view().steps; track step.key) {
              <!-- The current step is what we're waiting for, not done yet. -->
              <span class="text-caption" [class]="step.state === 'upcoming' ? 'text-ink-muted' : step.state === 'current' ? 'font-semibold text-clay-ink' : 'text-ink'">
                {{ 'reservations.step.' + step.key | t }}
              </span>
            }
          </div>
        </div>
      }

      <div class="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
        <span class="text-small font-semibold" [class]="statusToneClass()">{{ status().key | t }}</span>
        @if (status().countdown; as until) {
          <span class="text-small text-ink-muted">
            {{ status().countdownKey! | t }} <bt-countdown [until]="until" />
          </span>
        }
      </div>

      @if (primaryActions().length) {
        <div class="flex flex-wrap gap-2">
          @for (a of primaryActions(); track a) {
            <bt-button
              [variant]="a === 'confirm' || a === 'pay' || a === 'mark_delivered' ? 'solid' : a === 'cancel' ? 'ghost' : 'outline'"
              class="flex-1 sm:flex-none"
              [block]="true"
              (click)="action.emit(a)"
            >
              {{ 'reservations.action.' + a | t }}
            </bt-button>
          }
          @if (contactUrl(); as url) {
            <a
              [href]="url"
              target="_blank"
              rel="noopener"
              class="flex h-control-md flex-1 items-center justify-center gap-2 rounded-full border border-line-strong px-5 text-body-strong sm:flex-none"
            >
              <bt-icon name="message" [size]="18" />
              {{ 'reservations.action.contact' | t }}
            </a>
          }
        </div>
      }

      @if (has('review')) {
        <div class="flex items-center justify-between gap-3 rounded-lg bg-clay-soft p-4">
          <div class="flex flex-col">
            <span class="text-body-strong text-clay-ink">{{ 'reservations.reviewCtaTitle' | t }}</span>
            <span class="text-small text-clay-ink/80">{{ 'reservations.reviewCtaBody' | t }}</span>
          </div>
          <bt-button size="md" (click)="action.emit('review')">{{ 'reservations.action.review' | t }} ★</bt-button>
        </div>
      }
    </article>
  `
})
export class ReservationCardComponent {
  readonly view = input.required<ReservationView>();
  readonly highlighted = input(false);
  /** Whose list this card is in: the buyer's, or the selling farmer's. */
  readonly perspective = input<'buyer' | 'farmer'>('buyer');
  /** Already translated (the page owns the wording and the locale). */
  readonly whatsappMessage = input('');
  readonly action = output<ReservationAction>();

  protected readonly methodKey = computed(
    () => ({ mtn_momo: 'mtnMomo', orange_money: 'orangeMoney', card: 'card' })[this.view().payment?.method ?? 'card']
  );

  protected has(a: ReservationAction): boolean {
    return this.view().actions.includes(a);
  }

  /** Everything but review (its own band) and contact (a link). */
  protected readonly primaryActions = computed(() => this.view().actions.filter((a) => a !== 'review' && a !== 'contact'));

  protected readonly contactUrl = computed(() => {
    const phone = this.view().farm.contactPhone;
    return this.has('contact') && phone ? whatsappUrl(phone, this.whatsappMessage()) : null;
  });

  /** "Livré · à confirmer" and friends need the buyer now. */
  protected readonly needsAttention = computed(() => this.has('confirm') || this.has('pay') || this.has('mark_delivered'));

  protected readonly status = computed<StatusLine>(() => {
    const v = this.view();
    if (v.group === 'cancelled') {
      return { key: v.payment?.status === 'refunded' ? 'reservations.status.refunded' : 'reservations.status.cancelled', tone: 'muted' };
    }
    if (v.group === 'done') return { key: 'reservations.status.released', tone: 'forest' };
    if (v.disputeOpen) return { key: 'reservations.status.disputed', tone: 'alert' };
    if (v.payment?.status === 'pending') {
      // On the waiting list: no deadline, the places go to whoever pays.
      return { key: 'reservations.status.unpaid', tone: 'clay' };
    }
    const current = v.steps.find((s) => s.state === 'current')?.key;
    if (current === 'confirmed') {
      const key = this.perspective() === 'farmer' ? 'reservations.status.awaitingBuyer' : 'reservations.status.toConfirm';
      return { key, tone: 'clay', countdown: v.autoReleaseAt, countdownKey: 'reservations.autoReleaseIn' };
    }
    return { key: 'reservations.status.awaitingDelivery', tone: 'forest' };
  });

  protected readonly statusToneClass = computed(
    () => ({ clay: 'text-clay-ink', forest: 'text-forest', muted: 'text-ink-muted', alert: 'text-alert' })[this.status().tone]
  );
}
