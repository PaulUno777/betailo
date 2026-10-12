import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ToastService } from '../../../core/notifications/toast.service';
import { ToastComponent } from './toast.component';

/** Just above the bottom tab bar on phones and tablets (never over it),
 * bottom-right on desktop (README) —
 * mounted once in the app shell, fed by the global `ToastService`. */
@Component({
  selector: 'bt-toast-host',
  imports: [ToastComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="fixed inset-x-0 bottom-[calc(var(--height-bottom-nav)+env(safe-area-inset-bottom)+0.75rem)] z-50 flex flex-col items-center gap-2 px-4 lg:bottom-6 lg:left-auto lg:right-6 lg:items-end lg:px-0">
      @for (toast of toastService.toasts(); track toast.id) {
        <bt-toast [toast]="toast" (dismissed)="toastService.dismiss(toast.id)" />
      }
    </div>
  `
})
export class ToastHostComponent {
  protected readonly toastService = inject(ToastService);
}
