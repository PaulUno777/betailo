import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { IconName } from '../../shared/ui/icon/icon-paths';
import { AdminPageHeaderComponent } from './ui/admin-page-header.component';
import { BulkAddDialogComponent } from './reference/bulk-add-dialog.component';
import { ChangesPanelComponent } from './reference/changes-panel.component';
import { EntryDrawerComponent } from './reference/entry-drawer.component';
import { JsonDiffComponent } from './reference/json-diff.component';
import { Catalog, CatalogPart, partContent } from '../../domain/reference/catalog';
import { ReferenceStore } from './reference/reference.store';

type Tab = 'categories' | 'places' | 'lists' | 'history' | 'advanced';

/** `/admin/reference` — the layout of the reference data editor: one
 * route per resource (`categories`, `places`, `lists`, `history`,
 * `advanced`), each loaded when opened — the code editor only comes
 * with the JSON views — around what they share: the pending changes,
 * the add/edit drawer and the bulk paste dialog.
 *
 * It is how super admins change the reference data
 * every listing, farm and review is checked against. They browse it by
 * resource (categories and kinds, countries → regions → towns, the
 * vocabularies), add or edit one entry at a time — or many at once by
 * pasting — and every change lands in one shared changeset, checked by
 * the API as it is queued. The changeset is reviewed (as sentences, or
 * as a diff) and published as the next version with a note. History
 * compares and restores versions; Advanced edits everything as JSON. */
@Component({
  selector: 'bt-admin-reference',
  imports: [
    RouterLink,
    RouterLinkActive,
    RouterOutlet,
    AdminPageHeaderComponent,
    ButtonComponent,
    IconComponent,
    ChangesPanelComponent,
    EntryDrawerComponent,
    BulkAddDialogComponent,
    JsonDiffComponent,
    TranslatePipe
  ],
  providers: [ReferenceStore],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(window:focus)': 'store.load()' },
  template: `
    <div class="flex flex-col gap-5 p-4 sm:p-7 lg:px-9">
      <bt-admin-page-header [eyebrow]="'adminNav.group.system' | t" [title]="'adminReference.title' | t" [subtitle]="'adminReference.subtitle' | t" />

      @if (store.loadFailed()) {
        <div class="flex flex-col items-start gap-3" role="alert">
          <p class="text-body">{{ 'apiErrors.server' | t }}</p>
          <bt-button variant="outline" (click)="store.load()">{{ 'common.retry' | t }}</bt-button>
        </div>
      } @else if (!store.changeset()) {
        <div class="h-[60vh] animate-pulse rounded-2xl bg-surface-sunken" aria-busy="true"></div>
      } @else {
        <div class="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <div class="flex min-w-0 flex-col gap-4">
            <nav class="flex flex-wrap gap-1 border-b border-line" role="tablist">
              @for (t of tabs; track t.id) {
                <a
                  [routerLink]="t.id"
                  routerLinkActive
                  #rla="routerLinkActive"
                  [attr.aria-current]="rla.isActive ? 'page' : null"
                  class="-mb-px inline-flex items-center gap-1.5 border-b-2 px-3 pb-2.5 pt-1 text-small font-semibold no-underline transition"
                  [class]="rla.isActive ? 'border-forest text-ink' : 'border-transparent text-ink-muted hover:text-ink'"
                >
                  <bt-icon [name]="t.icon" [size]="16" />
                  {{ 'adminReference.tabs.' + t.id | t }}
                  @if (pendingIn(t.id); as n) {
                    <span class="rounded-full bg-clay px-1.5 font-mono text-[11px] text-on-clay">{{ n }}</span>
                  }
                </a>
              }
            </nav>

            <router-outlet />
          </div>

          <div class="xl:sticky xl:top-6 xl:self-start">
            <bt-changes-panel (review)="reviewing.set(true)" />
          </div>
        </div>
      }
    </div>

    @if (store.editing(); as request) {
      <bt-entry-drawer [request]="request" (closed)="store.editing.set(null)" />
    }
    @if (store.bulking(); as target) {
      <bt-bulk-add-dialog [target]="target" (closed)="store.bulking.set(null)" />
    }
    @if (reviewing()) {
      <div class="fixed inset-0 z-40 bg-ink/30" (click)="reviewing.set(false)" aria-hidden="true"></div>
      <section
        role="dialog"
        aria-modal="true"
        [attr.aria-label]="'adminReference.reviewDiff' | t"
        class="fixed inset-4 z-50 flex flex-col overflow-hidden rounded-2xl bg-surface-raised sm:inset-10"
        style="box-shadow: var(--shadow-dialog)"
        (keydown.escape)="reviewing.set(false)"
      >
        <header class="flex flex-wrap items-center gap-3 border-b border-line px-5 py-3">
          <h2 class="text-heading-4">{{ 'adminReference.reviewDiff' | t }}</h2>
          <div class="flex flex-1 flex-wrap gap-1.5" role="tablist">
            @for (name of store.changeset()?.touched ?? []; track name) {
              <button
                type="button"
                role="tab"
                [attr.aria-selected]="reviewPart() === name"
                (click)="reviewed.set(name)"
                class="h-8 rounded-full border px-3 text-caption font-semibold"
                [class]="reviewPart() === name ? 'border-ink bg-ink text-surface' : 'border-line-strong text-ink'"
              >
                {{ 'adminReference.part.' + name | t }} · v{{ store.changeset()?.baseVersions?.[name] }} → v{{ (store.changeset()?.baseVersions?.[name] ?? 0) + 1 }}
              </button>
            }
          </div>
          <button type="button" class="flex h-9 w-9 items-center justify-center rounded-md hover:bg-surface-sunken" (click)="reviewing.set(false)" [attr.aria-label]="'common.close' | t">
            <bt-icon name="x" [size]="18" />
          </button>
        </header>
        @defer {
          <bt-json-diff class="flex-1 overflow-auto" [original]="baseText()" [changed]="draftText()" />
        } @loading {
          <div class="flex-1 animate-pulse bg-surface-sunken"></div>
        }
      </section>
    }
  `
})
export class AdminReferenceComponent {
  protected readonly store = inject(ReferenceStore);

