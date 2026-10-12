import { ChangeDetectionStrategy, Component, booleanAttribute, computed, input } from '@angular/core';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { StaffRole } from '../../../domain/enums';

const TONES: Readonly<Record<StaffRole, string>> = {
  super_admin: 'bg-ink text-surface',
  moderator: 'bg-clay-soft text-clay-ink',
  finance: 'bg-sky-soft text-sky',
  support: 'bg-forest-soft text-forest'
};

/** A staff member's role as a labelled chip, each role in its own fixed
 * tone, per the design system's StaffRoleBadge. `onDark` is the super
 * admin's chip on the back office's dark sidebar, where ink would vanish. */
@Component({
  selector: 'bt-staff-role-badge',
  imports: [TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<span [class]="'inline-flex h-[22px] items-center rounded-full px-2.5 text-[11px] font-semibold whitespace-nowrap ' + tone()">{{ 'staffRole.' + role() | t }}</span>`
})
export class StaffRoleBadgeComponent {
  readonly role = input.required<StaffRole>();
  readonly onDark = input(false, { transform: booleanAttribute });

  protected readonly tone = computed(() => (this.onDark() && this.role() === 'super_admin' ? 'bg-on-forest-deep text-ink' : TONES[this.role()]));
}
