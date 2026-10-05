import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ToastService } from '../../../core/notifications/toast.service';
import { ToastComponent } from './toast.component';

/** Bottom of the screen on mobile, bottom-right on desktop (README) —
 * mounted once in the app shell, fed by the global `ToastService`. */
@Component({
  selector: 'bt-toast-host',
  imports: [ToastComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="fixed inset-x-0 bottom-24 z-50 flex flex-col items-center gap-2 px-4 sm:bottom-6 sm:items-end sm:right-6 sm:left-auto sm:px-0">
      @for (toast of toastService.toasts(); track toast.id) {
        <bt-toast [toast]="toast" (dismissed)="toastService.dismiss(toast.id)" />
      }
    </div>
  `
})
export class ToastHostComponent {
  protected readonly toastService = inject(ToastService);
}
