import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { ThemeService } from '../../core/theme/theme.service';
import { TranslatePipe } from '../../core/i18n/translate.pipe';
import { SidebarNavComponent, SidebarNavItem, SidebarNavSurface } from '../ui/sidebar-nav/sidebar-nav.component';
import { BottomNavComponent } from '../ui/bottom-nav/bottom-nav.component';
import { IconButtonComponent } from '../ui/icon-button/icon-button.component';
import { LocaleToggleComponent } from '../ui/locale-toggle/locale-toggle.component';
import { UserMenuComponent } from '../ui/user-menu/user-menu.component';
import { ToastHostComponent } from '../ui/toast/toast-host.component';

/** The shell shared by the farmer portal and admin — `SidebarNav` (on
 * whichever surface the caller passes; that's what makes admin visually
 * distinct, not a different component) plus a slim top bar for
 * locale/theme/account, and the same mobile `BottomNav` fallback so
 * neither section ever leaves a phone user without navigation. */
@Component({
  selector: 'bt-sidebar-shell',
  imports: [
    RouterOutlet,
    SidebarNavComponent,
    BottomNavComponent,
    IconButtonComponent,
    LocaleToggleComponent,
    UserMenuComponent,
    ToastHostComponent,
    TranslatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex min-h-full">
      <div class="hidden sm:block">
        <bt-sidebar-nav [items]="items()" [surface]="surface()" />
      </div>
      <div class="flex min-w-0 flex-1 flex-col">
        <header class="flex h-16 items-center justify-end gap-3 border-b border-line bg-surface-raised px-5 sm:px-8">
          <bt-locale-toggle />
          <bt-icon-button [icon]="theme.isDark() ? 'sun' : 'moon'" [label]="'nav.theme' | t" (click)="theme.toggle()" />
          <bt-user-menu />
        </header>
        <main class="flex-1 bg-surface pb-24 sm:pb-0">
          <router-outlet />
        </main>
      </div>
      <bt-bottom-nav />
      <bt-toast-host />
    </div>
  `
})
export class SidebarShellComponent {
  protected readonly theme = inject(ThemeService);

  readonly items = input.required<SidebarNavItem[]>();
  readonly surface = input<SidebarNavSurface>('forest-deep');
}