  protected readonly tabs: readonly { id: Tab; icon: IconName }[] = [
    { id: 'categories', icon: 'grid' },
    { id: 'places', icon: 'pin' },
    { id: 'lists', icon: 'flag' },
    { id: 'history', icon: 'clock' },
    { id: 'advanced', icon: 'filter' }
  ];
  protected readonly reviewing = signal(false);

  /** The reference the review shows: the one picked, else the first touched. */
  protected readonly reviewed = signal<CatalogPart | null>(null);
  protected readonly reviewPart = computed(() => {
    const touched = this.store.changeset()?.touched ?? [];
    const picked = this.reviewed();
    return picked && touched.includes(picked) ? picked : (touched[0] ?? 'animals');
  });
  protected readonly baseText = computed(() => partText(this.store.base.value(), this.reviewPart()));
  protected readonly draftText = computed(() => partText(this.store.draft(), this.reviewPart()));

  /** How many pending changes touch each tab. */
  private readonly pending = computed(() => {
    const counts: Partial<Record<Tab, number>> = {};
    for (const c of this.store.changes()) {
      const e = c.op.ref.entity;
      const tab: Tab = e === 'category' || e === 'subcategory' ? 'categories' : e === 'term' ? 'lists' : 'places';
      counts[tab] = (counts[tab] ?? 0) + 1;
    }
    return counts;
  });

  constructor() {
    void this.store.load();
  }

  protected pendingIn(tab: Tab): number {
    return this.pending()[tab] ?? 0;
  }
}

function partText(c: Catalog | undefined, name: CatalogPart): string {
  return c ? JSON.stringify(partContent(c, name), null, 2) : '';
}
