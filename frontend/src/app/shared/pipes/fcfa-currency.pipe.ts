import { Pipe, PipeTransform } from '@angular/core';

/** `toLocaleString('fr-FR')` separates thousands with U+202F (narrow
 * no-break space), built here from its char code rather than a literal
 * in source, which editors and linters alike tend to mangle. */
const NARROW_NO_BREAK_SPACE = new RegExp(String.fromCharCode(0x202f), 'g');

/** "76 500 FCFA" — spaces as the thousands separator, FCFA suffix, per
 * the `amount` type style. FCFA has no minor unit, so this is always a
 * whole-number format. */
@Pipe({ name: 'fcfa' })
export class FcfaCurrencyPipe implements PipeTransform {
  transform(value: number | null | undefined): string {
    if (value === null || value === undefined) return '';
    return `${Math.round(value).toLocaleString('fr-FR').replace(NARROW_NO_BREAK_SPACE, ' ')} FCFA`;
  }
}
