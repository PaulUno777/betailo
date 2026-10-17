import { ChangeDetectionStrategy, Component, computed, effect, inject, input, output, signal, untracked } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { AppError } from '../../../core/http/app-error';
import { LocaleService } from '../../../core/i18n/locale.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { cycleTypeLabelKey } from '../../../domain/reference/cycle-types';
import { CATEGORY_TONES, dotClass } from '../../../domain/reference/species';
import { countryName } from '../../../domain/reference/countries';
import { ButtonComponent } from '../../../shared/ui/button/button.component';
import { IconComponent } from '../../../shared/ui/icon/icon.component';
import { AnimalIconPickerComponent } from './animal-icon-picker.component';
import { JsonEditorComponent } from './json-editor.component';
import { JsonSchema, Problem } from './json-schema-tools';
import { changedFields, entryOf, nameOf, ownFields, slugify, usageKey } from './reference-ops';
import { ReferenceStore } from './reference.store';
import { CatalogOp, CatalogRef, Entity, EntryRequest, partOf } from './reference.types';

interface Form {
  key: string;
  autoKey: boolean;
  fr: string;
  en: string;
  icon: string;
  color: string;
  cycleType: string;
  unitKey: string;
  unitAutoKey: boolean;
  unitFr: string;
  unitEn: string;
  active: boolean;
  code: string;
  callingCode: string;
  parent: string;
  // A new category comes with its first kind.
  kindFr: string;
  kindEn: string;
  kindCycle: string;
  kindIcon: string;
  kindUnitFr: string;
  kindUnitEn: string;
}

/** Where an entry's schema sits in its reference's schema. */
const SCHEMA_PATH: Record<Entity, readonly string[]> = {
  category: ['categories'],
  subcategory: ['categories', 'subcategories'],
  country: ['countries'],
  region: ['countries', 'regions'],
  city: ['countries', 'regions', 'cities'],
  term: ['entries']
};

/** Adds one reference entry or edits one, as a pending change: in a
 * form, or as the entry's JSON. An edit sends only what changed (and a
 * move when its parent changes); its key never changes. */
