import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';
import { ForumPost, ForumReply } from '../../domain/models';
import { ForumApi } from '../../data-access/forum/forum.api';
import { AuthService } from '../../core/auth/auth.service';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { LocaleService } from '../../core/i18n/locale.service';
import { ReferenceService } from '../../core/reference/reference.service';
import { categoryTone } from '../../domain/rules/category-tone';
import { relativeTime } from '../../domain/rules/relative-time';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { AvatarComponent } from '../../shared/ui/avatar/avatar.component';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { apiResource } from '../../core/http/api-resource';

/** One reply with its place in the thread (the API marks the best
 * answer by index). */
interface ThreadReply {
  readonly reply: ForumReply;
  readonly index: number;
  readonly best: boolean;
  readonly byAuthor: boolean;
}

/** `/forum/:id`: the question in its own card, the best answer pinned
 * right under it, then the other replies in order, and the composer.
 * "Mark as best answer" shows to the topic's author; the server stays
 * the authority on who may call it (see
 * [[feedback_backend_owns_constraints]]): a refused call surfaces its
 * error. */
@Component({
  selector: 'bt-forum-detail',
  imports: [FormsModule, RouterLink, AvatarComponent, TextFieldComponent, ButtonComponent, BadgeComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-3xl px-4 py-6 sm:px-8 lg:py-10">
      <a routerLink="/forum" class="inline-flex items-center gap-1 text-small font-semibold text-ink-muted hover:text-forest">
        <bt-icon name="chevron-left" [size]="16" />{{ 'forum.backToForum' | t }}
      </a>

      @if (post.value(); as p) {
        <article class="mt-4 flex flex-col gap-4 rounded-xl border border-line bg-surface-card p-5 sm:p-7">
          <div class="flex flex-wrap items-center gap-2">
            <bt-badge [tone]="categoryTone(p.category)">{{ reference.forumLabelKey(p.category) | t }}</bt-badge>
            @if (solved()) {
              <bt-badge tone="forest-solid"><bt-icon name="check" [size]="12" /> {{ 'forum.solved' | t }}</bt-badge>
            }
          </div>
          <h1 class="text-heading-2">{{ p.title }}</h1>
          <div class="flex items-center gap-2.5">
            <bt-avatar [name]="p.authorName || ('forum.someone' | t)" [src]="p.authorAvatarUrl" [size]="36" tone="sky" />
            <span class="flex flex-col">
              <span class="text-small font-semibold text-ink">{{ p.authorName || ('forum.someone' | t) }}</span>
              <span class="text-caption text-ink-muted" [title]="p.createdAt">{{ ago(p.createdAt) }}</span>
            </span>
          </div>
          <p class="whitespace-pre-line text-body text-ink">{{ p.body }}</p>
        </article>

        <section class="mt-8 flex flex-col gap-3" aria-labelledby="replies-title">
          <h2 id="replies-title" class="flex items-center gap-2 text-heading-4">
            <bt-icon name="message" [size]="18" class="text-forest" />
            {{ 'forum.repliesCount' | t: { count: replies().length } }}
          </h2>
          @for (r of replies(); track r.index) {
            <div
              class="flex flex-col gap-2.5 rounded-lg border p-4"
              [class]="r.best ? 'border-forest bg-forest-soft/40 shadow-sm' : 'border-line bg-surface-card'"
            >
              @if (r.best) {
                <span class="flex items-center gap-1.5 text-caption font-bold uppercase tracking-wide text-forest">
                  <bt-icon name="check" [size]="14" />{{ 'forum.bestAnswerBadge' | t }}
                </span>
              }
              <div class="flex items-center justify-between gap-3">
                <span class="flex min-w-0 items-center gap-2.5">
                  <bt-avatar [name]="r.reply.authorName || ('forum.someone' | t)" [src]="r.reply.authorAvatarUrl" [size]="28" tone="sky" />
                  <span class="flex min-w-0 flex-col">
                    <span class="flex items-center gap-1.5 truncate text-small font-semibold text-ink">
                      {{ r.reply.authorName || ('forum.someone' | t) }}
                      @if (r.byAuthor) {
                        <bt-badge tone="neutral">{{ 'forum.authorTag' | t }}</bt-badge>
                      }
                    </span>
                    <span class="text-caption text-ink-muted" [title]="r.reply.createdAt">{{ ago(r.reply.createdAt) }}</span>
                  </span>
                </span>
                @if (!r.best && canMarkBest(p)) {
                  <button type="button" (click)="markBest(p.id, r.index)" class="shrink-0 text-small font-semibold text-forest underline underline-offset-2">
                    {{ 'forum.markBestAnswer' | t }}
                  </button>
                }
              </div>
              <p class="whitespace-pre-line text-body text-ink">{{ r.reply.body }}</p>
            </div>
          } @empty {
            <p class="rounded-lg border border-dashed border-line-strong p-5 text-center text-body text-ink-muted">{{ 'forum.noRepliesYet' | t }}</p>
          }
        </section>

        <section class="mt-6">
          @if (auth.isAuthenticated()) {
            <div class="flex gap-3 rounded-lg border border-line bg-surface-card p-4">
              <bt-avatar [name]="auth.user()?.displayName || ('forum.someone' | t)" [src]="auth.user()?.avatarUrl" [size]="36" tone="clay" class="hidden sm:inline-flex" />
              <div class="flex min-w-0 flex-1 flex-col gap-3">
                <bt-text-field [label]="'forum.replyPlaceholder' | t" multiline [ngModel]="replyBody()" (ngModelChange)="replyBody.set($event)" />
                <bt-button [disabled]="submittingReply() || !replyBody().trim()" (click)="submitReply(p.id)" class="self-end">
                  {{ (submittingReply() ? 'forum.replySubmitting' : 'forum.replySubmit') | t }}
                </bt-button>
              </div>
            </div>
          } @else {
            <div class="flex flex-col items-start gap-3 rounded-lg bg-forest-soft p-5 sm:flex-row sm:items-center sm:justify-between">
              <p class="text-body text-on-forest-soft">{{ 'forum.loginToReply' | t }}</p>
              <bt-button variant="solid" href="/login" [queryParams]="{ redirectTo: '/forum/' + p.id }">{{ 'auth.loginSubmit' | t }}</bt-button>
            </div>
          }
        </section>
      } @else if (post.isLoading()) {
        <div class="mt-4 flex flex-col gap-4" aria-busy="true">
          <div class="h-56 animate-pulse rounded-xl bg-surface-sunken"></div>
          <div class="h-24 animate-pulse rounded-lg bg-surface-sunken"></div>
        </div>
      } @else if (post.error()) {
        <div class="mt-10 flex flex-col items-center gap-3 text-center">
          <bt-icon name="search" [size]="32" class="text-ink-muted" />
          <p class="text-body">{{ 'forum.notFound' | t }}</p>
          <bt-button variant="outline" href="/forum">{{ 'forum.backToForum' | t }}</bt-button>
        </div>
      }
    </div>
  `
})
export class ForumDetailComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly forumApi = inject(ForumApi);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);
  protected readonly auth = inject(AuthService);
  protected readonly reference = inject(ReferenceService);

  private readonly id = toSignal(this.route.paramMap.pipe(map((p) => p.get('id') ?? '')), { initialValue: '' });

  protected readonly post = apiResource<ForumPost>(() => (this.id() ? this.forumApi.postUrl(this.id()) : undefined));

  protected readonly solved = computed(() => {
    const p = this.post.value();
    return p?.bestReplyIndex !== undefined && p?.bestReplyIndex !== null;
  });

  /** The best answer first, then the others in the order they came. */
  protected readonly replies = computed<ThreadReply[]>(() => {
    const p = this.post.value();
    if (!p) return [];
    const all = (p.replies ?? []).map((reply, index) => ({ reply, index, best: p.bestReplyIndex === index, byAuthor: reply.authorId === p.authorId }));
    return [...all.filter((r) => r.best), ...all.filter((r) => !r.best)];
  });

  protected readonly categoryTone = categoryTone;

  protected readonly replyBody = signal('');
  protected readonly submittingReply = signal(false);

  protected canMarkBest(post: ForumPost): boolean {
    return this.auth.isAuthenticated() && this.auth.user()?.id === post.authorId;
  }

  protected ago(iso: string): string {
    return relativeTime(iso, this.locale.locale());
  }

  protected submitReply(postId: string): void {
    const body = this.replyBody().trim();
    if (this.submittingReply() || !body) return;
    this.submittingReply.set(true);
    this.forumApi.reply(postId, body).subscribe({
      next: () => {
        this.submittingReply.set(false);
        this.replyBody.set('');
        this.post.reload();
      },
      error: (err: AppError) => {
        this.submittingReply.set(false);
        this.toast.show({ tone: 'danger', title: 'forum.replyFailedTitle', detail: err.message });
      }
    });
  }

  protected markBest(postId: string, index: number): void {
    this.forumApi.markBestAnswer(postId, index).subscribe({
      next: () => this.post.reload(),
      error: (err: AppError) => {
        this.toast.show({ tone: 'danger', title: 'forum.markBestFailedTitle', detail: err.message });
      }
    });
  }
}
