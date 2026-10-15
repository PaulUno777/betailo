import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { LocaleService } from '../../../core/i18n/locale.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { IconComponent } from '../../../shared/ui/icon/icon.component';
import { EntryBadgesComponent } from './entry-badges.component';
import { RowActionsComponent } from './row-actions.component';
import { ReferenceStore } from './reference.store';
import { CatalogRef, TERM_LISTS, TermList } from './reference.types';

/** The plain vocabularies: forum categories, review highlights, report
 * reasons and the farm review checklist. */
@Component({
  selector: 'bt-lists-view',
  imports: [IconComponent, EntryBadgesComponent, RowActionsComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="grid gap-3 2xl:grid-cols-2">
      @for (list of lists(); track list.name) {
        <section class="bo-card overflow-hidden">
          <header class="flex flex-wrap items-center gap-2 border-b border-line px-4 py-3">
            <div class="flex min-w-0 flex-1 flex-col">
              <h3 class="text-body-strong">{{ 'adminReference.lists.' + list.name + '.title' | t }}</h3>
              <span class="text-caption text-ink-muted">{{ 'adminReference.lists.' + list.name + '.hint' | t }}</span>
            </div>
            <button type="button" class="inline-flex items-center gap-1 text-caption font-semibold text-forest hover:underline" (click)="store.editing.set({ mode: 'add', entity: 'term', ref: { entity: 'term', list: list.name } })">
              <bt-icon name="plus" [size]="14" /> {{ 'adminReference.addEntry' | t }}
            </button>
            <button type="button" class="inline-flex items-center gap-1 text-caption font-semibold text-forest hover:underline" (click)="bulkAdd(list.name)">
              <bt-icon name="upload" [size]="14" /> {{ 'adminReference.pasteMany' | t }}
            </button>
          </header>
          <ul>
            @for (term of list.terms; track term.key; let first = $first, last = $last, i = $index) {
              <li class="flex flex-wrap items-center gap-3 border-b border-line px-4 py-2 last:border-b-0" [class.opacity-60]="!term.active">
                <div class="flex min-w-0 flex-1 flex-col">
                  <span class="flex flex-wrap items-center gap-2 text-small">
                    <span class="font-semibold">{{ term.label[lang()] }}</span>
                    <bt-entry-badges [mark]="store.marks().get('term:' + list.name + ':' + term.key)" [active]="term.active" [usage]="usage(list.name, term.key)" />
                  </span>
                  <span class="text-caption text-ink-muted">{{ term.label[lang() === 'fr' ? 'en' : 'fr'] }} · <span class="font-mono">{{ term.key }}</span></span>
                </div>
                <bt-row-actions
                  [active]="term.active"
                  [first]="first"
                  [last]="last"
                  [busy]="store.busy()"
                  (edit)="store.editing.set({ mode: 'edit', entity: 'term', ref: ref(list.name, term.key) })"
                  (toggled)="toggle(ref(list.name, term.key), term.active)"
                  (moved)="move(ref(list.name, term.key), i + $event)"
                />
              </li>
            }
          </ul>
        </section>
      }
    </div>
  `
})
export class ListsViewComponent {
  protected readonly store = inject(ReferenceStore);
  private readonly locale = inject(LocaleService);


  protected readonly lang = computed(() => this.locale.locale());
  protected readonly lists = computed(() => TERM_LISTS.map((name) => ({ name, terms: this.store.draft()?.[name] ?? [] })));

  protected bulkAdd(list: TermList): void {
    this.store.bulking.set({ entity: 'term', list, where: this.locale.t(`adminReference.lists.${list}.title`) });
  }

  protected ref(list: TermList, key: string): CatalogRef {
    return { entity: 'term', list, key };
  }

  protected usage(list: TermList, key: string): number {
    return this.store.usage.value()?.[`${list}:${key}`] ?? 0;
  }

  protected toggle(ref: CatalogRef, active: boolean): void {
    void this.store.queue([{ kind: 'update', ref, value: { active: !active } }]);
  }

  protected move(ref: CatalogRef, position: number): void {
    void this.store.queue([{ kind: 'move', ref, position }]);
  }
}
