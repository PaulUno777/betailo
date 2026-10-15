import { Injectable, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { ApiClient } from '../../../core/http/api-client';
import { AppError } from '../../../core/http/app-error';
import { apiResource } from '../../../core/http/api-resource';
import { LocaleService } from '../../../core/i18n/locale.service';
import { ToastService } from '../../../core/notifications/toast.service';
import { ReferenceService } from '../../../core/reference/reference.service';
import { CATALOG_PARTS, Catalog, CatalogPart } from '../../../domain/reference/catalog';
import { JsonSchema } from './json-schema-tools';
import { marksOf } from './reference-ops';
import { BulkTarget, CatalogOp, CatalogVersion, Changeset, EntryRequest, Usage } from './reference.types';

/** The reference data editor's state: the shared changeset (the
 * version in force plus the changes waiting to be published), what
 * uses each entry, the version history and the schema. Every change
 * goes through the API, which checks it against the cumulative draft;
 * the revision it was based on travels with it, so a write made after
 * another admin's is refused and the changeset reloaded instead of
 * silently overwritten. Provided by the page. */
@Injectable()
export class ReferenceStore {
  private readonly api = inject(ApiClient);
  private readonly toast = inject(ToastService);
  private readonly reference = inject(ReferenceService);
  private readonly locale = inject(LocaleService);

  readonly changeset = signal<Changeset | null>(null);
  /** The entry being added or edited (the drawer), and where a paste
   * of many entries goes (the bulk dialog): opened from any view. */
  readonly editing = signal<EntryRequest | null>(null);
  readonly bulking = signal<BulkTarget | null>(null);
  readonly loadFailed = signal(false);
  readonly busy = signal(false);

  readonly base = apiResource<Catalog>(() => this.api.url('/reference'));
  readonly usage = apiResource<Usage>(() => this.api.url('/admin/reference/usage'));
  readonly versions = apiResource<CatalogVersion[]>(() => this.api.url('/admin/reference/versions'));
  /** Each reference's JSON Schema, fetched when first needed, again
   * when the language changes. */
  private readonly schemas = signal<Partial<Record<string, JsonSchema>>>({});
  private readonly fetching = new Set<string>();

  readonly draft = computed(() => this.changeset()?.draft);
  readonly changes = computed(() => this.changeset()?.changes ?? []);
  readonly marks = computed(() => marksOf(this.changes()));

  /** The JSON Schema of reference name, null until it has loaded. */
  schemaOf(name: CatalogPart): JsonSchema | null {
    const lang = this.locale.locale();
    const key = `${name}:${lang}`;
    const known = this.schemas()[key];
    // Read from templates and computed signals: the fetch writes later,
    // never while they are being evaluated.
    if (known === undefined && !this.fetching.has(key)) {
      this.fetching.add(key);
      firstValueFrom(this.api.get<JsonSchema>(`/admin/reference/schema/${name}`, { lang })).then(
        (schema) => this.schemas.update((s) => ({ ...s, [key]: schema })),
        () => this.fetching.delete(key)
      );
    }
    return known ?? null;
  }

  /** The references, in their order. */
  readonly parts = CATALOG_PARTS;

  /** Reads the changeset again (on load, on focus, after a conflict). */
  async load(): Promise<void> {
    try {
      this.changeset.set(await firstValueFrom(this.api.get<Changeset>('/admin/reference/changeset')));
      this.loadFailed.set(false);
    } catch {
      this.loadFailed.set(!this.changeset());
    }
  }

  /** Queues ops (all or none). Resolves to null, or to the refusal for
   * the caller to show next to what was typed. */
  queue(ops: readonly CatalogOp[]): Promise<AppError | null> {
    return this.write(() => this.api.post<Changeset>('/admin/reference/changeset/ops', { rev: this.rev(), ops }));
  }

  undo(id: string): Promise<AppError | null> {
    return this.write(() => this.api.delete<Changeset>(`/admin/reference/changeset/ops/${id}?rev=${this.rev()}`), true);
  }

  clear(): Promise<AppError | null> {
    return this.write(() => this.api.delete<Changeset>(`/admin/reference/changeset?rev=${this.rev()}`), true);
  }

  /** One reference, edited as JSON, queued as the changes it makes. */
  importPart(name: CatalogPart, content: unknown): Promise<AppError | null> {
    return this.write(() => this.api.post<Changeset>('/admin/reference/changeset/import', { rev: this.rev(), name, content }));
  }

  /** What brings a reference back to an older version, queued as changes. */
  restore(name: CatalogPart, version: number): Promise<AppError | null> {
    return this.write(() => this.api.post<Changeset>('/admin/reference/changeset/restore', { rev: this.rev(), name, version }), true);
  }

  /** Publishes the changeset: a new version of each reference it touches. */
  async publish(note: string): Promise<AppError | null> {
    this.busy.set(true);
    try {
      const out = await firstValueFrom(
        this.api.post<{ reference: Catalog; published: CatalogVersion[] }>('/admin/reference/changeset/publish', { rev: this.rev(), note })
      );
      this.reference.apply(out.reference);
      this.toast.show({
        tone: 'success',
        title: 'adminReference.published',
        detail: out.published.map((p) => `${this.locale.t('adminReference.part.' + p.name)} v${p.version}`).join(' · ')
      });
      await this.load();
      this.base.reload();
      this.versions.reload();
      this.usage.reload();
      return null;
    } catch (err) {
      return this.refused(err as AppError, true);
    } finally {
      this.busy.set(false);
    }
  }

  private rev(): number {
    return this.changeset()?.rev ?? 0;
  }

  private async write(request: () => ReturnType<ApiClient['post']>, toastErrors = false): Promise<AppError | null> {
    this.busy.set(true);
    try {
      this.changeset.set((await firstValueFrom(request())) as Changeset);
      return null;
    } catch (err) {
      return this.refused(err as AppError, toastErrors);
    } finally {
      this.busy.set(false);
    }
  }

  private async refused(err: AppError, toast: boolean): Promise<AppError> {
    if (err.code === 'catalog.changeset_stale') {
      await this.load();
      this.toast.show({ tone: 'warning', title: 'adminReference.reloaded' });
    } else if (toast) {
      this.toast.show({ tone: 'danger', title: err.message });
    }
    return err;
  }
}
