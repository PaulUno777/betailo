import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { LocaleService } from '../../../core/i18n/locale.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { cycleTypeLabelKey } from '../../../domain/reference/cycle-types';
import { dotClass } from '../../../domain/reference/species';
import { IconComponent } from '../../../shared/ui/icon/icon.component';
import { AnimalIconComponent } from '../../../shared/ui/animal-icon/animal-icon.component';
import { EntryBadgesComponent } from './entry-badges.component';
import { RowActionsComponent } from './row-actions.component';
import { ReferenceStore } from './reference.store';
import { CatalogRef } from './reference.types';

type Filter = 'all' | 'active' | 'inactive' | 'pending';

/** The animal categories and their kinds, as the draft has them:
 * searchable in both languages, each with what uses it and what is
 * pending on it, editable in place. */
@Component({
  selector: 'bt-categories-view',
  imports: [FormsModule, IconComponent, AnimalIconComponent, EntryBadgesComponent, RowActionsComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-wrap items-center gap-2">
      <div class="relative min-w-60 flex-1">
        <bt-icon name="search" [size]="16" class="pointer-events-none absolute left-3 top-2.5 text-ink-muted" />
        <input class="bo-input pl-9" type="search" [placeholder]="'adminReference.searchKinds' | t" [ngModel]="query()" (ngModelChange)="query.set($event)" />
      </div>
      @for (f of filters; track f) {
        <button type="button" class="h-9 rounded-full border px-3 text-caption font-semibold" [class]="filter() === f ? 'border-ink bg-ink text-surface' : 'border-line-strong text-ink'" (click)="filter.set(f)">
          {{ 'adminReference.filter.' + f | t }}
        </button>
      }
      <button type="button" class="inline-flex h-9 items-center gap-1.5 rounded-full bg-forest px-4 text-caption font-semibold text-on-forest" (click)="store.editing.set({ mode: 'add', entity: 'category', ref: { entity: 'category' } })">
        <bt-icon name="plus" [size]="14" /> {{ 'adminReference.addCategory' | t }}
      </button>
    </div>

    <div class="mt-3 flex flex-col gap-3">
      @for (cat of rows(); track cat.key; let first = $first, last = $last) {
        <section class="bo-card overflow-hidden" [class.opacity-70]="!cat.active">
          <div class="flex flex-wrap items-center gap-3 border-b border-line px-4 py-3">
            <span class="flex h-11 w-11 items-center justify-center rounded-xl bg-forest-soft text-on-forest-soft">
              <bt-animal-icon [name]="cat.icon" [size]="30" />
            </span>
            <div class="flex min-w-0 flex-1 flex-col">
              <span class="flex flex-wrap items-center gap-2">
                <b class="text-body-strong">{{ cat.label[lang()] }}</b>
                <span class="h-2.5 w-2.5 rounded-full" [class]="dotClass(cat.color)" [attr.title]="cat.color"></span>
                <bt-entry-badges [mark]="mark('category:' + cat.key)" [active]="cat.active" [usage]="cat.usage" />
              </span>
              <span class="text-caption text-ink-muted">{{ cat.label[otherLang()] }} · <span class="font-mono">{{ cat.key }}</span> · {{ 'adminReference.kindsCount' | t: { count: cat.subcategories.length } }}</span>
            </div>
            <bt-row-actions
              [active]="cat.active"
              [first]="first"
              [last]="last"
              [busy]="store.busy()"
              (edit)="store.editing.set({ mode: 'edit', entity: 'category', ref: ref(cat.key) })"
              (toggled)="toggle(ref(cat.key), cat.active)"
              (moved)="move(ref(cat.key), cat.index + $event)"
            />
          </div>
          <ul>
            @for (s of cat.subcategories; track s.key; let sFirst = $first, sLast = $last) {
              <li class="flex flex-wrap items-center gap-3 border-b border-line px-4 py-2 last:border-b-0" [class.opacity-60]="!s.active">
                <bt-animal-icon [name]="s.icon" [size]="24" class="text-ink-muted" />
                <div class="flex min-w-0 flex-1 flex-col">
                  <span class="flex flex-wrap items-center gap-2 text-small">
                    <span class="font-semibold">{{ s.label[lang()] }}</span>
                    <bt-entry-badges [mark]="mark('subcategory:' + s.key)" [active]="s.active" [usage]="s.usage" />
                  </span>
                  <span class="text-caption text-ink-muted">
                    {{ s.label[otherLang()] }} · <span class="font-mono">{{ s.key }}</span> · {{ cycleTypeLabel(s.cycleType) | t }} · {{ s.unit.label[lang()] }}
                  </span>
                </div>
                <bt-row-actions
                  [active]="s.active"
                  [first]="sFirst"
                  [last]="sLast"
                  [busy]="store.busy()"
                  (edit)="store.editing.set({ mode: 'edit', entity: 'subcategory', ref: { entity: 'subcategory', key: s.key } })"
                  (toggled)="toggle({ entity: 'subcategory', key: s.key }, s.active)"
                  (moved)="move({ entity: 'subcategory', key: s.key }, s.index + $event)"
                />
              </li>
            }
          </ul>
          <div class="flex flex-wrap gap-2 px-4 py-2.5">
            <button type="button" class="inline-flex items-center gap-1 text-caption font-semibold text-forest hover:underline" (click)="store.editing.set({ mode: 'add', entity: 'subcategory', ref: { entity: 'subcategory', parent: cat.key } })">
              <bt-icon name="plus" [size]="14" /> {{ 'adminReference.addKind' | t }}
            </button>
            <button type="button" class="inline-flex items-center gap-1 text-caption font-semibold text-forest hover:underline" (click)="store.bulking.set({ entity: 'subcategory', parent: cat.key, where: cat.label[lang()] })">
              <bt-icon name="upload" [size]="14" /> {{ 'adminReference.pasteMany' | t }}
            </button>
          </div>
        </section>
      } @empty {
        <p class="py-10 text-center text-body text-ink-muted">{{ 'adminReference.nothingFound' | t }}</p>
      }
    </div>
  `
})
export class CategoriesViewComponent {
  protected readonly store = inject(ReferenceStore);
  private readonly locale = inject(LocaleService);


  protected readonly filters: readonly Filter[] = ['all', 'active', 'inactive', 'pending'];
  protected readonly query = signal('');
  protected readonly filter = signal<Filter>('all');
  protected readonly dotClass = dotClass;
  protected readonly cycleTypeLabel = cycleTypeLabelKey;
  protected readonly lang = computed(() => this.locale.locale());
  protected readonly otherLang = computed(() => (this.lang() === 'fr' ? 'en' : 'fr'));

  /** Categories and kinds matching the search and the filter; a
   * category shows when it or any of its kinds matches. */
  protected readonly rows = computed(() => {
    const q = fold(this.query());
    const f = this.filter();
    const usage = this.store.usage.value() ?? {};
    const marks = this.store.marks();
    const keep = (key: string, label: { fr: string; en: string }, active: boolean, id: string) =>
      (!q || fold(`${key} ${label.fr} ${label.en}`).includes(q)) &&
      (f === 'all' || (f === 'active' && active) || (f === 'inactive' && !active) || (f === 'pending' && marks.has(id)));
    return (this.store.draft()?.categories ?? [])
      .map((cat, index) => {
        const subs = cat.subcategories.map((s, i) => ({ ...s, index: i, usage: usage[`species:${s.key}`] ?? 0 }));
        const own = keep(cat.key, cat.label, cat.active, `category:${cat.key}`);
        const matching = subs.filter((s) => keep(s.key, s.label, s.active, `subcategory:${s.key}`));
        return { ...cat, index, usage: subs.reduce((n, s) => n + s.usage, 0), subcategories: own && !q ? subs : matching, show: own || matching.length > 0 };
      })
      .filter((c) => c.show);
  });

  protected ref(key: string): CatalogRef {
    return { entity: 'category', key };
  }

  protected mark(id: string) {
    return this.store.marks().get(id);
  }

  protected toggle(ref: CatalogRef, active: boolean): void {
    void this.store.queue([{ kind: 'update', ref, value: { active: !active } }]);
  }

  protected move(ref: CatalogRef, position: number): void {
    void this.store.queue([{ kind: 'move', ref, position }]);
  }
}

function fold(text: string): string {
  return text
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .trim();
}
