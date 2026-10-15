import { ChangeDetectionStrategy, Component, DestroyRef, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { forkJoin, map, of, switchMap } from 'rxjs';
import { ReservationList, ReviewCriteria } from '../../domain/models';
import { ReservationsApi } from '../../data-access/reservations/reservations.api';
import { REVIEW_TAGS, ReviewsApi } from '../../data-access/reviews/reviews.api';
import { PhotoUploadService } from '../../core/http/photo-upload.service';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { apiResource } from '../../core/http/api-resource';
import { RatingComponent } from '../../shared/ui/rating/rating.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { StickyActionBarComponent } from '../../shared/ui/sticky-action-bar/sticky-action-bar.component';
import { LocalDatePipe } from '../../shared/pipes/local-date.pipe';

const MAX_PHOTOS = 4;
const CRITERIA: readonly (keyof ReviewCriteria)[] = ['health', 'conformity', 'communication', 'timeliness'];

interface PickedPhoto {
  readonly file: File;
  readonly preview: string;
}

/** "Noter mon achat", per the mobile design: an overall rating with its
 * word, the four detailed scores, what you liked, up to four photos
 * (compressed before upload), and a comment, published as a verified
 * review tied to the reservation. The API decides whether the
 * reservation can be reviewed (escrow released, not reviewed yet). */
@Component({
  selector: 'bt-review-form',
  imports: [FormsModule, RatingComponent, IconComponent, ButtonComponent, TextFieldComponent, StickyActionBarComponent, TranslatePipe, LocalDatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-2xl px-4 pb-32 pt-4 sm:px-8 lg:pb-16 lg:pt-10">
      <header class="flex items-center gap-3">
        <button
          type="button"
          (click)="back()"
          [attr.aria-label]="'common.back' | t"
          class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-line bg-surface-card"
        >
          <bt-icon name="chevron-left" [size]="22" />
        </button>
        <h1 class="text-heading-2">{{ 'reviewForm.title' | t }}</h1>
      </header>

      @if (target(); as v) {
        <div class="mt-5 flex items-center gap-3 rounded-lg border border-line bg-surface-card p-3">
          <div class="h-16 w-16 shrink-0 overflow-hidden rounded-md bg-surface-sunken">
            @if (v.listing.photo; as photo) {
              <img [src]="photo" [alt]="v.listing.title" class="h-full w-full object-cover" />
            }
          </div>
          <div class="flex min-w-0 flex-col">
            <span class="line-clamp-2 text-body-strong">{{ v.listing.title }}</span>
            <span class="text-small text-ink-muted">
              {{ v.farm.name }}
              @if (v.deliveredAt) {
                · {{ 'reviewForm.receivedOn' | t: { date: (v.deliveredAt | localDate: 'dayMonth') } }}
              }
            </span>
          </div>
        </div>

        @if (!v.actions.includes('review')) {
          <p class="mt-6 rounded-lg bg-surface-sunken p-4 text-body text-ink-muted">{{ 'reviewForm.notReviewable' | t }}</p>
        } @else {
          <form class="mt-6 flex flex-col gap-7" (submit)="$event.preventDefault(); submit()">
            <section class="flex flex-col items-center gap-3 rounded-xl border border-line bg-surface-card p-6 text-center">
              <h2 class="text-heading-4">{{ 'reviewForm.overall' | t }}</h2>
              <bt-rating [value]="rating()" interactive [size]="44" (changed)="rating.set($event)" />
              <span class="h-6 text-heading-4 text-clay-ink" aria-live="polite">
                @if (rating()) {
                  {{ 'reviewForm.ratingWord.' + rating() | t }}
                }
              </span>
            </section>

            <section class="flex flex-col">
              <h2 class="mb-2 text-heading-4">{{ 'reviewForm.detail' | t }}</h2>
              @for (c of criteria; track c) {
                <div class="flex min-h-14 items-center justify-between gap-3 border-b border-line">
                  <span class="text-body">{{ 'reviewForm.criteria.' + c | t }}</span>
                  <bt-rating [value]="scores()[c] ?? 0" interactive [size]="24" (changed)="setScore(c, $event)" />
                </div>
              }
            </section>

            <section class="flex flex-col gap-3">
              <h2 class="text-heading-4">{{ 'reviewForm.liked' | t }}</h2>
              <div class="flex flex-wrap gap-2">
                @for (tag of tagOptions; track tag) {
                  <button
                    type="button"
                    (click)="toggleTag(tag)"
                    [attr.aria-pressed]="tags().includes(tag)"
                    class="h-11 rounded-full border px-4 text-body-strong transition"
                    [class]="tags().includes(tag) ? 'border-forest bg-forest text-on-forest' : 'border-line-strong bg-surface-card text-ink'"
                  >
                    {{ 'reviewTags.' + tag | t }}
                  </button>
                }
              </div>
            </section>

            <section class="flex flex-col gap-3">
              <h2 class="text-heading-4">{{ 'reviewForm.photos' | t }} <span class="text-small font-normal text-ink-muted">· {{ 'reviewForm.optional' | t }}</span></h2>
              <div class="flex flex-wrap gap-2">
                @for (p of photos(); track p.preview; let i = $index) {
                  <div class="relative h-24 w-24">
                    <img [src]="p.preview" [alt]="'review.photoAlt' | t" class="h-full w-full rounded-md object-cover" />
                    <button
                      type="button"
                      (click)="removePhoto(i)"
                      [attr.aria-label]="'reviewForm.removePhoto' | t"
                      class="absolute -right-2 -top-2 flex h-8 w-8 items-center justify-center rounded-full bg-ink text-surface"
                    >
                      <bt-icon name="x" [size]="16" />
                    </button>
                  </div>
                }
                @if (photos().length < maxPhotos) {
                  <label class="flex h-24 w-24 cursor-pointer flex-col items-center justify-center gap-1 rounded-md border-2 border-dashed border-line-strong text-ink-muted">
                    <bt-icon name="camera" [size]="22" />
                    <span class="text-caption">{{ 'reviewForm.addPhoto' | t }}</span>
                    <input type="file" accept="image/*" multiple class="sr-only" (change)="addPhotos($event)" />
                  </label>
                }
              </div>
            </section>

            <bt-text-field [label]="'reviewForm.comment' | t" multiline [ngModel]="comment()" (ngModelChange)="comment.set($event)" name="comment" />

            <div class="flex gap-3 rounded-lg bg-forest-soft p-4 text-small text-on-forest-soft">
              <bt-icon name="shield" [size]="18" class="mt-0.5 shrink-0 text-forest" />
              <span>{{ 'reviewForm.verifiedNotice' | t: { ref: v.reference } }}</span>
            </div>

            <div class="hidden lg:block">
              <bt-button type="submit" size="lg" block [disabled]="!canSubmit()">{{ submitLabel() | t }}</bt-button>
            </div>
          </form>

          <bt-sticky-action-bar>
            <bt-button class="flex-1" size="lg" block [disabled]="!canSubmit()" (click)="submit()">{{ submitLabel() | t }}</bt-button>
          </bt-sticky-action-bar>
        }
      } @else if (list.isLoading()) {
        <div class="mt-6 flex flex-col gap-4" aria-busy="true">
          @for (h of [80, 160, 200]; track h) {
            <div class="animate-pulse rounded-lg bg-surface-sunken" [style.height.px]="h"></div>
          }
        </div>
      } @else {
        <p class="mt-6 text-body text-ink-muted">{{ 'reviewForm.notFound' | t }}</p>
      }
    </div>
  `
})
export class ReviewFormComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly reservationsApi = inject(ReservationsApi);
  private readonly reviewsApi = inject(ReviewsApi);
  private readonly uploads = inject(PhotoUploadService);
  private readonly toast = inject(ToastService);

  protected readonly criteria = CRITERIA;
  protected readonly tagOptions = REVIEW_TAGS;
  protected readonly maxPhotos = MAX_PHOTOS;

  private readonly id = toSignal(this.route.paramMap.pipe(map((p) => p.get('id') ?? '')), { initialValue: '' });
  // Reviewable reservations are released ones: the "done" tab.
  protected readonly list = apiResource<ReservationList>(() => this.reservationsApi.mineUrl('done'));
  protected readonly target = computed(() => this.list.value()?.items.find((v) => v.id === this.id()));

  protected readonly rating = signal(0);
  protected readonly scores = signal<ReviewCriteria>({});
  protected readonly tags = signal<string[]>([]);
  protected readonly photos = signal<PickedPhoto[]>([]);
  protected readonly comment = signal('');
  protected readonly submitting = signal(false);

  protected readonly canSubmit = computed(() => this.rating() > 0 && !this.submitting());
  protected readonly submitLabel = computed(() => (this.submitting() ? 'reviewForm.publishing' : 'reviewForm.publish'));

  constructor() {
    inject(DestroyRef).onDestroy(() => this.photos().forEach((p) => URL.revokeObjectURL(p.preview)));
  }

  protected setScore(c: keyof ReviewCriteria, value: number): void {
    this.scores.update((s) => ({ ...s, [c]: value }));
  }

  protected toggleTag(tag: string): void {
    this.tags.update((t) => (t.includes(tag) ? t.filter((x) => x !== tag) : [...t, tag]));
  }

  protected addPhotos(event: Event): void {
    const input = event.target as HTMLInputElement;
    const room = MAX_PHOTOS - this.photos().length;
    const picked = Array.from(input.files ?? [])
      .slice(0, room)
      .map((file) => ({ file, preview: URL.createObjectURL(file) }));
    this.photos.update((p) => [...p, ...picked]);
    input.value = '';
  }

  protected removePhoto(i: number): void {
    const removed = this.photos()[i];
    URL.revokeObjectURL(removed.preview);
    this.photos.update((p) => p.filter((_, idx) => idx !== i));
  }

  protected back(): void {
    this.router.navigateByUrl('/account/reservations');
  }

  protected submit(): void {
    const v = this.target();
    if (!v || !this.canSubmit()) return;
    this.submitting.set(true);
    const files = this.photos().map((p) => p.file);
    const keys$ = files.length
      ? forkJoin(files.map((f) => this.uploads.upload(f, (sizes) => this.reviewsApi.presignPhoto(v.id, sizes), { thumb: true })))
      : of<string[]>([]);
    keys$
      .pipe(
        switchMap((photoKeys) =>
          this.reviewsApi.create(v.id, {
            rating: this.rating(),
            comment: this.comment().trim() || undefined,
            criteria: this.scores(),
            tags: this.tags(),
            photoKeys
          })
        )
      )
      .subscribe({
        next: () => {
          this.toast.show({ tone: 'success', title: 'reservations.reviewedToast' });
          this.router.navigateByUrl('/account/reservations');
        },
        error: (err: AppError) => {
          this.submitting.set(false);
          this.toast.show({ tone: 'danger', title: 'reviewForm.failed', detail: err.message });
        }
      });
  }
}
