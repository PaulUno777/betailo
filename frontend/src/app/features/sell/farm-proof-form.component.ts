import { ChangeDetectionStrategy, Component, DestroyRef, inject, input, output, signal } from '@angular/core';
import { forkJoin, of, switchMap } from 'rxjs';
import { Farm } from '../../domain/models';
import { FarmsApi } from '../../data-access/farms/farms.api';
import { PhotoUploadService } from '../../core/http/photo-upload.service';
import { AppError } from '../../core/http/app-error';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { IconComponent } from '../../shared/ui/icon/icon.component';
import { NoticeComponent } from '../../shared/ui/notice/notice.component';

interface PickedFile {
  readonly file: File;
  readonly preview: string;
}

const MAX_FARM_PHOTOS = 3;

/** What a farm's owner submits to get it validated: a photo of their
 * identity document and one to three photos of the farm. Staff review
 * them; nobody else ever sees them. Used when opening a farm and again
 * after a rejection (the reason is shown above the form).
 *
 * The API decides what is enough (and, on a test platform, may approve
 * without proof): this form uploads what was picked and shows the
 * API's answer rather than second-guessing it. */
@Component({
  selector: 'bt-farm-proof-form',
  imports: [ButtonComponent, IconComponent, NoticeComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form class="flex flex-col gap-6" (submit)="$event.preventDefault(); submit()">
      @if (farm().review?.rejectionReason; as reason) {
        <bt-notice tone="danger" [title]="'farmProof.rejectedTitle' | t">{{ reason }}</bt-notice>
      }
      <bt-notice tone="info" icon="lock">{{ 'farmProof.privacy' | t }}</bt-notice>

      <fieldset class="flex flex-col gap-2">
        <legend class="mb-2 text-body-strong">{{ 'farmProof.idLabel' | t }}</legend>
        <p class="text-small text-ink-muted">{{ 'farmProof.idHint' | t }}</p>
        @if (idDocument(); as doc) {
          <div class="flex items-center gap-3 rounded-lg border border-line bg-surface-card p-3">
            <img [src]="doc.preview" alt="" class="h-16 w-24 shrink-0 rounded-md object-cover" />
            <span class="min-w-0 flex-1 truncate text-small text-ink-muted">{{ doc.file.name }}</span>
            <button
              type="button"
              (click)="clearId()"
              [attr.aria-label]="'farmProof.removeId' | t"
              class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-ink-muted hover:bg-surface-sunken"
            >
              <bt-icon name="x" [size]="18" />
            </button>
          </div>
        } @else {
          <label [class]="tileClass">
            <bt-icon name="shield" [size]="22" />
            {{ 'farmProof.idAdd' | t }}
            <input type="file" accept="image/*" class="sr-only" (change)="pickId($event)" />
          </label>
        }
      </fieldset>

      <fieldset class="flex flex-col gap-2">
        <legend class="mb-2 text-body-strong">{{ 'farmProof.photosLabel' | t }}</legend>
        <p class="text-small text-ink-muted">{{ 'farmProof.photosHint' | t: { max: maxPhotos } }}</p>
        <ul class="grid grid-cols-3 gap-2">
          @for (p of photos(); track p.preview; let i = $index) {
            <li class="relative aspect-square">
              <img [src]="p.preview" alt="" class="h-full w-full rounded-lg object-cover" />
              <button
                type="button"
                (click)="removePhoto(i)"
                [attr.aria-label]="'farmProof.removePhoto' | t: { n: i + 1 }"
                class="absolute right-1 top-1 flex h-9 w-9 items-center justify-center rounded-full bg-ink/70 text-surface"
              >
                <bt-icon name="x" [size]="16" />
              </button>
            </li>
          }
          @if (photos().length < maxPhotos) {
            <li class="aspect-square">
              <label [class]="tileClass + ' h-full flex-col text-small'">
                <bt-icon name="camera" [size]="22" />
                {{ 'farmProof.photoAdd' | t }}
                <input type="file" accept="image/*" capture="environment" class="sr-only" (change)="pickPhoto($event)" />
              </label>
            </li>
          }
        </ul>
      </fieldset>

      @if (error(); as message) {
        <bt-notice tone="danger">{{ message }}</bt-notice>
      }

      <div class="flex flex-col gap-3 sm:flex-row-reverse">
        <bt-button class="sm:flex-1" type="submit" size="lg" block [disabled]="saving()">
          {{ (saving() ? 'farmProof.submitting' : 'farmProof.submit') | t }}
        </bt-button>
        @if (skippable()) {
          <bt-button variant="outline" size="lg" [disabled]="saving()" (click)="skipped.emit()">{{ 'farmProof.later' | t }}</bt-button>
        }
      </div>
    </form>
  `
})
export class FarmProofFormComponent {
  private readonly farmsApi = inject(FarmsApi);
  private readonly uploads = inject(PhotoUploadService);

  readonly farm = input.required<Farm>();
  /** Offers "later": the farm stays a draft the owner can prepare. */
  readonly skippable = input(false);
  /** The farm as the API returned it after submission. */
  readonly submitted = output<Farm>();
  readonly skipped = output<void>();

  protected readonly maxPhotos = MAX_FARM_PHOTOS;
  protected readonly tileClass =
    'flex min-h-16 cursor-pointer items-center justify-center gap-3 rounded-lg border-2 border-dashed border-line-strong bg-surface-card px-4 text-body-strong text-ink transition hover:border-forest has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-focus';

  protected readonly idDocument = signal<PickedFile | null>(null);
  protected readonly photos = signal<readonly PickedFile[]>([]);
  protected readonly saving = signal(false);
  protected readonly error = signal<string | null>(null);

  constructor() {
    inject(DestroyRef).onDestroy(() => {
      for (const f of [this.idDocument(), ...this.photos()]) {
        if (f) URL.revokeObjectURL(f.preview);
      }
    });
  }

  protected pickId(event: Event): void {
    const file = takeFile(event);
    if (file) this.idDocument.set({ file, preview: URL.createObjectURL(file) });
  }

  protected clearId(): void {
    const doc = this.idDocument();
    if (doc) URL.revokeObjectURL(doc.preview);
    this.idDocument.set(null);
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
    const upload = (f: PickedFile) => this.uploads.upload(f.file, () => this.farmsApi.presignProof(farmId));
    const doc = this.idDocument();
    const photos = this.photos();
    forkJoin({
      idDocumentKey: doc ? upload(doc) : of(''),
      photoKeys: photos.length ? forkJoin(photos.map(upload)) : of([] as string[])
    })
      .pipe(switchMap((proof) => this.farmsApi.submitForReview(farmId, proof)))
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
