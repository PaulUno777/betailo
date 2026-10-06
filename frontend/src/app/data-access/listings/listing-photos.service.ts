import { Injectable, inject } from '@angular/core';
import { Observable, forkJoin, of, switchMap } from 'rxjs';
import { Listing } from '../../domain/models';
import { PhotoUploadService } from '../../core/http/photo-upload.service';
import { PickedPhoto } from '../../shared/ui/photo-picker/photo-picker.component';
import { ListingsApi } from './listings.api';

/** Saves a listing's photos exactly as the picker shows them: uploads
 * the newly picked files (compressed, through presigned URLs), then sets
 * the full ordered list of keys, the first being the cover. */
@Injectable({ providedIn: 'root' })
export class ListingPhotosService {
  private readonly listingsApi = inject(ListingsApi);
  private readonly uploads = inject(PhotoUploadService);

  save(listingId: string, photos: readonly PickedPhoto[]): Observable<Listing | null> {
    if (!photos.length) return of(null);
    const keys$ = photos.map((p) =>
      p.key ? of(p.key) : this.uploads.upload(p.file!, () => this.listingsApi.presignPhoto(listingId))
    );
    return forkJoin(keys$).pipe(switchMap((keys) => this.listingsApi.setPhotos(listingId, keys)));
  }
}
