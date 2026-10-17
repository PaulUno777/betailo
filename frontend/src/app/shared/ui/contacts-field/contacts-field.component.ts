import { ChangeDetectionStrategy, Component, computed, input, model } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { FarmContact } from '../../../domain/models';
import { callingCodeOf } from '../../../domain/reference/countries';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { IconComponent } from '../icon/icon.component';
import { TextFieldComponent } from '../text-field/text-field.component';

/** The numbers buyers reach a farm on, up to `limit` (the API's
 * setting). Each is either WhatsApp (buyers can write or call) or call
 * only. A number typed without its "+" reads as a national number of
 * `country`, whose prefix the field shows; the API does the same. Blank
 * rows are left out on save. */
@Component({
  selector: 'bt-contacts-field',
  imports: [FormsModule, IconComponent, TextFieldComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <fieldset class="flex flex-col gap-3">
      <legend class="mb-1 text-small font-bold text-ink">{{ 'contactsField.label' | t }}</legend>
      <p class="-mt-1 text-small text-ink-muted">{{ 'contactsField.hint' | t: { max: limit() } }}</p>
      @for (c of rows(); track $index; let i = $index) {
        <div class="flex flex-col gap-2 rounded-md border border-line p-3">
          <div class="flex items-end gap-2">
            <bt-text-field
              class="min-w-0 flex-1"
              [label]="'contactsField.number' | t: { n: i + 1 }"
              type="tel"
              inputmode="tel"
              autocomplete="tel-national"
              [prefix]="prefixFor(c.phone)"
              [name]="'contact-' + i"
              [ngModel]="c.phone"
              (ngModelChange)="patch(i, { phone: $event })"
            />
            @if (rows().length > 1 || c.phone) {
              <button
                type="button"
                class="mb-2 flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-ink-muted hover:bg-surface-sunken"
                [attr.aria-label]="'contactsField.remove' | t: { n: i + 1 }"
                (click)="remove(i)"
              >
                <bt-icon name="x" [size]="18" />
              </button>
            }
          </div>
          <div class="inline-flex w-full gap-1 rounded-full bg-surface-sunken p-1" role="radiogroup" [attr.aria-label]="'contactsField.kind' | t">
            <button
              type="button"
              role="radio"
              [attr.aria-checked]="c.whatsapp"
              class="flex h-9 flex-1 items-center justify-center gap-1.5 rounded-full text-small font-semibold transition"
              [class]="c.whatsapp ? 'bg-surface-card text-forest shadow-sm' : 'text-ink-muted'"
              (click)="patch(i, { whatsapp: true })"
            >
              <bt-icon name="message" [size]="16" />{{ 'contactsField.whatsapp' | t }}
            </button>
            <button
              type="button"
              role="radio"
              [attr.aria-checked]="!c.whatsapp"
              class="flex h-9 flex-1 items-center justify-center gap-1.5 rounded-full text-small font-semibold transition"
              [class]="!c.whatsapp ? 'bg-surface-card text-ink shadow-sm' : 'text-ink-muted'"
              (click)="patch(i, { whatsapp: false })"
            >
              <bt-icon name="phone" [size]="16" />{{ 'contactsField.callOnly' | t }}
            </button>
          </div>
        </div>
      }
      @if (rows().length < limit()) {
        <button type="button" class="flex items-center gap-1.5 self-start text-small font-semibold text-forest" (click)="add()">
          <bt-icon name="plus" [size]="16" />{{ 'contactsField.add' | t }}
        </button>
      }
    </fieldset>
  `
})
export class ContactsFieldComponent {
  readonly contacts = model<FarmContact[]>([]);
  /** ISO code of the farm's country, for the prefix. */
  readonly country = input('');
  readonly limit = input(3);

  /** At least one row to type in, WhatsApp first: it's how buyers
   * here reach a farmer. */
  protected readonly rows = computed<FarmContact[]>(() => (this.contacts().length ? this.contacts() : [{ phone: '', whatsapp: true }]));

  protected prefixFor(phone: string): string | undefined {
    const code = callingCodeOf(this.country());
    return code && !phone.trim().startsWith('+') ? `+${code}` : undefined;
  }

  protected patch(i: number, change: Partial<FarmContact>): void {
    this.contacts.set(this.rows().map((c, j) => (j === i ? { ...c, ...change } : c)));
  }

  protected add(): void {
    const rows = this.rows();
    this.contacts.set([...rows, { phone: '', whatsapp: !rows.some((c) => c.whatsapp) }]);
  }

  protected remove(i: number): void {
    this.contacts.set(this.rows().filter((_, j) => j !== i));
  }
}

/** The rows worth sending: trimmed, blanks dropped. */
export function filledContacts(contacts: readonly FarmContact[]): FarmContact[] {
  return contacts.map((c) => ({ phone: c.phone.trim(), whatsapp: c.whatsapp })).filter((c) => c.phone);
}
