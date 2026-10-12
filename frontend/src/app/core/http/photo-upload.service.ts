import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, from, map, switchMap } from 'rxjs';

/** Longest side, in px, a photo is scaled down to before upload. */
const MAX_DIMENSION = 1600;
const JPEG_QUALITY = 0.82;

export interface PresignedUpload {
  readonly key: string;
  readonly uploadUrl: string;
}

/** Uploads one photo through a presigned URL. Phone cameras produce 3-8
 * MB images, which is slow and costly on mobile data, so each photo is
 * first scaled to at most 1600px and re-encoded as JPEG in the browser,
 * typically a few hundred KB. Emits the storage key the API minted. */
@Injectable({ providedIn: 'root' })
export class PhotoUploadService {
  private readonly http = inject(HttpClient);

  upload(file: File, presign: () => Observable<PresignedUpload>): Observable<string> {
    return from(compressImage(file)).pipe(
      switchMap((blob) =>
        presign().pipe(
          switchMap(({ key, uploadUrl }) =>
            this.http.put(uploadUrl, blob, { headers: { 'Content-Type': 'image/jpeg' } }).pipe(map(() => key))
          )
        )
      )
    );
  }
}

/** Scales an image down to MAX_DIMENSION and re-encodes it as JPEG. An
 * image that can't be decoded (an unsupported format) is sent as is. */
export async function compressImage(file: File): Promise<Blob> {
  let bitmap: ImageBitmap;
  try {
    bitmap = await createImageBitmap(file, { imageOrientation: 'from-image' });
  } catch {
    return file;
  }
  const scale = Math.min(1, MAX_DIMENSION / Math.max(bitmap.width, bitmap.height));
  const width = Math.round(bitmap.width * scale);
  const height = Math.round(bitmap.height * scale);
  const canvas = document.createElement('canvas');
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext('2d');
  if (!ctx) {
    bitmap.close();
    return file;
  }
  ctx.drawImage(bitmap, 0, 0, width, height);
  bitmap.close();
  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', JPEG_QUALITY));
  return blob ?? file;
}
