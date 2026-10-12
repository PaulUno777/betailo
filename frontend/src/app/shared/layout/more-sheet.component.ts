import { ChangeDetectionStrategy, Component, inject, input, output } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../core/auth/auth.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { BottomSheetComponent } from '../ui/bottom-sheet/bottom-sheet.component';
import { IconComponent } from '../ui/icon/icon.component';
import { IconName } from '../ui/icon/icon-paths';

export interface MoreSheetLink {
  readonly labelKey: string;
  readonly icon: IconName;
  readonly route: string;
}

/** The bottom nav's "Plus" overflow on phones and tablets: the
 * section's remaining destinations (including the way back to buying
 * and the account, where language and theme live), then sign-out. Each
 * setting lives in one place only. */
@Component({
  selector: 'bt-more-sheet',
  imports: [RouterLink, BottomSheetComponent, IconComponent, TranslatePipe],
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
        <button type="button" (click)="signOut()" class="-mx-2 {{ rowClass }} text-alert">
          <bt-icon name="logout" [size]="20" />
          <span class="flex-1 text-left">{{ 'nav.signOut' | t }}</span>
        </button>
      </div>
    </bt-bottom-sheet>
  `
})
export class MoreSheetComponent {
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
