import { ChangeDetectionStrategy, Component, computed, inject, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { AppError } from '../../../core/http/app-error';
import { LocaleService } from '../../../core/i18n/locale.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { Label } from '../../../domain/reference/catalog';
import { cycleTypeLabelKey } from '../../../domain/reference/cycle-types';
import { ButtonComponent } from '../../../shared/ui/button/button.component';
import { IconComponent } from '../../../shared/ui/icon/icon.component';
import { entryOf, nameOf } from './reference-ops';
import { ReferenceStore } from './reference.store';
import { Change } from './reference.types';

/** The shared pending changes, as sentences: who did what, the value
 * each edit replaces, each one undoable. Publishing needs a note and
 * a changeset that still applies to the version in force. */
@Component({
  selector: 'bt-changes-panel',
  imports: [FormsModule, ButtonComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="bo-card flex max-h-[calc(100vh-120px)] flex-col overflow-hidden">
      <header class="flex items-center gap-2 border-b border-line px-4 py-3">
        <h2 class="flex-1 text-body-strong">{{ 'adminReference.pending' | t }}</h2>
        <span class="rounded-full bg-surface-sunken px-2 py-0.5 font-mono text-caption">{{ lines().length }}</span>
      </header>

      @if (lines().length) {
        <ol class="flex-1 overflow-y-auto">
          @for (line of lines(); track line.id) {
            <li class="group flex gap-2.5 border-b border-line px-4 py-2.5 last:border-b-0" [class.bg-alert-soft]="line.error">
              <span class="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full" [class]="line.tone">
                <bt-icon [name]="line.icon" [size]="13" />
              </span>
              <div class="flex min-w-0 flex-1 flex-col gap-0.5">
                <span class="text-small">
                  <b>{{ line.name }}</b>
                  <span class="text-ink-muted"> · {{ line.entity }}</span>
                </span>
                @for (detail of line.details; track $index) {
                  <span class="text-caption">{{ detail }}</span>
                }
                @if (line.error) {
                  <span class="text-caption font-semibold text-alert">{{ 'adminReference.changeBroken' | t }}</span>
                }
                <span class="text-[11px] text-ink-muted">{{ line.by }} · {{ line.at }}</span>
              </div>
              <button type="button" class="h-7 shrink-0 self-start rounded-md px-2 text-caption font-semibold text-ink-muted hover:bg-surface-sunken hover:text-ink" [disabled]="store.busy()" (click)="undo(line.id)">
                {{ 'adminReference.undo' | t }}
              </button>
            </li>
          }
        </ol>
      } @else {
        <p class="flex-1 px-4 py-8 text-center text-small text-ink-muted">{{ 'adminReference.noPending' | t }}</p>
      }

      @if (undoError(); as e) {
        <p class="border-t border-line bg-alert-soft px-4 py-2 text-caption text-alert" role="alert">{{ e.message }}</p>
      }

      <footer class="flex flex-col gap-2 border-t border-line bg-surface-raised px-4 py-3">
        @if (store.changeset()?.problems?.length) {
          <div class="flex flex-col rounded-md bg-alert-soft p-2 text-caption text-alert">
            @for (p of store.changeset()!.problems; track $index) {
              <span class="font-mono">· {{ p }}</span>
            }
          </div>
        }
        <input class="bo-input" maxlength="300" [placeholder]="'adminReference.notePlaceholder' | t" [attr.aria-label]="'adminReference.note' | t" [disabled]="!lines().length" [ngModel]="note()" (ngModelChange)="note.set($event)" />
        <div class="flex flex-wrap gap-2">
          <bt-button variant="outline" size="sm" [disabled]="!lines().length" (click)="review.emit()">{{ 'adminReference.reviewDiff' | t }}</bt-button>
          <bt-button variant="outline" size="sm" [disabled]="!lines().length || store.busy()" (click)="store.clear()">{{ 'adminReference.discardAll' | t }}</bt-button>
          <bt-button class="ml-auto" variant="solid" size="sm" [disabled]="!canPublish()" (click)="publish()">
            {{ store.busy() ? ('adminReference.publishing' | t) : ('adminReference.publish' | t) }}
          </bt-button>
        </div>
        @if (nextVersions(); as next) {
          <span class="text-caption text-ink-muted">{{ 'adminReference.willPublish' | t: { versions: next } }}</span>
        }
        @if (publishError(); as e) {
          <span class="text-caption text-alert" role="alert">{{ e.message }}</span>
        }
      </footer>
    </section>
  `
})
export class ChangesPanelComponent {
  protected readonly store = inject(ReferenceStore);
  private readonly locale = inject(LocaleService);

  readonly review = output<void>();

  protected readonly note = signal('');
  protected readonly undoError = signal<AppError | null>(null);
  protected readonly publishError = signal<AppError | null>(null);
  /** "Animaux v5 · Lieux v8": the versions publishing will make. */
  protected readonly nextVersions = computed(() => {
    const cs = this.store.changeset();
    if (!cs?.touched.length) return '';
    return cs.touched.map((name) => `${this.locale.t('adminReference.part.' + name)} v${(cs.baseVersions[name] ?? 0) + 1}`).join(' · ');
  });

  protected readonly canPublish = computed(() => !!this.store.changeset()?.ready && !!this.note().trim() && !this.store.busy());

  protected readonly lines = computed(() => {
    const lang = this.locale.locale();
    const t = (key: string, params?: Record<string, string | number>) => this.locale.t(key, params);
    const draft = this.store.draft();
    const base = this.store.base.value();
    const time = new Intl.DateTimeFormat(lang, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
    return this.store.changes().map((c) => {
      const entry = entryOf(draft, c.op.ref) ?? entryOf(base, c.op.ref);
      const look = LOOKS[c.op.kind];
      return {
        id: c.id,
        icon: look.icon,
        tone: look.tone,
        name: nameOf(entry, lang, c.op.ref),
        entity: t('adminReference.entity.' + c.op.ref.entity),
        details: describe(c, t, (ref) => nameOf(entryOf(draft, ref), lang, ref)),
        error: !!c.error,
        by: c.by.name || '—',
        at: time.format(new Date(c.at))
      };
    });
  });

  protected async undo(id: string): Promise<void> {
    this.undoError.set(await this.store.undo(id));
  }

  protected async publish(): Promise<void> {
    const refused = await this.store.publish(this.note().trim());
    this.publishError.set(refused);
    if (!refused) this.note.set('');
  }
}

const LOOKS = {
  add: { icon: 'plus', tone: 'bg-forest-soft text-on-forest-soft' },
  update: { icon: 'settings', tone: 'bg-clay-soft text-clay-ink' },
  move: { icon: 'chevron-down', tone: 'bg-sky-soft text-sky' }
} as const;

type T = (key: string, params?: Record<string, string | number>) => string;

/** What a change does, one sentence per field it touches. */
function describe(c: Change, t: T, parentName: (ref: Change['op']['ref']) => string): string[] {
  const { op } = c;
  if (op.kind === 'add') {
    const where = op.ref.parent ? parentName({ ...op.ref, entity: op.ref.entity === 'city' ? 'region' : 'category', key: op.ref.parent }) : '';
    return [where ? t('adminReference.change.addedIn', { where }) : t('adminReference.change.added')];
  }
  if (op.kind === 'move') {
    if (op.ref.parent) {
      return [t('adminReference.change.movedTo', { where: parentName({ ...op.ref, entity: op.ref.entity === 'city' ? 'region' : 'category', key: op.ref.parent }) })];
    }
    return [t('adminReference.change.reordered', { position: (op.position ?? 0) + 1 })];
  }
  const before = c.before ?? {};
  return Object.entries(op.value ?? {}).flatMap(([field, value]) => {
    const was = before[field];
    switch (field) {
      case 'active':
        return [t(value ? 'adminReference.change.reactivated' : 'adminReference.change.deactivated')];
      case 'label':
      case 'unit': {
        const after = (field === 'unit' ? (value as { label?: Label }).label : value) as Partial<Label> | undefined;
        const prev = (field === 'unit' ? (was as { label?: Label } | undefined)?.label : was) as Partial<Label> | undefined;
        return (['fr', 'en'] as const)
          .filter((l) => after?.[l] !== undefined && after?.[l] !== prev?.[l])
          .map((l) => t(`adminReference.change.${field}`, { lang: l.toUpperCase(), before: prev?.[l] ?? '—', after: after?.[l] ?? '' }));
      }
      case 'cycleType':
        return [t('adminReference.change.field', { field: t('adminReference.field.cycleType'), before: t(cycleTypeLabelKey(String(was ?? ''))), after: t(cycleTypeLabelKey(String(value))) })];
      default:
        return [t('adminReference.change.field', { field: t('adminReference.field.' + field), before: String(was ?? '—'), after: String(value) })];
    }
  });
}
