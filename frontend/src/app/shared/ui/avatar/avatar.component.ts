import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

export type AvatarTone = 'clay' | 'forest' | 'sky' | 'plum' | 'mint' | 'amber';
export type AvatarShape = 'round' | 'square';

const TONE_CLASSES: Readonly<Record<AvatarTone, string>> = {
  clay: 'bg-clay-soft text-clay-ink',
  forest: 'bg-forest-soft text-forest',
  sky: 'bg-sky-soft text-sky',
  plum: 'bg-plum-soft text-plum',
  mint: 'bg-forest-soft text-forest',
  amber: 'bg-amber-soft text-amber'
};

function initialsOf(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

/** Initials, or a photo via `src`. Round for people, `square` for farms. */
@Component({
  selector: 'bt-avatar',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (src(); as photo) {
      <img [src]="photo" [alt]="name()" [class]="shapeClass() + ' h-full w-full object-cover'" />
    } @else {
      <span [attr.aria-hidden]="true" class="text-caption font-semibold">{{ initials() }}</span>
    }
  `,
  host: {
    '[class]': 'hostClass()',
    '[style.width.px]': 'size()',
    '[style.height.px]': 'size()',
    '[attr.role]': "src() ? null : 'img'",
    '[attr.aria-label]': 'src() ? null : name()'
  }
})
export class AvatarComponent {
  readonly name = input.required<string>();
  readonly src = input<string>();
  readonly tone = input<AvatarTone>('clay');
  readonly shape = input<AvatarShape>('round');
  readonly size = input(40);

  protected readonly initials = computed(() => initialsOf(this.name()));

  protected shapeClass(): string {
    return this.shape() === 'round' ? 'rounded-full' : 'rounded-sm';
  }

  protected hostClass(): string {
    return `inline-flex items-center justify-center overflow-hidden ${this.shapeClass()} ${TONE_CLASSES[this.tone()]}`;
  }
}
