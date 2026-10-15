import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, catchError, forkJoin, from, map, of, switchMap, throwError } from 'rxjs';
import { LocaleService } from '../i18n/locale.service';
import { AppError } from './app-error';

interface Encoding {
  /** Longest side, in px, the image is scaled down to. */
  readonly maxDimension: number;
  /** Hard ceiling on the encoded size; the API refuses anything over
   * it (domain.MaxPhotoBytes / MaxThumbBytes), so they must match. */
  readonly maxBytes: number;
  /** Tried in order until the encoded image fits maxBytes. */
  readonly qualities: readonly number[];
  /** When even the lowest quality is too heavy, the image shrinks by
   * DOWNSCALE_STEP and the qualities are tried again, down to this. */
  readonly minDimension: number;
}

const PHOTO: Encoding = { maxDimension: 1600, maxBytes: 800 * 1024, qualities: [0.82, 0.72, 0.62], minDimension: 800 };
/** The small copy cards and grids show instead of the photo. */
const THUMB: Encoding = { maxDimension: 640, maxBytes: 120 * 1024, qualities: [0.75, 0.65, 0.55], minDimension: 320 };
const DOWNSCALE_STEP = 0.8;

/** The byte sizes the API locks the upload URLs to. */
export interface PhotoSizes {
  readonly size: number;
  readonly thumbSize?: number;
}

export interface PresignedUpload {
  readonly key: string;
  readonly uploadUrl: string;
  /** Present when the endpoint stores a small copy too. */
  readonly thumbUploadUrl?: string;
}

export interface PreparedPhoto {
  readonly photo: Blob;
  readonly thumb?: Blob;
}

/** Why a picked file can't be uploaded. */
export class PhotoRejected extends Error {
  constructor(readonly code: 'photo.unsupported' | 'photo.too_large') {
    super(code);
  }
}

/** Uploads one photo through presigned URLs. Phone cameras produce 3-8
 * MB images, which is slow and costly on mobile data, so each photo is
 * first scaled to at most 1600px and re-encoded as JPEG in the browser,
 * typically a few hundred KB and never more than 800 KB. Re-encoding
 * also drops the EXIF block, GPS position included. With `thumb`, a
 * 640px copy is made too, for cards and grids. A file the browser can't
 * decode is refused rather than sent as is. The API signs each URL for
 * the declared size, so storage takes nothing heavier. Emits the
 * storage key the API minted. */
@Injectable({ providedIn: 'root' })
export class PhotoUploadService {
  private readonly http = inject(HttpClient);
  private readonly locale = inject(LocaleService);

  upload(file: File, presign: (sizes: PhotoSizes) => Observable<PresignedUpload>, opts: { thumb?: boolean } = {}): Observable<string> {
    return from(preparePhoto(file, opts.thumb ?? false)).pipe(
      catchError((err: unknown) => throwError(() => (err instanceof PhotoRejected ? this.toAppError(err) : err))),
      switchMap(({ photo, thumb }) =>
        presign({ size: photo.size, thumbSize: thumb?.size }).pipe(
          switchMap(({ key, uploadUrl, thumbUploadUrl }) =>
            forkJoin([this.put(uploadUrl, photo), thumb && thumbUploadUrl ? this.put(thumbUploadUrl, thumb) : of(null)]).pipe(map(() => key))
          )
        )
      )
    );
  }

  private put(url: string, blob: Blob): Observable<unknown> {
    return this.http.put(url, blob, { headers: { 'Content-Type': 'image/jpeg' } });
  }

  private toAppError(err: PhotoRejected): AppError {
    return {
      kind: 'validation',
      status: 0,
      code: err.code,
      message: this.locale.t(`apiErrors.${err.code}`, { max: Math.round(PHOTO.maxBytes / 1024) })
    };
  }
}

/** Decodes the file once and encodes the photo (and, with `withThumb`,
 * its small copy) as JPEG under their size caps. Throws PhotoRejected
 * when the browser can't decode the file (HEIC outside Safari, a PDF,
 * ...) or it can't be brought under a cap. */
export async function preparePhoto(file: File, withThumb: boolean): Promise<PreparedPhoto> {
  let bitmap: ImageBitmap;
  try {
    bitmap = await createImageBitmap(file, { imageOrientation: 'from-image' });
  } catch {
    throw new PhotoRejected('photo.unsupported');
  }
  try {
    const photo = await encode(bitmap, PHOTO);
    return withThumb ? { photo, thumb: await encode(bitmap, THUMB) } : { photo };
  } finally {
    bitmap.close();
  }
}

/** Lowers quality, then size, until the JPEG fits `enc.maxBytes`. */
async function encode(bitmap: ImageBitmap, enc: Encoding): Promise<Blob> {
  let longest = Math.min(enc.maxDimension, Math.max(bitmap.width, bitmap.height));
  for (;;) {
    const canvas = draw(bitmap, longest);
    for (const quality of enc.qualities) {
      const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', quality));
      if (!blob) throw new PhotoRejected('photo.unsupported');
      if (blob.size <= enc.maxBytes) return blob;
    }
    if (longest <= enc.minDimension) throw new PhotoRejected('photo.too_large');
    longest = Math.max(enc.minDimension, Math.round(longest * DOWNSCALE_STEP));
  }
}

/** The bitmap on a canvas whose longest side is `longest` px, over white
 * so a transparent PNG doesn't turn black as a JPEG. */
function draw(bitmap: ImageBitmap, longest: number): HTMLCanvasElement {
  const scale = Math.min(1, longest / Math.max(bitmap.width, bitmap.height));
  const canvas = document.createElement('canvas');
  canvas.width = Math.round(bitmap.width * scale);
  canvas.height = Math.round(bitmap.height * scale);
  const ctx = canvas.getContext('2d');
  if (!ctx) throw new PhotoRejected('photo.unsupported');
  ctx.fillStyle = '#fff';
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
  return canvas;
}
