import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { httpResource } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
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
import { DialogComponent } from '../../shared/ui/dialog/dialog.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';

/** `/forum` — per `Forum.dc.html`: search + category filter, a thread
 * list (category chip, title, reply count), and a "new topic" dialog for
 * signed-in users. Fully backed by `POST/GET /forum/posts` — nothing
 * here is a stub. */
@Component({
  selector: 'bt-forum-list',
  imports: [FormsModule, RouterLink, TextFieldComponent, ButtonComponent, BadgeComponent, DialogComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-content-max px-5 py-8 sm:px-18">
      <div class="mb-6 flex items-start justify-between gap-4">
        <div class="flex flex-col gap-1">
          <span class="text-eyebrow text-clay-ink">{{ 'forum.eyebrow' | t }}</span>
          <h1 class="text-heading-2">{{ 'forum.title' | t }}</h1>
        </div>
        <bt-button variant="solid" (click)="openNewPost()">
          <bt-icon name="plus" [size]="16" />
          {{ 'forum.newPostButton' | t }}
        </bt-button>
      </div>

      <div class="grid gap-8 lg:grid-cols-[1fr_280px]">
        <div>
          <div class="mb-6 grid gap-3 sm:grid-cols-[1fr_240px]">
            <bt-text-field [label]="'forum.searchPlaceholder' | t" [ngModel]="query()" (ngModelChange)="query.set($event)" />
            <bt-text-field
              [label]="'forum.categoryLabel' | t"
              [placeholder]="'forum.categoryPlaceholder' | t"
              [ngModel]="category()"
              (ngModelChange)="category.set($event)"
            />
          </div>

          @if (posts.isLoading()) {
            <p class="text-body text-ink-muted">{{ 'forum.loading' | t }}</p>
          } @else if ((posts.value() ?? []).length === 0) {
            <p class="text-body text-ink-muted">{{ 'forum.empty' | t }}</p>
          } @else {
            <div class="flex flex-col border-t border-line">
              @for (post of posts.value()!; track post.id) {
                <a
                  [routerLink]="['/forum', post.id]"
                  class="flex items-center gap-4 border-b border-line py-5 text-ink no-underline hover:text-forest"
                >
                  <bt-badge [tone]="categoryTone(post.category)">{{ post.category }}</bt-badge>
                  <span class="flex-1 text-body-strong">{{ post.title }}</span>
                  <span class="flex items-center gap-1.5 text-small text-ink-muted">
                    <bt-icon name="message" [size]="16" />
                    {{ 'forum.repliesCount' | t: { count: post.replies?.length ?? 0 } }}
                  </span>
                </a>
              }
            </div>
          }
        </div>

        <aside class="flex flex-col gap-4 rounded-lg bg-forest-deep p-5 text-on-forest-deep">
          <p class="text-heading-4">{{ 'forum.rulesTitle' | t }}</p>
          <p class="text-small text-on-forest-deep/80">{{ 'forum.rulesBody' | t }}</p>
        </aside>
      </div>
    </div>

    <bt-dialog
      [open]="newPostOpen()"
      [title]="'forum.newPostDialogTitle' | t"
      [primaryLabel]="(submitting() ? 'forum.submitting' : 'forum.submitButton') | t"
      [secondaryLabel]="'common.cancel' | t"
      (primaryClick)="submitNewPost()"
      (secondaryClick)="newPostOpen.set(false)"
      (closed)="newPostOpen.set(false)"
    >
      <div class="flex flex-col gap-4">
        <bt-text-field
          [label]="'forum.categoryFieldLabel' | t"
          [placeholder]="'forum.categoryFieldPlaceholder' | t"
          [ngModel]="newCategory()"
          (ngModelChange)="newCategory.set($event)"
        />
        <bt-text-field [label]="'forum.titleFieldLabel' | t" [ngModel]="newTitle()" (ngModelChange)="newTitle.set($event)" />
        <bt-text-field [label]="'forum.bodyFieldLabel' | t" multiline [ngModel]="newBody()" (ngModelChange)="newBody.set($event)" />
      </div>
    </bt-dialog>
  `
})
export class ForumListComponent {
  private readonly forumApi = inject(ForumApi);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly toast = inject(ToastService);
  private readonly locale = inject(LocaleService);

  protected readonly query = signal('');
  protected readonly category = signal('');

  protected readonly posts = httpResource<ForumPost[]>(() =>
    this.forumApi.postsUrl({ q: this.query() || undefined, category: this.category() || undefined, sort: 'recent' })
  );

  protected readonly categoryTone = categoryTone;

  protected readonly newPostOpen = signal(false);
  protected readonly newCategory = signal('');
  protected readonly newTitle = signal('');
  protected readonly newBody = signal('');
  protected readonly submitting = signal(false);

  protected openNewPost(): void {
    if (!this.auth.isAuthenticated()) {
      this.router.navigate(['/login'], { queryParams: { redirectTo: '/forum' } });
      return;
    }
    this.newPostOpen.set(true);
  }

  protected submitNewPost(): void {
    if (this.submitting() || !this.newTitle() || !this.newBody() || !this.newCategory()) {
      return;
    }
    this.submitting.set(true);
    this.forumApi.create({ category: this.newCategory(), title: this.newTitle(), body: this.newBody() }).subscribe({
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
