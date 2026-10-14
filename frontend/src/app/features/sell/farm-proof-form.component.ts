import { ChangeDetectionStrategy, Component, DestroyRef, computed, inject, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { forkJoin, of, switchMap } from 'rxjs';
import { IdDocumentType } from '../../domain/enums';
import { Farm } from '../../domain/models';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { PhotoUploadService } from '../../core/http/photo-upload.service';
import { AppError } from '../../core/http/app-error';
import { LocaleService } from '../../core/i18n/locale.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';
import { SelectFieldComponent } from '../../shared/ui/select-field/select-field.component';

interface PickedFile {
  readonly file: File;
  readonly preview: string;
}

const MAX_FARM_PHOTOS = 3;
const DOCUMENT_TYPES: readonly IdDocumentType[] = ['national_id', 'passport', 'driving_licence'];
/** A passport's identity page stands alone; a card has two faces. The
 * API holds the rule (`farm.id_document_back_required`); this only
 * decides whether to show the second tile. */
const TWO_SIDED: ReadonlySet<IdDocumentType> = new Set(['national_id', 'driving_licence']);

/** What a farm's owner sends to get it verified, per the "Créer ma
 * ferme" design: the identity document of the person responsible (its
 * type, front and, for a card, back), one to three recent photos of the
 * farm, and their declaration that the farm is theirs. Staff review
 * them; nobody else ever sees them. Used when opening a farm and again
 * after a refusal (the reason is shown above the form).
 *
 * The API decides what is enough (and, on a test platform, may approve
 * without proof): the form says what still seems to be missing, sends
 * what was picked and shows the API's answer rather than blocking on
 * its own guess. */
@Component({
  selector: 'bt-farm-proof-form',
  imports: [FormsModule, ButtonComponent, IconComponent, NoticeComponent, SelectFieldComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form class="flex flex-col gap-7" (submit)="$event.preventDefault(); submit()">
      @if (farm().review?.rejectionReason; as reason) {
        <bt-notice tone="danger" [title]="rejectedTitle()">{{ reason }}</bt-notice>
      }

      <fieldset class="flex flex-col gap-3">
        <legend class="mb-1 flex items-center gap-2.5 text-body-strong">
          <span class="flex h-7 w-7 items-center justify-center rounded-full bg-forest-soft font-mono text-caption font-semibold text-forest" aria-hidden="true">1</span>
          {{ 'farmProof.idLabel' | t }}
        </legend>
        <p class="text-small text-ink-muted">{{ 'farmProof.idHint' | t }}</p>
        <bt-select-field [label]="'farmProof.idTypeLabel' | t" [options]="typeOptions()" [ngModel]="docType()" (ngModelChange)="setType($event)" />
        <div class="grid gap-2.5" [class.sm:grid-cols-2]="twoSided()">
          @for (side of sides(); track side.key) {
            @if (side.file(); as doc) {
              <div class="relative h-36 overflow-hidden rounded-xl border border-line bg-surface-sunken">
                <img [src]="doc.preview" [alt]="side.labelKey | t" class="h-full w-full object-cover" />
                <span class="absolute inset-x-0 bottom-0 flex items-center gap-1.5 bg-ink/75 px-3 py-1.5 text-caption font-semibold text-surface">
                  <bt-icon name="check" [size]="14" />{{ side.labelKey | t }}
                </span>
                <button
                  type="button"
                  (click)="clearSide(side.key)"
                  [attr.aria-label]="'farmProof.removeSide' | t: { side: (side.labelKey | t) }"
                  class="absolute top-1.5 right-1.5 flex h-11 w-11 items-center justify-center rounded-full bg-ink/70 text-surface"
                >
                  <bt-icon name="x" [size]="18" />
                </button>
              </div>
            } @else {
              <label [class]="tileClass + ' h-36 flex-col gap-1.5 text-center'">
                <bt-icon name="shield" [size]="22" />
                <span class="text-body-strong">{{ side.addKey | t }}</span>
                <span class="text-caption font-normal text-ink-muted">{{ 'farmProof.sideHint' | t }}</span>
                <input type="file" accept="image/*" class="sr-only" (change)="pickSide(side.key, $event)" />
              </label>
            }
          }
        </div>
      </fieldset>

      <fieldset class="flex flex-col gap-3">
        <legend class="mb-1 flex w-full items-center gap-2.5 text-body-strong">
          <span class="flex h-7 w-7 items-center justify-center rounded-full bg-forest-soft font-mono text-caption font-semibold text-forest" aria-hidden="true">2</span>
          <span class="flex-1">{{ 'farmProof.photosLabel' | t }}</span>
          <span class="font-mono text-caption font-medium text-ink-muted">{{ photos().length }} / {{ maxPhotos }}</span>
        </legend>
        <p class="text-small text-ink-muted">{{ 'farmProof.photosHint' | t: { max: maxPhotos } }}</p>
        <ul class="grid grid-cols-3 gap-2">
          @for (p of photos(); track p.preview; let i = $index) {
            <li class="relative aspect-square">
              <img [src]="p.preview" [alt]="'farmProof.photoAlt' | t: { n: i + 1 }" class="h-full w-full rounded-xl object-cover" />
              <button
                type="button"
                (click)="removePhoto(i)"
                [attr.aria-label]="'farmProof.removePhoto' | t: { n: i + 1 }"
                class="absolute top-1 right-1 flex h-11 w-11 items-center justify-center rounded-full bg-ink/70 text-surface"
              >
                <bt-icon name="x" [size]="16" />
              </button>
            </li>
          }
          @if (photos().length < maxPhotos) {
            <li class="aspect-square">
              <label [class]="tileClass + ' h-full flex-col gap-1.5 text-small'">
                <bt-icon name="camera" [size]="22" />
                {{ 'farmProof.photoAdd' | t }}
                <input type="file" accept="image/*" capture="environment" class="sr-only" (change)="pickPhoto($event)" />
              </label>
            </li>
          }
        </ul>
      </fieldset>

      <label class="flex cursor-pointer items-start gap-3 rounded-xl border border-line bg-surface-card p-4 text-small leading-relaxed">
        <input type="checkbox" class="mt-0.5 h-5 w-5 shrink-0 accent-forest" [ngModel]="certified()" (ngModelChange)="certified.set($event)" name="certified" />
        <span>{{ 'farmProof.certify' | t }}</span>
      </label>

      <bt-notice tone="info" icon="lock">{{ 'farmProof.privacy' | t }}</bt-notice>

      @if (error(); as message) {
        <bt-notice tone="danger">{{ message }}</bt-notice>
      }

      <div class="flex flex-col gap-2">
        <div class="flex flex-col gap-3 sm:flex-row-reverse">
          <bt-button class="sm:flex-1" type="submit" size="lg" block [disabled]="saving()">
            {{ (saving() ? 'farmProof.submitting' : 'farmProof.submit') | t }}
          </bt-button>
          @if (skippable()) {
            <bt-button variant="outline" size="lg" [disabled]="saving()" (click)="skipped.emit()">{{ 'farmProof.later' | t }}</bt-button>
          }
        </div>
        @if (missing(); as key) {
          <p class="text-small text-ink-muted sm:text-right" aria-live="polite">{{ key | t }}</p>
        }
      </div>
    </form>
  `
})
export class FarmProofFormComponent {
  private readonly farmsApi = inject(FarmsApi);
  private readonly uploads = inject(PhotoUploadService);
  private readonly locale = inject(LocaleService);

  readonly farm = input.required<Farm>();
  /** Offers "later": the farm stays a draft the owner can prepare. */
  readonly skippable = input(false);
  /** The farm as the API returned it after submission. */
  readonly submitted = output<Farm>();
  readonly skipped = output<void>();

  protected readonly maxPhotos = MAX_FARM_PHOTOS;
  protected readonly tileClass =
    'flex min-h-16 cursor-pointer items-center justify-center rounded-xl border-2 border-dashed border-line-strong bg-surface-card px-4 text-ink transition hover:border-forest has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-focus';

  protected readonly docType = signal<IdDocumentType>('national_id');
  protected readonly front = signal<PickedFile | null>(null);
  protected readonly back = signal<PickedFile | null>(null);
  protected readonly photos = signal<readonly PickedFile[]>([]);
  protected readonly certified = signal(false);
  protected readonly saving = signal(false);
  protected readonly error = signal<string | null>(null);

  protected readonly twoSided = computed(() => TWO_SIDED.has(this.docType()));

  protected readonly typeOptions = computed(() => DOCUMENT_TYPES.map((value) => ({ value, label: this.locale.t(`idDocumentType.${value}`) })));

  protected readonly sides = computed(() => {
    const front = { key: 'front' as const, file: this.front, labelKey: 'farmProof.front', addKey: 'farmProof.frontAdd' };
    const back = { key: 'back' as const, file: this.back, labelKey: 'farmProof.back', addKey: 'farmProof.backAdd' };
    return this.twoSided() ? [front, back] : [front];
  });

  /** The refusal's kind, when staff picked one, as the notice's title. */
  protected readonly rejectedTitle = computed(() => {
    const code = this.farm().review?.rejectionCode;
    return code && code !== 'other' ? this.locale.t(`rejectionCode.${code}`) : this.locale.t('farmProof.rejectedTitle');
  });

  /** What still seems to be missing, said under the button. A hint
   * only: the API has the last word on what is enough. */
  protected readonly missing = computed<string | null>(() => {
    if (!this.front()) return 'farmProof.missing.front';
    if (this.twoSided() && !this.back()) return 'farmProof.missing.back';
    if (this.photos().length === 0) return 'farmProof.missing.photos';
    if (!this.certified()) return 'farmProof.missing.certify';
    return null;
  });

  constructor() {
    inject(DestroyRef).onDestroy(() => {
      for (const f of [this.front(), this.back(), ...this.photos()]) {
        if (f) URL.revokeObjectURL(f.preview);
      }
    });
  }

  protected setType(type: IdDocumentType): void {
    this.docType.set(type);
    if (!TWO_SIDED.has(type)) this.clearSide('back');
  }

  protected pickSide(side: 'front' | 'back', event: Event): void {
    const file = takeFile(event);
    if (!file) return;
    this.clearSide(side);
    (side === 'front' ? this.front : this.back).set({ file, preview: URL.createObjectURL(file) });
  }

  protected clearSide(side: 'front' | 'back'): void {
    const target = side === 'front' ? this.front : this.back;
    const doc = target();
    if (doc) URL.revokeObjectURL(doc.preview);
    target.set(null);
  }

  protected pickPhoto(event: Event): void {
    const file = takeFile(event);
    if (file && this.photos().length < MAX_FARM_PHOTOS) {
      this.photos.update((list) => [...list, { file, preview: URL.createObjectURL(file) }]);
    }
  }

  protected removePhoto(index: number): void {
    URL.revokeObjectURL(this.photos()[index].preview);
    this.photos.update((list) => list.filter((_, i) => i !== index));
  }

  protected submit(): void {
    if (this.saving()) return;
    this.saving.set(true);
    this.error.set(null);
    const farmId = this.farm().id;
    const upload = (f: PickedFile | null) => (f ? this.uploads.upload(f.file, () => this.farmsApi.presignProof(farmId)) : of(''));
    const photos = this.photos();
    forkJoin({
      idDocumentKey: upload(this.front()),
      idDocumentBackKey: upload(this.twoSided() ? this.back() : null),
      photoKeys: photos.length ? forkJoin(photos.map(upload)) : of([] as string[])
    })
      .pipe(switchMap((keys) => this.farmsApi.submitForReview(farmId, { ...keys, idDocumentType: this.docType(), certified: this.certified() })))
      .subscribe({
        next: (farm) => {
          this.saving.set(false);
          this.submitted.emit(farm);
        },
        error: (err: AppError) => {
          this.saving.set(false);
          this.error.set(err.message);
        }
      });
  }
}

/** The picked file; the input is cleared so the same file can be picked
 * again after being removed. */
function takeFile(event: Event): File | null {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0] ?? null;
  input.value = '';
  return file;
}
