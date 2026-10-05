import { ChangeDetectionStrategy, Component, inject, input, output } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../core/auth/auth.service';
import { ThemeService } from '../../core/theme/theme.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { BottomSheetComponent } from '../ui/bottom-sheet/bottom-sheet.component';
import { IconComponent } from '../ui/icon/icon.component';
import { IconName } from '../ui/icon/icon-paths';
import { LocaleToggleComponent } from '../ui/locale-toggle/locale-toggle.component';

export interface MoreSheetLink {
  readonly labelKey: string;
  readonly icon: IconName;
  readonly route: string;
}

/** The bottom nav's "Plus" overflow on phones: the section's remaining
 * destinations, then the preferences the compact mobile header leaves
 * out (language, theme) and sign-out — everything a desktop user reaches
 * from the sidebar and header, one thumb-reachable sheet away. */
@Component({
  selector: 'bt-more-sheet',
  imports: [RouterLink, BottomSheetComponent, IconComponent, LocaleToggleComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <bt-bottom-sheet [open]="open()" [title]="'moreSheet.title' | t" (closed)="closed.emit()">
      <nav class="-mx-2 flex flex-col">
        @for (link of links(); track link.route) {
          <a [routerLink]="link.route" (click)="closed.emit()" [class]="rowClass">
            <bt-icon [name]="link.icon" [size]="20" class="text-ink-muted" />
            <span class="flex-1">{{ link.labelKey | t }}</span>
            <bt-icon name="chevron-right" [size]="18" class="text-ink-muted" />
          </a>
        }
      </nav>
      <div class="my-3 border-t border-line"></div>
      <div class="flex flex-col gap-1">
        <div class="flex min-h-control-lg items-center justify-between gap-3">
          <span class="text-body">{{ 'moreSheet.language' | t }}</span>
          <bt-locale-toggle />
        </div>
        <button type="button" (click)="theme.toggle()" class="-mx-2 {{ rowClass }}">
          <bt-icon [name]="theme.isDark() ? 'sun' : 'moon'" [size]="20" class="text-ink-muted" />
          <span class="flex-1 text-left">{{ (theme.isDark() ? 'moreSheet.lightMode' : 'moreSheet.darkMode') | t }}</span>
        </button>
        <button type="button" (click)="signOut()" class="-mx-2 {{ rowClass }} text-alert">
          <bt-icon name="logout" [size]="20" />
          <span class="flex-1 text-left">{{ 'nav.signOut' | t }}</span>
        </button>
      </div>
    </bt-bottom-sheet>
  `
})
export class MoreSheetComponent {
  protected readonly theme = inject(ThemeService);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  readonly open = input.required<boolean>();
  readonly links = input.required<readonly MoreSheetLink[]>();
  readonly closed = output<void>();

  protected readonly rowClass =
    'flex min-h-control-lg items-center gap-3 rounded-md px-2 text-body text-ink transition active:bg-surface-sunken hover:bg-surface-sunken';

  protected signOut(): void {
    this.closed.emit();
    this.auth.signOut();
    this.router.navigateByUrl('/');
  }
}
