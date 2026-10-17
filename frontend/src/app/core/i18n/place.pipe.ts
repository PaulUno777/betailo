import { ChangeDetectorRef, Pipe, PipeTransform, effect, inject } from '@angular/core';
import { LocaleService } from './locale.service';
import { placeLabelKey } from '../../domain/reference/countries';

/** `{{ farm.location | place }}`: a farm's stored town or region in the
 * reader's language. Farms store the catalog's keys ("far-north",
 * "ngaoundere") for listed places and the text for others, which shows
 * as typed. Impure for the same reason as TranslatePipe. */
@Pipe({ name: 'place', pure: false })
export class PlacePipe implements PipeTransform {
  private readonly locale = inject(LocaleService);

  constructor() {
    const changeDetectorRef = inject(ChangeDetectorRef);
    effect(() => {
      this.locale.locale();
      changeDetectorRef.markForCheck();
    });
  }

  transform(value: string | null | undefined): string {
    return value ? this.locale.t(placeLabelKey(value)) : '';
  }
}
