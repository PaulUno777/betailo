import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { LocaleService } from '../../../core/i18n/locale.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { countryName } from '../../../domain/reference/countries';
import { IconComponent } from '../../../shared/ui/icon/icon.component';
import { EntryBadgesComponent } from './entry-badges.component';
import { RowActionsComponent } from './row-actions.component';
import { ReferenceStore } from './reference.store';
import { CatalogRef } from './reference.types';

/** Where farms can be: the countries on the left (with their regions
 * and farms counted), the chosen country's regions on the right, each
 * with its towns. A town is edited, moved to another region or
 * deactivated from its chip. */
@Component({
  selector: 'bt-places-view',
  imports: [FormsModule, IconComponent, EntryBadgesComponent, RowActionsComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="grid gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
      <aside class="bo-card flex max-h-[70vh] flex-col self-start overflow-hidden">
        <div class="border-b border-line p-2">
          <input class="bo-input" type="search" [placeholder]="'adminReference.searchCountries' | t" [ngModel]="countryQuery()" (ngModelChange)="countryQuery.set($event)" />
        </div>
        <ul class="flex-1 overflow-y-auto py-1" role="listbox" [attr.aria-label]="'adminReference.entity.country' | t">
          @for (k of countries(); track k.code) {
            <li>
              <button
                type="button"
                role="option"
                [attr.aria-selected]="k.code === selected()"
                (click)="selected.set(k.code)"
                class="flex w-full items-center gap-2 px-3 py-2 text-left text-small"
                [class]="k.code === selected() ? 'bg-forest-soft text-on-forest-soft' : 'hover:bg-surface-sunken'"
                [class.opacity-60]="!k.active"
              >
                <span class="flex-1 truncate font-semibold">{{ k.name }}</span>
                <span class="font-mono text-caption text-ink-muted">{{ k.code }}</span>
                @if (k.regions) {
                  <span class="rounded-full bg-surface-sunken px-1.5 font-mono text-[11px]">{{ k.regions }}</span>
                }
              </button>
            </li>
          }
        </ul>
        <button type="button" class="flex items-center gap-1.5 border-t border-line px-3 py-2.5 text-caption font-semibold text-forest hover:bg-surface-sunken" (click)="store.editing.set({ mode: 'add', entity: 'country', ref: { entity: 'country' } })">
          <bt-icon name="plus" [size]="14" /> {{ 'adminReference.addCountry' | t }}
        </button>
      </aside>

      @if (country(); as k) {
        <div class="flex min-w-0 flex-col gap-3">
          <section class="bo-card flex flex-wrap items-center gap-3 px-4 py-3">
            <div class="flex min-w-0 flex-1 flex-col">
              <span class="flex flex-wrap items-center gap-2">
                <b class="text-heading-4">{{ name(k.code) }}</b>
                <bt-entry-badges [mark]="mark('country:' + k.code)" [active]="k.active" [usage]="usage('country:' + k.code)" />
              </span>
              <span class="text-caption text-ink-muted">{{ k.code }} · +{{ k.callingCode }} · {{ 'adminReference.regionsCount' | t: { count: k.regions?.length ?? 0 } }}</span>
            </div>
            <button type="button" class="h-8 rounded-md px-2 text-caption font-semibold text-forest hover:bg-forest-soft" (click)="store.editing.set({ mode: 'edit', entity: 'country', ref: { entity: 'country', key: k.code } })">
              {{ 'adminReference.edit' | t }}
            </button>
          </section>

          <div class="flex flex-wrap items-center gap-2">
            <input class="bo-input min-w-56 flex-1" type="search" [placeholder]="'adminReference.searchPlaces' | t" [ngModel]="query()" (ngModelChange)="query.set($event)" />
            <button type="button" class="inline-flex h-9 items-center gap-1.5 rounded-full bg-forest px-4 text-caption font-semibold text-on-forest" (click)="store.editing.set({ mode: 'add', entity: 'region', ref: { entity: 'region', country: k.code } })">
              <bt-icon name="plus" [size]="14" /> {{ 'adminReference.addRegion' | t }}
            </button>
            <button type="button" class="inline-flex h-9 items-center gap-1.5 rounded-full border border-line-strong px-4 text-caption font-semibold" (click)="store.bulking.set({ entity: 'region', country: k.code, where: name(k.code) })">
              <bt-icon name="upload" [size]="14" /> {{ 'adminReference.pasteMany' | t }}
            </button>
          </div>

          @if (!k.regions?.length) {
            <p class="bo-card px-4 py-6 text-small text-ink-muted">{{ 'adminReference.noRegions' | t }}</p>
          }
          @for (r of regions(); track r.key; let first = $first, last = $last) {
            <section class="bo-card overflow-hidden" [class.opacity-70]="!r.active">
              <div class="flex flex-wrap items-center gap-3 border-b border-line px-4 py-2.5">
                <div class="flex min-w-0 flex-1 flex-col">
                  <span class="flex flex-wrap items-center gap-2">
                    <b class="text-body-strong">{{ r.label[lang()] }}</b>
                    <bt-entry-badges [mark]="mark('region:' + k.code + ':' + r.key)" [active]="r.active" [usage]="usage('region:' + k.code + ':' + r.key)" />
                  </span>
                  <span class="text-caption text-ink-muted">{{ r.label[otherLang()] }} · <span class="font-mono">{{ r.key }}</span></span>
                </div>
                <bt-row-actions
                  [active]="r.active"
                  [first]="first"
                  [last]="last"
                  [busy]="store.busy()"
                  (edit)="store.editing.set({ mode: 'edit', entity: 'region', ref: regionRef(k.code, r.key) })"
                  (toggled)="toggle(regionRef(k.code, r.key), r.active)"
                  (moved)="move(regionRef(k.code, r.key), r.index + $event)"
                />
              </div>
              <div class="flex flex-wrap gap-1.5 px-4 py-3">
                @for (c of r.cities; track c.key) {
                  <button
                    type="button"
                    (click)="store.editing.set({ mode: 'edit', entity: 'city', ref: { entity: 'city', country: k.code, key: c.key } })"
                    class="inline-flex h-8 items-center gap-1.5 rounded-full border px-3 text-caption font-semibold transition hover:border-forest"
                    [class]="chipClass(k.code, c.key, c.active)"
                    [attr.title]="c.label[otherLang()] + ' · ' + c.key"
                  >
                    {{ c.label[lang()] }}
                    @if (usage('city:' + k.code + ':' + c.key); as n) {
                      <span class="font-mono text-[10.5px] opacity-70">{{ n }}</span>
                    }
                  </button>
                } @empty {
                  <span class="text-caption text-ink-muted">{{ 'adminReference.noTowns' | t }}</span>
                }
                <button type="button" class="inline-flex h-8 items-center gap-1 rounded-full border border-dashed border-line-strong px-3 text-caption font-semibold text-forest" (click)="store.editing.set({ mode: 'add', entity: 'city', ref: { entity: 'city', country: k.code, parent: r.key } })">
                  <bt-icon name="plus" [size]="12" /> {{ 'adminReference.addTown' | t }}
                </button>
                <button type="button" class="inline-flex h-8 items-center gap-1 rounded-full border border-dashed border-line-strong px-3 text-caption font-semibold text-forest" (click)="store.bulking.set({ entity: 'city', country: k.code, parent: r.key, where: r.label[lang()] })">
                  <bt-icon name="upload" [size]="12" /> {{ 'adminReference.pasteMany' | t }}
                </button>
              </div>
            </section>
          }
        </div>
      }
    </div>
  `
})
export class PlacesViewComponent {
  protected readonly store = inject(ReferenceStore);
  private readonly locale = inject(LocaleService);


  protected readonly selected = signal('CM');
  protected readonly countryQuery = signal('');
  protected readonly query = signal('');
  protected readonly lang = computed(() => this.locale.locale());
  protected readonly otherLang = computed(() => (this.lang() === 'fr' ? 'en' : 'fr'));

  /** Countries with regions first, then by name in the reader's language. */
  protected readonly countries = computed(() => {
    const q = fold(this.countryQuery());
    return (this.store.draft()?.countries ?? [])
      .map((k) => ({ code: k.code, name: this.name(k.code), active: k.active, regions: k.regions?.length ?? 0 }))
      .filter((k) => !q || fold(`${k.code} ${k.name}`).includes(q))
      .sort((a, b) => b.regions - a.regions || a.name.localeCompare(b.name, this.lang()));
  });

  protected readonly country = computed(() => this.store.draft()?.countries.find((k) => k.code === this.selected()));

  /** The country's regions, each with the towns matching the search. */
  protected readonly regions = computed(() => {
    const q = fold(this.query());
    const k = this.country();
    return (k?.regions ?? [])
      .map((r, index) => {
        const regionHit = !q || fold(`${r.key} ${r.label.fr} ${r.label.en}`).includes(q);
        const cities = (r.cities ?? []).filter((c) => regionHit || fold(`${c.key} ${c.label.fr} ${c.label.en}`).includes(q));
        return { ...r, index, cities, show: regionHit || cities.length > 0 };
      })
      .filter((r) => r.show);
  });

  protected name(code: string): string {
    return countryName(code, this.lang());
  }

  protected regionRef(country: string, key: string): CatalogRef {
    return { entity: 'region', country, key };
  }

  protected mark(id: string) {
    return this.store.marks().get(id);
  }

  protected usage(key: string): number {
    return this.store.usage.value()?.[key] ?? 0;
  }

  protected chipClass(country: string, key: string, active: boolean): string {
    const mark = this.store.marks().get(`city:${country}:${key}`);
    if (mark === 'added') return 'border-forest bg-forest-soft text-on-forest-soft';
    if (mark) return 'border-clay bg-clay-soft text-clay-ink';
    return active ? 'border-line-strong bg-surface-card text-ink' : 'border-line bg-surface-sunken text-ink-muted line-through';
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
