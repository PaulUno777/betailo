import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';
import { ForumPost } from '../../domain/models';
import { ForumApi } from '../../data-access/forum/forum.api';
import { AuthService } from '../../core/auth/auth.service';
import { AppError } from '../../core/http/app-error';
import { ToastService } from '../../core/notifications/toast.service';
import { LocaleService } from '../../core/i18n/locale.service';
import { categoryTone } from '../../domain/rules/category-tone';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { TextFieldComponent } from '../../shared/ui/text-field/text-field.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { BadgeComponent } from '../../shared/ui/badge/badge.component';
import { apiResource } from '../../core/http/api-resource';

/** `/forum/:id` — thread + reply composer + "mark as best answer", per
 * `Forum.dc.html`'s open-thread state. The mark-best affordance is shown
 * to any signed-in viewer; the backend usecase is the actual authority
 * on who may call it (see [[feedback_backend_owns_constraints]]) — a
 * rejected call just surfaces the server's error, never a client guess
 * at ownership rules. */
@Component({
  selector: 'bt-forum-detail',
  imports: [FormsModule, RouterLink, TextFieldComponent, ButtonComponent, BadgeComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-3xl px-5 py-8 sm:px-0">
      <a routerLink="/forum" class="text-small text-ink-muted hover:text-forest">← {{ 'forum.backToForum' | t }}</a>

      @if (post.value(); as p) {
        <div class="mt-4 flex flex-col gap-3 border-b border-line pb-6">
          <bt-badge [tone]="categoryTone(p.category)" class="w-fit">{{ p.category }}</bt-badge>
          <h1 class="text-heading-2">{{ p.title }}</h1>
          <span class="text-small text-ink-muted">{{ 'forum.postedOn' | t: { date: dateLabel(p.createdAt) } }}</span>
          <p class="whitespace-pre-line text-body text-ink">{{ p.body }}</p>
        </div>

        <div class="mt-6 flex flex-col gap-4">
          <h2 class="text-heading-4">{{ 'forum.repliesTitle' | t }}</h2>
          @if (!p.replies?.length) {
            <p class="text-body text-ink-muted">{{ 'forum.noReplies' | t }}</p>
          }
          @for (reply of p.replies ?? []; track $index) {
            <div class="flex flex-col gap-2 rounded-lg border border-line bg-surface-card p-4">
              <div class="flex items-center justify-between">
                <span class="text-small text-ink-muted">{{ 'forum.postedOn' | t: { date: dateLabel(reply.createdAt) } }}</span>
                @if (p.bestReplyIndex === $index) {
                  <bt-badge tone="forest-solid">{{ 'forum.bestAnswerBadge' | t }}</bt-badge>
                } @else if (canMarkBest(p)) {
                  <button type="button" (click)="markBest(p.id, $index)" class="text-small text-forest underline">
                    {{ 'forum.markBestAnswer' | t }}
                  </button>
                }
              </div>
              <p class="whitespace-pre-line text-body text-ink">{{ reply.body }}</p>
            </div>
          }
        </div>

        <div class="mt-6 flex flex-col gap-3 border-t border-line pt-6">
          @if (auth.isAuthenticated()) {
            <bt-text-field
              [label]="'forum.replyPlaceholder' | t"
              multiline
              [ngModel]="replyBody()"
              (ngModelChange)="replyBody.set($event)"
            />
            <bt-button [disabled]="submittingReply() || !replyBody()" (click)="submitReply(p.id)" class="w-fit">
              {{ (submittingReply() ? 'forum.replySubmitting' : 'forum.replySubmit') | t }}
            </bt-button>
          } @else {
            <p class="text-body text-ink-muted">
              {{ 'forum.loginToReply' | t }}
              <a routerLink="/login" [queryParams]="{ redirectTo: '/forum/' + p.id }" class="text-forest">{{ 'auth.loginSubmit' | t }}</a>
            </p>
          }
        </div>
      } @else if (post.isLoading()) {
        <p class="mt-6 text-body text-ink-muted">{{ 'forum.loading' | t }}</p>
      } @else if (post.error()) {
        <p class="mt-6 text-body text-alert">{{ 'forum.notFound' | t }}</p>
      }
    </div>
  `
})
export class ForumDetailComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly forumApi = inject(ForumApi);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);
  protected readonly auth = inject(AuthService);

  private readonly id = toSignal(this.route.paramMap.pipe(map((p) => p.get('id') ?? '')), { initialValue: '' });

  protected readonly post = apiResource<ForumPost>(() => (this.id() ? this.forumApi.postUrl(this.id()) : undefined));

  protected readonly categoryTone = categoryTone;

  protected readonly replyBody = signal('');
  protected readonly submittingReply = signal(false);

  protected canMarkBest(post: ForumPost): boolean {
    return this.auth.isAuthenticated() && this.auth.user()?.id === post.authorId;
  }

  protected dateLabel(iso: string): string {
    const localeCode = this.locale.locale() === 'fr' ? 'fr-FR' : 'en-US';
    return new Date(iso).toLocaleDateString(localeCode, { day: 'numeric', month: 'short', year: 'numeric' });
  }

  protected submitReply(postId: string): void {
    if (this.submittingReply() || !this.replyBody()) return;
    this.submittingReply.set(true);
    this.forumApi.reply(postId, this.replyBody()).subscribe({
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
