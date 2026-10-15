import { ChangeDetectionStrategy, Component, computed, inject, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { AppError } from '../../../core/http/app-error';
import { LocaleService } from '../../../core/i18n/locale.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { cycleTypeLabelKey } from '../../../domain/reference/cycle-types';
import { ButtonComponent } from '../../../shared/ui/button/button.component';
import { IconComponent } from '../../../shared/ui/icon/icon.component';
import { AnimalIconPickerComponent } from './animal-icon-picker.component';
import { parsePaste, slugify } from './reference-ops';
import { ReferenceStore } from './reference.store';
import { BulkTarget, CatalogOp, CatalogRef } from './reference.types';

/** Adds many entries at once: paste one per line ("Nom ; Name", or
 * "clé ; nom ; name" from a spreadsheet), check the preview — keys
 * made from the names, duplicates and keys already taken flagged —
 * then queue them as one batch of changes. */
@Component({
  selector: 'bt-bulk-add-dialog',
  imports: [FormsModule, ButtonComponent, IconComponent, AnimalIconPickerComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(keydown.escape)': 'closed.emit()' },
  template: `
    <div class="fixed inset-0 z-40 bg-ink/30" (click)="closed.emit()" aria-hidden="true"></div>
    <section
      role="dialog"
      aria-modal="true"
      [attr.aria-label]="title()"
      class="fixed left-1/2 top-1/2 z-50 flex max-h-[90vh] w-[min(760px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 flex-col rounded-2xl bg-surface-raised"
      style="box-shadow: var(--shadow-dialog)"
    >
      <header class="flex items-start gap-3 border-b border-line px-5 py-4">
        <div class="flex flex-1 flex-col gap-0.5">
          <h2 class="text-heading-4">{{ title() }}</h2>
          <p class="text-small text-ink-muted">{{ 'adminReference.bulk.hint' | t }}</p>
        </div>
        <button type="button" class="flex h-9 w-9 items-center justify-center rounded-md hover:bg-surface-sunken" (click)="closed.emit()" [attr.aria-label]="'common.close' | t">
          <bt-icon name="x" [size]="18" />
        </button>
      </header>

      <div class="grid flex-1 gap-4 overflow-y-auto px-5 py-4 md:grid-cols-2">
        <div class="flex flex-col gap-3">
          <textarea
            class="bo-input h-56 resize-y py-2 font-mono"
            [placeholder]="'adminReference.bulk.placeholder.' + target().entity | t"
            [attr.aria-label]="'adminReference.bulk.paste' | t"
            [ngModel]="text()"
            (ngModelChange)="text.set($event); error.set(null)"
          ></textarea>
          @if (target().entity === 'subcategory') {
            <label class="flex flex-col gap-1">
              <span class="text-small font-semibold">{{ 'adminReference.field.cycleType' | t }}</span>
              <select class="bo-input" [ngModel]="cycleType()" (ngModelChange)="cycleType.set($event)">
                @for (type of cycleTypes(); track type) {
                  <option [value]="type">{{ cycleTypeLabel(type) | t }}</option>
                }
              </select>
            </label>
            <div class="flex flex-col gap-1.5">
              <span class="text-small font-semibold">{{ 'adminReference.field.icon' | t }}</span>
              <bt-animal-icon-picker [value]="icon()" (valueChange)="icon.set($event)" />
            </div>
            <p class="text-caption text-ink-muted">{{ 'adminReference.bulk.unitHint' | t }}</p>
          }
        </div>

        <div class="flex min-w-0 flex-col gap-2">
          <span class="text-small font-semibold">{{ 'adminReference.bulk.preview' | t: { count: rows().length } }}</span>
          <div class="overflow-auto rounded-lg border border-line">
            <table class="bo-table">
              <thead>
                <tr>
                  <th class="bo-th pt-2">{{ 'adminReference.field.key' | t }}</th>
                  <th class="bo-th pt-2">FR</th>
                  <th class="bo-th pt-2">EN</th>
                </tr>
              </thead>
              <tbody>
                @for (row of rows(); track $index) {
                  <tr [class.bg-alert-soft]="row.issue">
                    <td class="bo-td font-mono text-caption">{{ row.key || '—' }}</td>
                    <td class="bo-td">{{ row.fr }}</td>
                    <td class="bo-td">
                      {{ row.en }}
                      @if (row.issue) {
                        <span class="block text-caption text-alert">{{ 'adminReference.bulk.issue.' + row.issue | t }}</span>
                      }
                    </td>
                  </tr>
                } @empty {
                  <tr>
                    <td class="bo-td text-ink-muted" colspan="3">{{ 'adminReference.bulk.empty' | t }}</td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
          @if (error(); as e) {
            <p class="rounded-md bg-alert-soft p-2 text-small text-alert" role="alert">
              {{ e.message }}
              @if (failedLine(); as line) {
                · {{ 'adminReference.bulk.atLine' | t: { line } }}
              }
            </p>
          }
        </div>
      </div>

      <footer class="flex items-center justify-end gap-2 border-t border-line px-5 py-3">
        <bt-button variant="outline" (click)="closed.emit()">{{ 'common.cancel' | t }}</bt-button>
        <bt-button variant="solid" [disabled]="!canQueue()" (click)="submit()">
          {{ 'adminReference.bulk.queue' | t: { count: rows().length } }}
        </bt-button>
      </footer>
    </section>
  `
})
export class BulkAddDialogComponent {
  protected readonly store = inject(ReferenceStore);
  private readonly locale = inject(LocaleService);

  readonly target = input.required<BulkTarget>();
  readonly closed = output<void>();

  protected readonly cycleTypeLabel = cycleTypeLabelKey;
  protected readonly text = signal('');
  protected readonly cycleType = signal('elevage');
  protected readonly icon = signal('poule');
  protected readonly error = signal<AppError | null>(null);

  protected readonly title = computed(() => this.locale.t('adminReference.bulk.title.' + this.target().entity, { where: this.target().where }));

  /** Keys already in use where the new entries go (the API counts kinds
   * against every category, towns against the whole country). */
  private readonly taken = computed(() => {
    const t = this.target();
    const d = this.store.draft();
    if (!d) return new Set<string>();
    switch (t.entity) {
      case 'subcategory':
        return new Set(d.categories.flatMap((c) => [c.key, ...c.subcategories.map((s) => s.key)]));
      case 'region':
        return new Set((d.countries.find((k) => k.code === t.country)?.regions ?? []).map((r) => r.key));
      case 'city':
        return new Set((d.countries.find((k) => k.code === t.country)?.regions ?? []).flatMap((r) => (r.cities ?? []).map((c) => c.key)));
      case 'term':
        return new Set((d[t.list!] ?? []).map((x) => x.key));
    }
  });

  protected readonly rows = computed(() => {
    const seen = new Set<string>();
    return parsePaste(this.text()).map((row) => {
      let issue: 'key' | 'duplicate' | 'taken' | null = null;
      if (!/^[a-z0-9]+([_-][a-z0-9]+)*$/.test(row.key)) issue = 'key';
      else if (seen.has(row.key)) issue = 'duplicate';
      else if (this.taken().has(row.key)) issue = 'taken';
      seen.add(row.key);
      return { ...row, issue };
    });
  });

  protected readonly canQueue = computed(() => this.rows().length > 0 && !this.rows().some((r) => r.issue) && !this.store.busy());

  /** The pasted line the API refused, from its index in the batch. */
  protected readonly failedLine = computed(() => {
    const i = this.error()?.params?.['index'];
    return typeof i === 'number' ? i + 1 : null;
  });

  protected cycleTypes(): readonly string[] {
    return this.store.draft()?.cycleTypes ?? [];
  }

  protected async submit(): Promise<void> {
    const t = this.target();
    const ref: CatalogRef = { entity: t.entity, key: '', country: t.country, parent: t.parent, list: t.list };
    const ops: CatalogOp[] = this.rows().map((row) => {
      const label = { fr: row.fr, en: row.en };
      const value: Record<string, unknown> =
        t.entity === 'subcategory'
          ? {
              key: row.key,
              label,
              cycleType: this.cycleType(),
              icon: this.icon(),
              unit: { key: slugify(row.fr), label: { fr: row.fr.toLowerCase(), en: row.en.toLowerCase() } },
              active: true
            }
          : t.entity === 'region'
            ? { key: row.key, label, active: true, cities: [] }
            : { key: row.key, label, active: true };
      return { kind: 'add', ref, value };
    });
    const refused = await this.store.queue(ops);
    if (refused) this.error.set(refused);
    else this.closed.emit();
  }
}
