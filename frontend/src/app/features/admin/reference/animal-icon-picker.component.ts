import { ChangeDetectionStrategy, Component, model } from '@angular/core';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { AnimalIconComponent } from '../../../shared/ui/animal-icon/animal-icon.component';
import { ANIMAL_NAMES } from '../../../shared/ui/animal-icon/animal-paths';

/** Picks one of the brand's twelve animal drawings: the API accepts
 * exactly these names. Shown as a grid, the chosen one highlighted. */
@Component({
  selector: 'bt-animal-icon-picker',
  imports: [AnimalIconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div role="radiogroup" [attr.aria-label]="'adminReference.field.icon' | t" class="grid grid-cols-6 gap-1">
      @for (name of names; track name) {
        <button
          type="button"
          role="radio"
          [attr.aria-checked]="name === value()"
          [attr.title]="name"
          (click)="value.set(name)"
          class="flex h-12 items-center justify-center rounded-lg border transition"
          [class]="name === value() ? 'border-forest bg-forest-soft text-on-forest-soft' : 'border-transparent text-ink-muted hover:border-line-strong hover:text-ink'"
        >
          <bt-animal-icon [name]="name" [size]="30" [label]="name" />
        </button>
      }
    </div>
  `
})
export class AnimalIconPickerComponent {
  readonly value = model.required<string>();
  protected readonly names = ANIMAL_NAMES;
}
