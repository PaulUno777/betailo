import { ChangeDetectionStrategy, Component, computed, effect, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { debounceTime, distinctUntilChanged } from 'rxjs';
import { ForumPost } from '../../domain/models';
import { ForumApi } from '../../data-access/forum/forum.api';
import { AuthService } from '../../core/auth/auth.service';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { LocaleService } from '../../core/i18n/locale.service';
import { ReferenceService } from '../../core/reference/reference.service';
import { categoryTone } from '../../domain/rules/category-tone';
import { relativeTime } from '../../domain/rules/relative-time';
import { isSolved } from '../../domain/rules/forum-thread';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { AvatarComponent } from '../../shared/ui/avatar/avatar.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { apiResource } from '../../core/http/api-resource';

type ForumSort = 'recent' | 'popular' | 'unanswered';
const SORTS: readonly ForumSort[] = ['recent', 'popular', 'unanswered'];
const TITLE_MAX = 120;
// Written out so Tailwind sees every class.
const DOT_CLASSES: Readonly<Record<string, string>> = {
  clay: 'bg-clay',
  forest: 'bg-forest',
  sky: 'bg-sky',
  amber: 'bg-amber',
  plum: 'bg-plum',
  alert: 'bg-alert',
  stone: 'bg-stone'
};

/** `/forum`: farmers asking each other. Category chips and sort tabs
 * (recent, most answered, still unanswered) over a debounced search,
 * all kept in the address so Back and shared links restore them; each
 * topic a card with its excerpt, who asked, when, and whether it was
 * solved. Signed-in users open a topic from the dialog. */
@Component({
  selector: 'bt-forum-list',
  imports: [FormsModule, RouterLink, AvatarComponent, TextFieldComponent, ButtonComponent, BadgeComponent, DialogComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-content-max px-4 py-6 sm:px-8 lg:px-18 lg:py-10">
      <header class="mb-6 flex flex-col gap-4 rounded-xl bg-forest-deep p-5 text-on-forest-deep sm:flex-row sm:items-center sm:justify-between sm:p-7">
        <div class="flex flex-col gap-1.5">
          <span class="text-eyebrow text-on-forest-deep/70">{{ 'forum.eyebrow' | t }}</span>
          <h1 class="text-heading-2">{{ 'forum.title' | t }}</h1>
          <p class="max-w-xl text-body text-on-forest-deep/80">{{ 'forum.pitch' | t }}</p>
        </div>
        <bt-button size="lg" (click)="openNewPost()">
          <bt-icon name="plus" [size]="18" />
          {{ 'forum.newPostButton' | t }}
        </bt-button>
      </header>

      <div class="grid gap-8 lg:grid-cols-[minmax(0,1fr)_300px]">
        <div class="flex min-w-0 flex-col gap-4">
          <label class="flex h-control-lg items-center gap-3 rounded-full border border-line-strong bg-surface-card px-4 focus-within:border-forest focus-within:shadow-[var(--shadow-focus)]">
            <bt-icon name="search" [size]="18" class="text-ink-muted" />
            <input
              type="search"
              class="min-w-0 flex-1 bg-transparent text-body text-ink outline-none placeholder:text-ink-muted"
              [placeholder]="'forum.searchPlaceholder' | t"
              [attr.aria-label]="'forum.searchPlaceholder' | t"
              [ngModel]="query()"
              (ngModelChange)="query.set($event)"
            />
          </label>

          <div class="no-scrollbar -mx-1 flex gap-2 overflow-x-auto px-1" role="tablist" [attr.aria-label]="'forum.categoryLabel' | t">
            <button type="button" role="tab" [attr.aria-selected]="!category()" (click)="category.set('')" [class]="chipClass(!category())">
              {{ 'forum.allCategories' | t }}
            </button>
            @for (c of reference.forumOptions(); track c.value) {
              <button type="button" role="tab" [attr.aria-selected]="category() === c.value" (click)="category.set(c.value)" [class]="chipClass(category() === c.value)">
                <span class="h-2 w-2 rounded-full" [class]="dotClass(c.value)"></span>
                {{ c.label }}
              </button>
            }
          </div>

          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="inline-flex gap-1 rounded-full bg-surface-sunken p-1" role="tablist" [attr.aria-label]="'forum.sortLabel' | t">
              @for (s of sorts; track s) {
                <button
                  type="button"
                  role="tab"
                  [attr.aria-selected]="sort() === s"
                  (click)="sort.set(s)"
                  class="h-9 rounded-full px-3.5 text-small font-semibold whitespace-nowrap transition"
                  [class]="sort() === s ? 'bg-surface-card text-ink shadow-sm' : 'text-ink-muted'"
                >
                  {{ 'forum.sort.' + s | t }}
                </button>
              }
            </div>
            @if (posts.value(); as list) {
              <span class="text-small text-ink-muted">{{ 'forum.topicsCount' | t: { count: list.length } }}</span>
            }
          </div>

          @if (posts.value(); as list) {
            @if (list.length) {
              <ul class="flex flex-col gap-3">
                @for (post of list; track post.id) {
                  <li>
                    <a
                      [routerLink]="['/forum', post.id]"
                      class="group flex flex-col gap-2.5 rounded-lg border border-line bg-surface-card p-4 text-ink no-underline transition hover:border-forest/50 hover:shadow-sm focus-visible:outline-2 focus-visible:outline-focus sm:p-5"
                    >
                      <span class="flex flex-wrap items-center gap-2">
                        <bt-badge [tone]="categoryTone(post.category)">{{ reference.forumLabelKey(post.category) | t }}</bt-badge>
                        @if (solved(post)) {
                          <bt-badge tone="forest-solid"><bt-icon name="check" [size]="12" /> {{ 'forum.solved' | t }}</bt-badge>
                        } @else if (!post.replies?.length) {
                          <bt-badge tone="amber">{{ 'forum.unanswered' | t }}</bt-badge>
                        }
                      </span>
                      <span class="text-heading-4 group-hover:text-forest">{{ post.title }}</span>
                      <span class="line-clamp-2 text-small text-ink-muted">{{ post.body }}</span>
                      <span class="mt-1 flex items-center gap-2.5 text-caption text-ink-muted">
                        <bt-avatar [name]="post.authorName || ('forum.someone' | t)" [src]="post.authorAvatarUrl" [size]="24" tone="sky" />
                        <span class="min-w-0 truncate font-semibold text-ink">{{ post.authorName || ('forum.someone' | t) }}</span>
                        <span aria-hidden="true">·</span>
                        <span class="shrink-0">{{ ago(post.createdAt) }}</span>
                        <span class="ml-auto flex shrink-0 items-center gap-1 rounded-full bg-surface-sunken px-2.5 py-1 font-semibold text-ink">
                          <bt-icon name="message" [size]="14" />
                          {{ post.replies?.length ?? 0 }}
                        </span>
                      </span>
                    </a>
                  </li>
                }
              </ul>
            } @else {
              <div class="flex flex-col items-center gap-3 rounded-xl border border-dashed border-line-strong px-6 py-12 text-center">
                <span class="flex h-14 w-14 items-center justify-center rounded-full bg-forest-soft text-forest">
                  <bt-icon name="message" [size]="26" />
                </span>
                @if (filtered()) {
                  <p class="text-body-strong">{{ 'forum.noMatch' | t }}</p>
                  <button type="button" class="text-small font-semibold text-forest underline" (click)="resetFilters()">{{ 'forum.resetFilters' | t }}</button>
                } @else {
                  <p class="text-body-strong">{{ 'forum.emptyTitle' | t }}</p>
                  <p class="max-w-sm text-small text-ink-muted">{{ 'forum.emptyBody' | t }}</p>
                  <bt-button (click)="openNewPost()">{{ 'forum.newPostButton' | t }}</bt-button>
                }
              </div>
            }
          } @else if (posts.isLoading()) {
            @for (i of [1, 2, 3]; track i) {
              <div class="h-36 animate-pulse rounded-lg bg-surface-sunken" aria-hidden="true"></div>
            }
          } @else if (posts.error()) {
            <div class="flex flex-col items-center gap-3 py-10 text-center">
              <p class="text-body">{{ 'forum.loadError' | t }}</p>
              <bt-button variant="outline" (click)="posts.reload()">{{ 'common.retry' | t }}</bt-button>
            </div>
          }
        </div>

        <aside class="flex flex-col gap-4">
          <section class="flex flex-col gap-3 rounded-lg border border-line bg-surface-card p-5">
            <h2 class="flex items-center gap-2 text-heading-4"><bt-icon name="leaf" [size]="18" class="text-forest" />{{ 'forum.tipsTitle' | t }}</h2>
            <ul class="flex flex-col gap-2.5 text-small text-ink-muted">
              @for (k of tips; track k) {
                <li class="flex gap-2"><bt-icon name="check" [size]="16" class="mt-0.5 shrink-0 text-forest" />{{ k | t }}</li>
              }
            </ul>
          </section>
          <section class="flex flex-col gap-2 rounded-lg bg-clay-soft p-5 text-clay-ink">
            <h2 class="flex items-center gap-2 text-heading-4"><bt-icon name="shield" [size]="18" />{{ 'forum.rulesTitle' | t }}</h2>
            <p class="text-small">{{ 'forum.rulesBody' | t }}</p>
          </section>
        </aside>
      </div>
    </div>

    <bt-dialog
      [open]="newPostOpen()"
      [title]="'forum.newPostDialogTitle' | t"
      [primaryLabel]="(submitting() ? 'forum.submitting' : 'forum.submitButton') | t"
      [primaryDisabled]="!canSubmit()"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="submitNewPost()"
      (secondaryClick)="newPostOpen.set(false)"
      (closed)="newPostOpen.set(false)"
    >
      <div class="flex flex-col gap-5">
        <fieldset class="flex flex-col gap-2">
          <legend class="mb-2 text-small text-ink-muted">{{ 'forum.categoryFieldLabel' | t }}</legend>
          <div class="flex flex-wrap gap-2">
            @for (c of reference.forumOptions(); track c.value) {
              <button type="button" [attr.aria-pressed]="newCategory() === c.value" (click)="newCategory.set(c.value)" [class]="chipClass(newCategory() === c.value)">
                <span class="h-2 w-2 rounded-full" [class]="dotClass(c.value)"></span>
                {{ c.label }}
              </button>
            }
          </div>
        </fieldset>
        <bt-text-field
          [label]="'forum.titleFieldLabel' | t"
          [hint]="'forum.titleCounter' | t: { count: newTitle().length, max: titleMax }"
          [error]="newTitle().length > titleMax ? ('forum.titleTooLong' | t: { max: titleMax }) : undefined"
          [ngModel]="newTitle()"
          (ngModelChange)="newTitle.set($event)"
        />
        <bt-text-field
          [label]="'forum.bodyFieldLabel' | t"
          [hint]="'forum.bodyHint' | t"
          multiline
          [ngModel]="newBody()"
          (ngModelChange)="newBody.set($event)"
        />
      </div>
    </bt-dialog>
  `
})
export class ForumListComponent {
  private readonly forumApi = inject(ForumApi);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);
  protected readonly reference = inject(ReferenceService);

  protected readonly sorts = SORTS;
  protected readonly titleMax = TITLE_MAX;
  protected readonly tips = ['forum.tip1', 'forum.tip2', 'forum.tip3'];

  // The address holds the search, so Back and shared links restore it.
  private readonly initial = this.route.snapshot.queryParamMap;
  protected readonly query = signal(this.initial.get('q') ?? '');
  protected readonly category = signal(this.initial.get('category') ?? '');
  protected readonly sort = signal<ForumSort>(SORTS.find((s) => s === this.initial.get('sort')) ?? 'recent');

  /** The search as typed, settled: one request per pause, not per key. */
  private readonly settledQuery = toSignal(toObservable(this.query).pipe(debounceTime(300), distinctUntilChanged()), {
    initialValue: this.query()
  });

  protected readonly posts = apiResource<ForumPost[]>(() =>
    this.forumApi.postsUrl({ q: this.settledQuery().trim() || undefined, category: this.category() || undefined, sort: this.sort() })
  );

  protected readonly filtered = computed(() => !!this.settledQuery().trim() || !!this.category() || this.sort() === 'unanswered');

  protected readonly categoryTone = categoryTone;

  protected readonly newPostOpen = signal(false);
  protected readonly newCategory = signal('');
  protected readonly newTitle = signal('');
  protected readonly newBody = signal('');
  protected readonly submitting = signal(false);
  protected readonly canSubmit = computed(
    () => !this.submitting() && !!this.newCategory() && !!this.newTitle().trim() && this.newTitle().length <= TITLE_MAX && !!this.newBody().trim()
  );

  constructor() {
    effect(() => {
      const queryParams = {
        q: this.settledQuery().trim() || null,
        category: this.category() || null,
        sort: this.sort() === 'recent' ? null : this.sort()
      };
      this.router.navigate([], { relativeTo: this.route, queryParams, queryParamsHandling: 'merge', replaceUrl: true });
    });
  }

  protected readonly solved = isSolved;

  protected ago(iso: string): string {
    return relativeTime(iso, this.locale.locale());
  }

  protected chipClass(active: boolean): string {
    return (
      'flex h-10 shrink-0 items-center gap-2 rounded-full border px-4 text-small font-semibold transition ' +
      (active ? 'border-forest bg-forest text-on-forest' : 'border-line-strong bg-surface-card text-ink hover:border-forest/60')
    );
  }

  /** The chip's dot wears the category's badge colour. */
  protected dotClass(category: string): string {
    return DOT_CLASSES[categoryTone(category)] ?? 'bg-ink-muted';
  }

  protected resetFilters(): void {
    this.query.set('');
    this.category.set('');
    this.sort.set('recent');
  }

  protected openNewPost(): void {
    if (!this.auth.isAuthenticated()) {
      this.router.navigate(['/login'], { queryParams: { redirectTo: '/forum' } });
      return;
    }
    if (!this.newCategory() && this.category()) this.newCategory.set(this.category());
    this.newPostOpen.set(true);
  }

  protected submitNewPost(): void {
    if (!this.canSubmit()) return;
    this.submitting.set(true);
    this.forumApi.create({ category: this.newCategory(), title: this.newTitle().trim(), body: this.newBody().trim() }).subscribe({
      next: (post) => {
        this.submitting.set(false);
        this.newPostOpen.set(false);
        this.newCategory.set('');
        this.newTitle.set('');
        this.newBody.set('');
        this.router.navigate(['/forum', post.id]);
      },
      error: (err: AppError) => {
        this.submitting.set(false);
        this.toast.show({ tone: 'danger', title: 'forum.postFailedTitle', detail: err.message });
      }
    });
  }
}
