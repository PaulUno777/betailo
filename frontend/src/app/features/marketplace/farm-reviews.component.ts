import { ChangeDetectionStrategy, Component, computed, inject, linkedSignal, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';
import { FarmProfile, FarmReviewsPage, Review, ReviewFilter } from '../../domain/models';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { ReviewsApi } from '../../data-access/reviews/reviews.api';
import { AuthService } from '../../core/auth/auth.service';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { ReviewCardComponent } from '../../shared/ui/review-card/review-card.component';
import { RatingComponent } from '../../shared/ui/rating/rating.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { BottomSheetComponent } from '../../shared/ui/bottom-sheet/bottom-sheet.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';

const PAGE_SIZE = 10;
const FILTERS: readonly { id: ReviewFilter; labelKey: string }[] = [
  { id: '', labelKey: 'farmReviews.filterAll' },
  { id: 'photos', labelKey: 'farmReviews.filterPhotos' },
  { id: '5', labelKey: 'farmReviews.filterFive' },
  { id: 'critical', labelKey: 'farmReviews.filterCritical' }
];

/** "Avis vérifiés", per the mobile design: the farm's rating summary
 * (histogram and the four detailed averages, all computed by the API),
 * filter chips, and the reviews themselves with "Voir plus". Signed-in
 * visitors can mark reviews helpful; the farm's owner can answer each
 * one, once. */
@Component({
  selector: 'bt-farm-reviews',
  imports: [FormsModule, ReviewCardComponent, RatingComponent, IconComponent, ButtonComponent, BottomSheetComponent, TextFieldComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-2xl px-4 pb-10 pt-4 sm:px-8 lg:pt-10">
      <header class="flex items-center gap-3">
        <button
          type="button"
          (click)="back()"
          [attr.aria-label]="'common.back' | t"
          class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-line bg-surface-card"
        >
          <bt-icon name="chevron-left" [size]="22" />
        </button>
        <div class="flex min-w-0 flex-col">
          <h1 class="text-heading-2">{{ 'farmReviews.title' | t }}</h1>
          @if (farm.value(); as f) {
            <span class="truncate text-small text-ink-muted">{{ f.name }} · {{ 'review.count' | t: { count: summary()?.count ?? 0 } }}</span>
          }
        </div>
      </header>

      @if (summary(); as s) {
        <section class="mt-5 rounded-xl bg-forest-deep p-5 text-on-forest-deep">
          <div class="flex items-center gap-5">
            <div class="flex flex-col items-start gap-1">
              <span class="text-display-l leading-none">{{ s.average.toFixed(1) }}</span>
              <bt-rating [value]="s.average" [size]="16" />
            </div>
            <div class="flex flex-1 flex-col gap-1">
              @for (star of [5, 4, 3, 2, 1]; track star) {
                <div class="flex items-center gap-2 text-caption">
                  <span class="w-3 text-on-forest-deep/80">{{ star }}</span>
                  <span class="h-1.5 flex-1 overflow-hidden rounded-full bg-on-forest-deep/15">
                    <span class="block h-full rounded-full bg-clay" [style.width.%]="share(s.histogram[star] ?? 0, s.count)"></span>
                  </span>
                  <span class="w-6 text-right font-mono text-on-forest-deep/80">{{ s.histogram[star] ?? 0 }}</span>
                </div>
              }
            </div>
          </div>
          @if (hasCriteria()) {
            <div class="mt-4 grid grid-cols-2 gap-x-6 gap-y-2 border-t border-on-forest-deep/15 pt-4 text-small">
              @for (c of criteria; track c) {
                <div class="flex justify-between">
                  <span class="text-on-forest-deep/80">{{ 'reviewForm.criteriaShort.' + c | t }}</span>
                  <span class="font-mono">{{ s.criteria[c] ? s.criteria[c].toFixed(1) : '–' }}</span>
                </div>
              }
            </div>
          }
        </section>
      }

      <div class="no-scrollbar -mx-4 mt-5 flex gap-2 overflow-x-auto px-4" role="group">
        @for (f of filters; track f.id) {
          <button
            type="button"
            (click)="filter.set(f.id)"
            [attr.aria-pressed]="filter() === f.id"
            class="h-11 shrink-0 rounded-full border px-4 text-body-strong transition"
            [class]="filter() === f.id ? 'border-ink bg-ink text-surface' : 'border-line bg-surface-card text-ink'"
          >
            {{ f.labelKey | t }}
            @if (f.id === '' && summary()) {
              · {{ summary()!.count }}
            }
          </button>
        }
      </div>

      <div class="mt-4 flex flex-col gap-3" aria-live="polite">
        @for (r of items(); track r.id) {
          <bt-review-card
            [review]="r"
            [canVote]="auth.isAuthenticated()"
            [canReply]="isOwner()"
            [anonymousName]="'farmReviews.anonymous' | t"
            (helpful)="markHelpful(r)"
            (reply)="replyTo.set(r)"
          />
        } @empty {
          @if (page.isLoading()) {
            @for (i of [1, 2]; track i) {
              <div class="h-40 animate-pulse rounded-xl bg-surface-sunken"></div>
            }
          } @else {
            <p class="rounded-xl border border-dashed border-line-strong px-6 py-10 text-center text-body text-ink-muted">
              {{ 'farmReviews.empty' | t }}
            </p>
          }
        }
        @if (hasMore()) {
          <div class="flex justify-center pt-2">
            <bt-button variant="outline" [disabled]="page.isLoading()" (click)="pageNo.set(pageNo() + 1)">
              {{ (page.isLoading() ? 'common.loading' : 'farmReviews.loadMore') | t }}
            </bt-button>
          </div>
        }
      </div>
    </div>

    <bt-bottom-sheet [open]="!!replyTo()" [title]="'farmReviews.replyTitle' | t" (closed)="replyTo.set(null)">
      <p class="mb-3 text-small text-ink-muted">{{ 'farmReviews.replyHint' | t }}</p>
      <bt-text-field [label]="'farmReviews.replyLabel' | t" multiline [ngModel]="replyText()" (ngModelChange)="replyText.set($event)" />
      <div sheetFooter class="border-t border-line px-5 pt-3" style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))">
        <bt-button block size="lg" [disabled]="!replyText().trim() || sending()" (click)="sendReply()">{{ 'farmReviews.replySubmit' | t }}</bt-button>
      </div>
    </bt-bottom-sheet>
  `
})
export class FarmReviewsComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly farmsApi = inject(FarmsApi);
  private readonly reviewsApi = inject(ReviewsApi);
  private readonly toast = inject(ToastService);
  protected readonly auth = inject(AuthService);

  protected readonly filters = FILTERS;
  protected readonly criteria = ['health', 'conformity', 'communication', 'timeliness'] as const;

  private readonly farmId = toSignal(this.route.paramMap.pipe(map((p) => p.get('id') ?? '')), { initialValue: '' });
  protected readonly farm = apiResource<FarmProfile>(() => (this.farmId() ? this.farmsApi.detailUrl(this.farmId()) : undefined));

  protected readonly filter = signal<ReviewFilter>('');
  protected readonly pageNo = linkedSignal({ source: this.filter, computation: () => 1 });
  protected readonly page = apiResource<FarmReviewsPage>(() =>
    this.farmId() ? this.reviewsApi.farmReviewsUrl(this.farmId(), this.filter(), this.pageNo(), PAGE_SIZE) : undefined
  );

  /** Accumulated across "Voir plus"; replaced when the filter changes.
   * Local edits (a helpful vote, a reply) are patched in place. */
  protected readonly items = linkedSignal<{ res: FarmReviewsPage | undefined; page: number }, Review[]>({
    source: () => ({ res: this.page.value(), page: this.pageNo() }),
    computation: ({ res, page }, previous) => {
      const kept = previous?.value ?? [];
      if (!res) return page === 1 ? [] : kept;
      if (page === 1) return res.items;
      const seen = new Set(kept.map((r) => r.id));
      return [...kept, ...res.items.filter((r) => !seen.has(r.id))];
    }
  });

  protected readonly summary = computed(() => this.page.value()?.summary);
  protected readonly hasMore = computed(() => this.items().length < (this.page.value()?.total ?? 0));
  protected readonly hasCriteria = computed(() => {
    const c = this.summary()?.criteria;
    return !!c && (c.health || c.conformity || c.communication || c.timeliness) > 0;
  });
  protected readonly isOwner = computed(() => !!this.farm.value() && this.farm.value()!.ownerId === this.auth.user()?.id);

  protected readonly replyTo = signal<Review | null>(null);
  protected readonly replyText = signal('');
  protected readonly sending = signal(false);

  protected share(n: number, total: number): number {
    return total ? (n / total) * 100 : 0;
  }

  protected back(): void {
    if (this.router.lastSuccessfulNavigation()?.previousNavigation) {
      history.back();
    } else {
      this.router.navigateByUrl('/marketplace');
    }
  }

  protected markHelpful(r: Review): void {
    // Optimistic: the API counts each user once anyway.
    this.patch(r.id, { votedByMe: true, helpfulCount: r.helpfulCount + 1 });
    this.reviewsApi.markHelpful(r.id).subscribe({
      next: ({ helpfulCount }) => this.patch(r.id, { helpfulCount }),
      error: (err: AppError) => {
        this.patch(r.id, { votedByMe: false, helpfulCount: r.helpfulCount });
        this.toast.show({ tone: 'danger', title: 'farmReviews.voteFailed', detail: err.message });
      }
    });
  }

  protected sendReply(): void {
    const r = this.replyTo();
    if (!r) return;
    this.sending.set(true);
    this.reviewsApi.reply(r.id, this.replyText().trim()).subscribe({
      next: (updated) => {
        this.sending.set(false);
        this.patch(r.id, { reply: updated.reply });
        this.replyTo.set(null);
        this.replyText.set('');
        this.toast.show({ tone: 'success', title: 'farmReviews.replied' });
      },
      error: (err: AppError) => {
        this.sending.set(false);
        this.toast.show({ tone: 'danger', title: 'farmReviews.replyFailed', detail: err.message });
      }
    });
  }

  private patch(id: string, change: Partial<Review>): void {
    this.items.update((list) => list.map((r) => (r.id === id ? { ...r, ...change } : r)));
  }
}
