import { ChangeDetectionStrategy, Component, computed, inject, signal, viewChild } from '@angular/core';
import { AppError } from '../../../core/http/app-error';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { CatalogPart, partContent } from '../../../domain/reference/catalog';
import { ButtonComponent } from '../../../shared/ui/button/button.component';
import { JsonEditorComponent } from './json-editor.component';
import { Problem } from './json-schema-tools';
import { ReferenceStore } from './reference.store';

/** For experts: one reference of the draft at a time as JSON, checked
 * against that reference's own schema as you type. Export it, or edit
 * it and queue the result — the API turns the differences into ordinary
 * changes (what you leave out is deactivated, never removed), reviewed
 * like any other. */
@Component({
  selector: 'bt-advanced-view',
  imports: [ButtonComponent, JsonEditorComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-3">
      <p class="text-small text-ink-muted">{{ 'adminReference.advancedHint' | t }}</p>
      <div class="flex flex-wrap gap-1.5" role="tablist" [attr.aria-label]="'adminReference.reference' | t">
        @for (name of store.parts; track name) {
          <button
            type="button"
            role="tab"
            [attr.aria-selected]="part() === name"
            (click)="part.set(name)"
            class="inline-flex h-8 items-center gap-1.5 rounded-full border px-3 text-caption font-semibold"
            [class]="part() === name ? 'border-ink bg-ink text-surface' : 'border-line-strong text-ink hover:border-ink'"
          >
            {{ 'adminReference.part.' + name | t }}
            <span class="font-mono opacity-70">v{{ store.changeset()?.baseVersions?.[name] ?? 0 }}</span>
          </button>
        }
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <span class="inline-flex h-8 items-center rounded-full px-3 text-caption font-semibold" [class]="errors() ? 'bg-alert-soft text-alert' : 'bg-forest-soft text-on-forest-soft'">
          {{ errors() ? ('adminReference.errors' | t: { count: errors() }) : ('adminReference.valid' | t) }}
        </span>
        <span class="ml-auto"></span>
        <bt-button variant="outline" size="sm" (click)="editor()?.format()">{{ 'adminReference.format' | t }}</bt-button>
        <bt-button variant="outline" size="sm" (click)="download()">{{ 'adminReference.export' | t }}</bt-button>
        <bt-button variant="outline" size="sm" [disabled]="!edited()" (click)="editor()?.reset(original())">{{ 'adminReference.revertJson' | t }}</bt-button>
        <bt-button variant="solid" size="sm" [disabled]="!edited() || errors() > 0 || store.busy()" (click)="queue()">{{ 'adminReference.queueJson' | t }}</bt-button>
      </div>
      <div class="overflow-hidden rounded-xl border border-line">
        <bt-json-editor
          class="h-[calc(100vh-380px)] min-h-[420px]"
          [doc]="original()"
          [schema]="store.schemaOf(part())"
          [label]="'adminReference.part.' + part() | t"
          (textChange)="text.set($event)"
          (problems)="problems.set($event)"
        />
      </div>
      @if (problems().length) {
        <ul class="bo-card max-h-48 overflow-auto">
          @for (p of problems(); track $index) {
            <li>
              <button type="button" class="flex w-full gap-3 px-4 py-1.5 text-left text-small hover:bg-surface-sunken" (click)="editor()?.goTo(p.from, p.to)">
                <span class="w-12 shrink-0 font-mono text-caption text-ink-muted">L{{ p.line }}</span>
                <span [class.text-alert]="p.severity === 'error'">{{ p.message }}</span>
              </button>
            </li>
          }
        </ul>
      }
      @if (error(); as e) {
        <p class="rounded-md bg-alert-soft p-3 text-small text-alert" role="alert">{{ e.message }}</p>
      }
    </div>
  `
})
export class AdvancedViewComponent {
  protected readonly store = inject(ReferenceStore);
  protected readonly editor = viewChild(JsonEditorComponent);

  protected readonly part = signal<CatalogPart>('animals');
  protected readonly original = computed(() => {
    const draft = this.store.draft();
    return draft ? JSON.stringify(partContent(draft, this.part()), null, 2) : '';
  });
  protected readonly text = signal('');
  protected readonly problems = signal<Problem[]>([]);
  protected readonly error = signal<AppError | null>(null);
  protected readonly errors = computed(() => this.problems().filter((p) => p.severity === 'error').length);
  protected readonly edited = computed(() => !!this.text() && this.text() !== this.original());

  protected download(): void {
    const blob = new Blob([this.text() || this.original()], { type: 'application/json' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = `betailo-${this.part()}-v${this.store.changeset()?.baseVersions?.[this.part()] ?? 0}-draft.json`;
    a.click();
    URL.revokeObjectURL(a.href);
  }

  protected async queue(): Promise<void> {
    let content: unknown;
    try {
      content = JSON.parse(this.text());
    } catch {
      return;
    }
    this.error.set(await this.store.importPart(this.part(), content));
  }
}
