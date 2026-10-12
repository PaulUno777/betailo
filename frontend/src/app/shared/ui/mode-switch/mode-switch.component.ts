import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { IconComponent } from '../icon/icon.component';

export type ModeSwitchMode = 'buy' | 'sell';
export type ModeSwitchTone = 'light' | 'dark';

/** "Acheter | Vendre": the always-visible way a farmer moves between the
 * market and their seller space (the Airbnb "switch to hosting" idea),
 * instead of digging for it in the account menu. `mode` is the side the
 * page is on; the other half links across. Renders nothing for anyone
 * who isn't a farmer: buyers are invited to sell elsewhere, and admins
 * have their own space. */
@Component({
  selector: 'bt-mode-switch',
  imports: [RouterLink, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (auth.role() === 'farmer') {
      <div
        role="group"
        [attr.aria-label]="'modeSwitch.label' | t"
        class="inline-flex rounded-full p-1"
        [class]="tone() === 'dark' ? 'bg-on-forest-deep/10' : 'bg-surface-sunken'"
      >
        @for (m of modes; track m.mode) {
          @if (m.mode === mode()) {
            <span
              aria-current="page"
              class="inline-flex h-9 items-center gap-1.5 rounded-full px-3.5 text-small font-semibold"
              [class]="tone() === 'dark' ? 'bg-surface text-ink' : 'bg-surface-card text-ink shadow-sm'"
            >
              <bt-icon [name]="m.icon" [size]="16" />
              {{ m.labelKey | t }}
            </span>
          } @else {
            <a
              [routerLink]="m.route"
              class="inline-flex h-9 items-center gap-1.5 rounded-full px-3.5 text-small font-semibold transition"
              [class]="tone() === 'dark' ? 'text-on-forest-deep/80 hover:text-on-forest-deep' : 'text-ink-muted hover:text-ink'"
            >
              <bt-icon [name]="m.icon" [size]="16" />
              {{ m.labelKey | t }}
            </a>
          }
        }
      </div>
    }
  `
})
export class ModeSwitchComponent {
  protected readonly auth = inject(AuthService);

  readonly mode = input.required<ModeSwitchMode>();
  readonly tone = input<ModeSwitchTone>('light');

  protected readonly modes = [
    { mode: 'buy', labelKey: 'modeSwitch.buy', icon: 'search', route: '/' },
    { mode: 'sell', labelKey: 'modeSwitch.sell', icon: 'leaf', route: '/portal' }
  ] as const;
}
