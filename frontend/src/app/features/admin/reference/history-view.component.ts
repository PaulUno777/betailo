import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { ApiClient } from '../../../core/http/api-client';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { CatalogPart, partContent } from '../../../domain/reference/catalog';
import { ButtonComponent } from '../../../shared/ui/button/button.component';
import { LocalDatePipe } from '../../../shared/pipes/local-date.pipe';
import { JsonDiffComponent } from './json-diff.component';
import { ReferenceStore } from './reference.store';
import { CatalogVersion } from './reference.types';

/** Every published version of every reference, newest first, filtered
 * by reference: compare one with the draft, or restore it — which
 * queues the changes that bring that reference back to it (what came
 * later is deactivated, never removed). */
@Component({
  selector: 'bt-history-view',
  imports: [ButtonComponent, JsonDiffComponent, TranslatePipe, LocalDatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mb-3 flex flex-wrap gap-1.5">
      @for (name of filters; track name) {
        <button
          type="button"
          (click)="filter.set(name)"
          class="h-8 rounded-full border px-3 text-caption font-semibold"
          [class]="filter() === name ? 'border-ink bg-ink text-surface' : 'border-line-strong text-ink hover:border-ink'"
        >
          {{ (name === 'all' ? 'adminReference.filter.all' : 'adminReference.part.' + name) | t }}
        </button>
      }
    </div>
    <div class="grid gap-4 lg:grid-cols-[340px_minmax(0,1fr)]">
      <ul class="bo-card max-h-[75vh] self-start overflow-y-auto">
        @for (v of versions(); track v.name + v.version) {
          <li class="flex flex-col gap-1 border-b border-line px-4 py-3 last:border-b-0" [class.bg-surface-sunken]="isCompared(v)">
            <span class="flex items-center gap-2 text-small font-semibold">
              {{ 'adminReference.part.' + v.name | t }} · v{{ v.version }}
              @if (v.version === inForce(v.name)) {
                <span class="rounded-full bg-forest-soft px-2 py-0.5 text-caption text-on-forest-soft">{{ 'adminReference.inForce' | t }}</span>
              }
              <span class="ml-auto text-caption font-normal text-ink-muted">{{ v.publishedAt | localDate: 'dayMonthYear' }}</span>
            </span>
            <span class="text-caption text-ink-muted">{{ v.fromSeed && !v.note ? ('adminReference.fromSeed' | t) : v.note }}</span>
            <span class="mt-1 flex gap-2">
              <bt-button variant="outline" size="sm" (click)="compare(v)">{{ 'adminReference.compare' | t }}</bt-button>
              @if (v.version !== inForce(v.name)) {
                <bt-button variant="outline" size="sm" [disabled]="store.busy()" (click)="store.restore(v.name, v.version)">{{ 'adminReference.restoreVersion' | t }}</bt-button>
              }
            </span>
          </li>
        }
      </ul>
      <div class="flex min-w-0 flex-col gap-2">
        @if (compared(); as c) {
          <p class="text-small text-ink-muted">{{ 'adminReference.comparing' | t: { reference: ('adminReference.part.' + c.name | t), version: c.version } }}</p>
          <div class="overflow-hidden rounded-xl border border-line">
            @if (c.text === draftText()) {
              <p class="p-8 text-center text-body text-ink-muted">{{ 'adminReference.sameAsDraft' | t }}</p>
            } @else {
              @defer {
                <bt-json-diff class="max-h-[70vh] overflow-auto" [original]="c.text" [changed]="draftText()" />
              } @loading {
                <div class="h-80 animate-pulse bg-surface-sunken"></div>
              }
            }
          </div>
        } @else {
          <p class="bo-card px-4 py-10 text-center text-small text-ink-muted">{{ 'adminReference.pickVersion' | t }}</p>
        }
      </div>
    </div>
  `
})
export class HistoryViewComponent {
  protected readonly store = inject(ReferenceStore);
  private readonly api = inject(ApiClient);

  protected readonly filters: readonly ('all' | CatalogPart)[] = ['all', ...this.store.parts];
  protected readonly filter = signal<'all' | CatalogPart>('all');
  protected readonly compared = signal<{ name: CatalogPart; version: number; text: string } | null>(null);

  protected readonly versions = computed(() => (this.store.versions.value() ?? []).filter((v) => this.filter() === 'all' || v.name === this.filter()));
  protected readonly draftText = computed(() => {
    const c = this.compared();
    const draft = this.store.draft();
    return c && draft ? JSON.stringify(partContent(draft, c.name), null, 2) : '';
  });

  protected inForce(name: CatalogPart): number | undefined {
    return this.store.changeset()?.baseVersions?.[name];
  }

  protected isCompared(v: CatalogVersion): boolean {
    const c = this.compared();
    return !!c && c.name === v.name && c.version === v.version;
  }

  protected async compare(v: CatalogVersion): Promise<void> {
    const out = await firstValueFrom(this.api.get<{ content: unknown }>(`/admin/reference/versions/${v.name}/${v.version}`));
    this.compared.set({ name: v.name, version: v.version, text: JSON.stringify(out.content, null, 2) });
  }
}
