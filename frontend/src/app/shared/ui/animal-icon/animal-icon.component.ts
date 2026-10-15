import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { DomSanitizer } from '@angular/platform-browser';
import { ANIMAL_PATHS, isAnimalName } from './animal-paths';

/** One of the brand's animal drawings, `currentColor`, named the way the
 * reference data names it. A name this build doesn't know draws
 * nothing. Decorative unless given a `label`. Reads best at 20px and up. */
@Component({
  selector: 'bt-animal-icon',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <svg
      [attr.width]="size()"
      [attr.height]="size()"
      viewBox="0 0 64 64"
      fill="none"
      stroke="currentColor"
      stroke-width="2.5"
      stroke-linecap="round"
      stroke-linejoin="round"
      [attr.role]="label() ? 'img' : null"
      [attr.aria-label]="label() ?? null"
      [attr.aria-hidden]="label() ? null : 'true'"
      [innerHTML]="markup()"
    ></svg>
  `,
  host: { class: 'inline-flex shrink-0' }
})
export class AnimalIconComponent {
  private readonly sanitizer = inject(DomSanitizer);

  readonly name = input.required<string>();
  readonly size = input(24);
  readonly label = input<string>();

  protected readonly markup = computed(() => {
    const name = this.name();
    return this.sanitizer.bypassSecurityTrustHtml(isAnimalName(name) ? ANIMAL_PATHS[name] : '');
  });
}
