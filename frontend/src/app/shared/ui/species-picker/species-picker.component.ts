import { ChangeDetectionStrategy, Component, model } from '@angular/core';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { speciesGroups, speciesLabelKey } from '../../../domain/reference/species';

/** What a farm raises: the sub-categories it ticks, grouped under their
 * category (Volaille, Porcins…), from the reference data. */
@Component({
  selector: 'bt-species-picker',
  imports: [TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-4">
      @for (g of groups; track g.key) {
        <div class="flex flex-col gap-2">
          <span class="flex items-center gap-2 text-eyebrow text-ink-muted">
            <span class="h-2.5 w-2.5 rounded-full" [class]="g.dot"></span>
            {{ g.labelKey | t }}
          </span>
          <div class="flex flex-wrap gap-2">
            @for (s of g.species; track s) {
              <button
                type="button"
                (click)="toggle(s)"
                [attr.aria-pressed]="value().includes(s)"
                class="flex h-10 items-center rounded-full border px-4 text-small font-semibold transition"
                [class]="value().includes(s) ? 'border-forest bg-forest-soft text-on-forest-soft' : 'border-line-strong bg-surface-card text-ink'"
              >
                {{ label(s) | t }}
              </button>
            }
          </div>
        </div>
      }
    </div>
  `
})
export class SpeciesPickerComponent {
  readonly value = model<string[]>([]);

  protected readonly groups = speciesGroups();
  protected readonly label = speciesLabelKey;

  protected toggle(s: string): void {
    const v = this.value();
    this.value.set(v.includes(s) ? v.filter((x) => x !== s) : [...v, s]);
  }
}