@Component({
  selector: 'bt-entry-drawer',
  imports: [FormsModule, ButtonComponent, IconComponent, AnimalIconPickerComponent, JsonEditorComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(keydown.escape)': 'closed.emit()' },
  template: `
    <div class="fixed inset-0 z-40 bg-ink/30" (click)="closed.emit()" aria-hidden="true"></div>
    <aside
      role="dialog"
      aria-modal="true"
      [attr.aria-label]="title()"
      class="fixed inset-y-0 right-0 z-50 flex w-full flex-col bg-surface-raised sm:w-[480px]"
      style="box-shadow: var(--shadow-dialog)"
    >
      <header class="flex items-start gap-3 border-b border-line px-5 py-4">
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <span class="text-eyebrow text-ink-muted">{{ 'adminReference.entity.' + request().entity | t }}</span>
          <h2 class="text-heading-4">{{ title() }}</h2>
          @if (request().mode === 'edit') {
            <span class="font-mono text-caption text-ink-muted">{{ request().ref.key }} · {{ 'adminReference.usedBy' | t: { count: usage() } }}</span>
          }
        </div>
        <button type="button" class="flex h-9 w-9 items-center justify-center rounded-md hover:bg-surface-sunken" (click)="closed.emit()" [attr.aria-label]="'common.close' | t">
          <bt-icon name="x" [size]="18" />
        </button>
      </header>

      <div class="flex gap-1 border-b border-line px-5 pt-2" role="tablist">
        @for (t of ['form', 'json']; track t) {
          <button
            type="button"
            role="tab"
            [attr.aria-selected]="tab() === t"
            (click)="switchTab(t === 'form' ? 'form' : 'json')"
            class="-mb-px border-b-2 px-3 pb-2 text-small font-semibold"
            [class]="tab() === t ? 'border-forest text-ink' : 'border-transparent text-ink-muted hover:text-ink'"
          >
            {{ 'adminReference.drawerTab.' + t | t }}
          </button>
        }
      </div>

      <div class="flex-1 overflow-y-auto px-5 py-4">
        @if (tab() === 'form') {
          @let f = form();
          <div class="flex flex-col gap-4">
            @switch (request().entity) {
              @case ('country') {
                @if (request().mode === 'add') {
                  <label class="flex flex-col gap-1">
                    <span class="text-small font-semibold">{{ 'adminReference.field.code' | t }}</span>
                    <input class="bo-input uppercase" maxlength="2" placeholder="SN" [ngModel]="f.code" (ngModelChange)="set({ code: $event.toUpperCase() })" />
                    <span class="text-caption text-ink-muted">{{ f.code.length === 2 ? countryLabel(f.code) : ('adminReference.hint.code' | t) }}</span>
                  </label>
                }
                <label class="flex flex-col gap-1">
                  <span class="text-small font-semibold">{{ 'adminReference.field.callingCode' | t }}</span>
                  <input class="bo-input" inputmode="numeric" placeholder="221" [ngModel]="f.callingCode" (ngModelChange)="set({ callingCode: $event })" />
                </label>
              }
              @default {
                <div class="grid gap-3 sm:grid-cols-2">
                  <label class="flex flex-col gap-1">
                    <span class="text-small font-semibold">{{ 'adminReference.field.nameFr' | t }}</span>
                    <input class="bo-input" [ngModel]="f.fr" (ngModelChange)="setFr($event)" />
                  </label>
                  <label class="flex flex-col gap-1">
                    <span class="text-small font-semibold">{{ 'adminReference.field.nameEn' | t }}</span>
                    <input class="bo-input" [ngModel]="f.en" (ngModelChange)="set({ en: $event })" />
                  </label>
                </div>
                @if (request().mode === 'add') {
                  <label class="flex flex-col gap-1">
                    <span class="text-small font-semibold">{{ 'adminReference.field.key' | t }}</span>
                    <input class="bo-input font-mono" [ngModel]="f.key" (ngModelChange)="set({ key: $event.trim().toLowerCase(), autoKey: false })" />
                    <span class="text-caption text-ink-muted">{{ 'adminReference.hint.key' | t }}</span>
                  </label>
                }
              }
            }

            @if (request().entity === 'subcategory' || request().entity === 'city') {
              <label class="flex flex-col gap-1">
                <span class="text-small font-semibold">{{ (request().entity === 'city' ? 'adminReference.field.region' : 'adminReference.field.category') | t }}</span>
                <select class="bo-input" [ngModel]="f.parent" (ngModelChange)="set({ parent: $event })">
                  @for (p of parents(); track p.key) {
                    <option [value]="p.key">{{ p.label }}</option>
                  }
                </select>
                @if (request().mode === 'edit' && f.parent !== originalParent()) {
                  <span class="text-caption text-clay-ink">{{ 'adminReference.hint.moves' | t }}</span>
                }
              </label>
            }

            @if (request().entity === 'category' || request().entity === 'subcategory') {
              <div class="flex flex-col gap-1.5">
                <span class="text-small font-semibold">{{ 'adminReference.field.icon' | t }}</span>
                <bt-animal-icon-picker [value]="f.icon" (valueChange)="set({ icon: $event })" />
              </div>
            }
            @if (request().entity === 'category') {
              <div class="flex flex-col gap-1.5">
                <span class="text-small font-semibold">{{ 'adminReference.field.tone' | t }}</span>
                <div class="flex flex-wrap gap-1.5">
                  @for (tone of tones; track tone) {
                    <button
                      type="button"
                      (click)="set({ color: tone })"
                      class="inline-flex h-8 items-center gap-1.5 rounded-full border px-3 text-caption font-semibold"
                      [class]="f.color === tone ? 'border-ink bg-ink text-surface' : 'border-line-strong'"
                    >
                      <span class="h-3 w-3 rounded-full" [class]="dotClass(tone)"></span>{{ 'adminReference.tone.' + tone | t }}
                    </button>
                  }
                </div>
              </div>
            }
            @if (request().entity === 'subcategory') {
              <label class="flex flex-col gap-1">
                <span class="text-small font-semibold">{{ 'adminReference.field.cycleType' | t }}</span>
                <select class="bo-input" [ngModel]="f.cycleType" (ngModelChange)="set({ cycleType: $event })">
                  @for (type of cycleTypes(); track type) {
                    <option [value]="type">{{ cycleTypeLabel(type) | t }}</option>
                  }
                </select>
              </label>
              <div class="grid gap-3 sm:grid-cols-2">
                <label class="flex flex-col gap-1">
                  <span class="text-small font-semibold">{{ 'adminReference.field.unitFr' | t }}</span>
                  <input class="bo-input" placeholder="chick" [ngModel]="f.unitFr" (ngModelChange)="setUnitFr($event)" />
                </label>
                <label class="flex flex-col gap-1">
                  <span class="text-small font-semibold">{{ 'adminReference.field.unitEn' | t }}</span>
                  <input class="bo-input" placeholder="chick" [ngModel]="f.unitEn" (ngModelChange)="set({ unitEn: $event })" />
                </label>
              </div>
              <span class="-mt-2 text-caption text-ink-muted">{{ 'adminReference.hint.unit' | t: { key: f.unitKey || '…' } }}</span>
            }

            @if (request().entity === 'category' && request().mode === 'add') {
              <fieldset class="flex flex-col gap-3 rounded-lg border border-line p-3">
                <legend class="px-1 text-small font-semibold">{{ 'adminReference.firstKind' | t }}</legend>
                <div class="grid gap-3 sm:grid-cols-2">
                  <input class="bo-input" [attr.aria-label]="'adminReference.field.nameFr' | t" [placeholder]="'adminReference.field.nameFr' | t" [ngModel]="f.kindFr" (ngModelChange)="set({ kindFr: $event })" />
                  <input class="bo-input" [attr.aria-label]="'adminReference.field.nameEn' | t" [placeholder]="'adminReference.field.nameEn' | t" [ngModel]="f.kindEn" (ngModelChange)="set({ kindEn: $event })" />
                  <input class="bo-input" [attr.aria-label]="'adminReference.field.unitFr' | t" [placeholder]="'adminReference.field.unitFr' | t" [ngModel]="f.kindUnitFr" (ngModelChange)="set({ kindUnitFr: $event })" />
                  <input class="bo-input" [attr.aria-label]="'adminReference.field.unitEn' | t" [placeholder]="'adminReference.field.unitEn' | t" [ngModel]="f.kindUnitEn" (ngModelChange)="set({ kindUnitEn: $event })" />
                </div>
                <select class="bo-input" [attr.aria-label]="'adminReference.field.cycleType' | t" [ngModel]="f.kindCycle" (ngModelChange)="set({ kindCycle: $event })">
                  @for (type of cycleTypes(); track type) {
                    <option [value]="type">{{ cycleTypeLabel(type) | t }}</option>
                  }
                </select>
              </fieldset>
            }

            <label class="flex items-start gap-3 rounded-lg bg-surface-sunken p-3">
              <input type="checkbox" class="mt-0.5 h-5 w-5 accent-forest" [ngModel]="f.active" (ngModelChange)="set({ active: $event })" />
              <span class="flex flex-col gap-0.5">
                <span class="text-small font-semibold">{{ 'adminReference.field.active' | t }}</span>
                <span class="text-caption text-ink-muted">
                  {{ (!f.active && usage() ? 'adminReference.hint.deactivateUsed' : 'adminReference.hint.active') | t: { count: usage() } }}
                </span>
              </span>
            </label>
          </div>
        } @else {
          <p class="mb-2 text-caption text-ink-muted">{{ (request().mode === 'add' ? 'adminReference.hint.jsonAdd' : 'adminReference.hint.jsonEdit') | t }}</p>
          <!-- The code editor is only fetched when the JSON tab opens. -->
          @defer {
            <div class="overflow-hidden rounded-lg border border-line">
              <bt-json-editor class="h-[52vh]" [doc]="jsonText()" [schema]="entrySchema()" [label]="title()" (textChange)="jsonDraft.set($event)" (problems)="jsonProblems.set($event)" />
            </div>
          } @loading (minimum 200ms) {
            <div class="h-[52vh] animate-pulse rounded-lg bg-surface-sunken"></div>
          }
          @if (jsonErrors()) {
            <p class="mt-2 text-caption text-alert">{{ 'adminReference.errors' | t: { count: jsonErrors() } }}</p>
          }
        }

        @if (error(); as e) {
          <div class="mt-4 flex flex-col gap-1 rounded-md bg-alert-soft p-3 text-small text-alert" role="alert">
            <b>{{ e.message }}</b>
            @for (p of problemsOf(e); track $index) {
              <span class="font-mono text-caption">· {{ p }}</span>
            }
          </div>
        }
      </div>

      <footer class="flex items-center justify-end gap-2 border-t border-line px-5 py-3">
        <bt-button variant="outline" (click)="closed.emit()">{{ 'common.cancel' | t }}</bt-button>
        <bt-button variant="solid" [disabled]="store.busy() || (tab() === 'json' && jsonErrors() > 0)" (click)="submit()">
          {{ (request().mode === 'add' ? 'adminReference.queueAdd' : 'adminReference.queueEdit') | t }}
        </bt-button>
      </footer>
    </aside>
  `
})
export class EntryDrawerComponent {
  protected readonly store = inject(ReferenceStore);
  private readonly locale = inject(LocaleService);

  readonly request = input.required<EntryRequest>();
  readonly closed = output<void>();

  protected readonly tones = CATEGORY_TONES;
  protected readonly dotClass = dotClass;
  protected readonly cycleTypeLabel = cycleTypeLabelKey;
  protected readonly tab = signal<'form' | 'json'>('form');
  protected readonly form = signal<Form>(emptyForm());
  protected readonly error = signal<AppError | null>(null);
  protected readonly jsonText = signal('');
  protected readonly jsonDraft = signal('');
  protected readonly jsonProblems = signal<Problem[]>([]);
  protected readonly jsonErrors = computed(() => this.jsonProblems().filter((p) => p.severity === 'error').length);

  private readonly original = computed(() => {
    const r = this.request();
    return r.mode === 'edit' && r.ref.key ? entryOf(this.store.draft(), r.ref as CatalogRef) : undefined;
  });
  protected readonly originalParent = signal('');

  protected readonly usage = computed(() => {
    const r = this.request();
    if (r.mode !== 'edit' || !r.ref.key) return 0;
    const usage = this.store.usage.value() ?? {};
    if (r.entity === 'category') {
      const cat = this.store.draft()?.categories.find((c) => c.key === r.ref.key);
      return (cat?.subcategories ?? []).reduce((n, s) => n + (usage[`species:${s.key}`] ?? 0), 0);
    }
    return usage[usageKey(r.ref as CatalogRef)] ?? 0;
  });

  protected readonly title = computed(() => {
    const r = this.request();
    const lang = this.locale.locale();
    if (r.mode === 'edit') return nameOf(this.original(), lang, r.ref as CatalogRef);
    const where = this.parentName();
    return this.locale.t('adminReference.newEntry.' + r.entity, { where });
  });

  protected readonly parents = computed(() => {
    const r = this.request();
    const d = this.store.draft();
    const lang = this.locale.locale();
    if (!d) return [];
    if (r.entity === 'subcategory') return d.categories.map((c) => ({ key: c.key, label: c.label[lang] }));
    if (r.entity === 'city') {
      return (d.countries.find((k) => k.code === r.ref.country)?.regions ?? []).map((x) => ({ key: x.key, label: x.label[lang] }));
    }
    return [];
  });

  protected readonly entrySchema = computed<JsonSchema | null>(() => {
    const r = this.request();
    const root = this.store.schemaOf(partOf(r.ref as CatalogRef));
    if (!root) return null;
    let s: JsonSchema | undefined = root;
    for (const name of SCHEMA_PATH[r.entity]) s = s?.properties?.[name]?.items;
    if (!s || r.mode === 'add') return s ?? null;
    // An edit carries the entry's own fields: no key, no children.
    const drop = ['key', 'code', 'subcategories', 'regions', 'cities'];
    const properties = Object.fromEntries(Object.entries(s.properties ?? {}).filter(([k]) => !drop.includes(k)));
    return { ...s, properties, required: (s.required ?? []).filter((k) => !drop.includes(k)) };
  });

  constructor() {
    effect(() => {
      const r = this.request();
      untracked(() => this.reset(r));
    });
  }

  protected cycleTypes(): readonly string[] {
    return this.store.draft()?.cycleTypes ?? [];
  }

  protected countryLabel(code: string): string {
    return countryName(code, this.locale.locale());
  }

  private parentName(): string {
    const r = this.request();
    const d = this.store.draft();
    const lang = this.locale.locale();
    switch (r.entity) {
      case 'subcategory':
        return d?.categories.find((c) => c.key === r.ref.parent)?.label[lang] ?? '';
      case 'region':
        return countryName(r.ref.country ?? '', lang);
      case 'city':
        return d?.countries.find((k) => k.code === r.ref.country)?.regions?.find((x) => x.key === r.ref.parent)?.label[lang] ?? '';
      case 'term':
        return this.locale.t(`adminReference.lists.${r.ref.list}.title`);
      default:
        return '';
    }
  }

  protected set(change: Partial<Form>): void {
    this.form.update((f) => ({ ...f, ...change }));
    this.error.set(null);
  }

  protected setFr(fr: string): void {
    const f = this.form();
    this.set({ fr, ...(this.request().mode === 'add' && f.autoKey ? { key: slugify(fr) } : {}) });
  }

  protected setUnitFr(unitFr: string): void {
    const f = this.form();
    this.set({ unitFr, ...(this.request().mode === 'add' && f.unitAutoKey ? { unitKey: slugify(unitFr) } : {}) });
  }

  protected switchTab(tab: 'form' | 'json'): void {
    if (tab === this.tab()) return;
    if (tab === 'json') {
      const text = JSON.stringify(this.valueOf(this.form()), null, 2);
      this.jsonText.set(text);
      this.jsonDraft.set(text);
    } else {
      try {
        this.form.set(this.formOf(JSON.parse(this.jsonDraft()), this.form()));
      } catch {
        /* the JSON doesn't parse: keep the form as it was */
      }
    }
    this.tab.set(tab);
  }

  protected problemsOf(e: AppError): string[] {
    const p = e.params?.['problems'];
    return Array.isArray(p) ? p.map(String) : [];
  }

  protected async submit(): Promise<void> {
    let value: Record<string, unknown>;
    if (this.tab() === 'json') {
      try {
        value = JSON.parse(this.jsonDraft());
      } catch {
        return;
      }
    } else {
      value = this.valueOf(this.form());
    }
    const ops = this.opsFor(value);
    if (!ops.length) {
      this.closed.emit();
      return;
    }
    const refused = await this.store.queue(ops);
    if (refused) this.error.set(refused);
    else this.closed.emit();
  }

  /** The operations the drawer's content makes: one add, or an update
   * of what changed plus a move when the parent changed. */
  private opsFor(value: Record<string, unknown>): CatalogOp[] {
    const r = this.request();
    if (r.mode === 'add') {
      const ref = { ...r.ref, entity: r.entity, key: '', parent: this.form().parent || r.ref.parent } as CatalogRef;
      return [{ kind: 'add', ref, value }];
    }
    const ref = r.ref as CatalogRef;
    const ops: CatalogOp[] = [];
    const original = this.original();
    const changed = original ? changedFields(ownFields(r.entity, original), ownFields(r.entity, value)) : null;
    if (changed) ops.push({ kind: 'update', ref, value: changed });
    const parent = this.form().parent;
    if ((r.entity === 'subcategory' || r.entity === 'city') && parent && parent !== this.originalParent()) {
      ops.push({ kind: 'move', ref: { ...ref, parent }, position: 9999 });
    }
    return ops;
  }

  /** The entry the form describes, as the API takes it. */
  private valueOf(f: Form): Record<string, unknown> {
    const label = { fr: f.fr.trim(), en: f.en.trim() || f.fr.trim() };
    const add = this.request().mode === 'add';
    switch (this.request().entity) {
      case 'category':
        return {
          ...(add ? { key: f.key } : {}),
          label,
          icon: f.icon,
          color: f.color,
          active: f.active,
          ...(add
            ? {
                subcategories: [
                  {
                    key: slugify(f.kindFr),
                    label: { fr: f.kindFr.trim(), en: f.kindEn.trim() || f.kindFr.trim() },
                    cycleType: f.kindCycle,
                    icon: f.icon,
                    unit: { key: slugify(f.kindUnitFr), label: { fr: f.kindUnitFr.trim(), en: f.kindUnitEn.trim() || f.kindUnitFr.trim() } },
                    active: true
                  }
                ]
              }
            : {})
        };
      case 'subcategory':
        return {
          ...(add ? { key: f.key } : {}),
          label,
          cycleType: f.cycleType,
          icon: f.icon,
          unit: { key: f.unitKey, label: { fr: f.unitFr.trim(), en: f.unitEn.trim() || f.unitFr.trim() } },
          active: f.active
        };
      case 'country':
        return { ...(add ? { code: f.code } : {}), callingCode: f.callingCode.trim(), active: f.active };
      case 'region':
        return { ...(add ? { key: f.key } : {}), label, active: f.active, ...(add ? { cities: [] } : {}) };
      default:
        return { ...(add ? { key: f.key } : {}), label, active: f.active };
    }
  }

  /** Back from JSON to the form: what it can show. */
  private formOf(v: Record<string, unknown>, f: Form): Form {
    const label = (v['label'] ?? {}) as { fr?: string; en?: string };
    const unit = (v['unit'] ?? {}) as { key?: string; label?: { fr?: string; en?: string } };
    return {
      ...f,
      key: typeof v['key'] === 'string' ? v['key'] : f.key,
      fr: label.fr ?? f.fr,
      en: label.en ?? f.en,
      icon: typeof v['icon'] === 'string' ? v['icon'] : f.icon,
      color: typeof v['color'] === 'string' ? v['color'] : f.color,
      cycleType: typeof v['cycleType'] === 'string' ? v['cycleType'] : f.cycleType,
      unitKey: unit.key ?? f.unitKey,
      unitFr: unit.label?.fr ?? f.unitFr,
      unitEn: unit.label?.en ?? f.unitEn,
      active: typeof v['active'] === 'boolean' ? v['active'] : f.active,
      code: typeof v['code'] === 'string' ? v['code'] : f.code,
      callingCode: typeof v['callingCode'] === 'string' ? v['callingCode'] : f.callingCode
    };
  }

  private reset(r: EntryRequest): void {
    this.tab.set('form');
    this.error.set(null);
    const f = emptyForm();
    f.parent = r.ref.parent ?? '';
    const d = this.store.draft();
    if (r.mode === 'edit') {
      const e = this.original() as Record<string, unknown> | undefined;
      if (e) Object.assign(f, this.formOf(e, f), { autoKey: false, unitAutoKey: false });
      if (r.entity === 'subcategory') f.parent = d?.categories.find((c) => c.subcategories.some((s) => s.key === r.ref.key))?.key ?? '';
      if (r.entity === 'city') {
        f.parent = d?.countries.find((k) => k.code === r.ref.country)?.regions?.find((x) => x.cities?.some((c) => c.key === r.ref.key))?.key ?? '';
      }
    } else if (r.entity === 'subcategory') {
      f.icon = d?.categories.find((c) => c.key === r.ref.parent)?.icon ?? f.icon;
    }
    this.originalParent.set(f.parent);
    this.form.set(f);
  }
}

function emptyForm(): Form {
  return {
    key: '',
    autoKey: true,
    fr: '',
    en: '',
    icon: 'hen',
    color: 'stone',
    cycleType: 'rearing',
    unitKey: '',
    unitAutoKey: true,
    unitFr: '',
    unitEn: '',
    active: true,
    code: '',
    callingCode: '',
    parent: '',
    kindFr: '',
    kindEn: '',
    kindCycle: 'rearing',
    kindIcon: 'hen',
    kindUnitFr: '',
    kindUnitEn: ''
  };
}
