import { Injectable, signal } from '@angular/core';

export type ToastTone = 'info' | 'success' | 'warning' | 'danger';

/** `label`/`title`/`detail` are i18n translation keys, not literal text —
 * `ToastComponent` pipes each through `t`, so callers pass e.g.
 * `'reservations.paymentSuccessTitle'`, never the rendered string. */
export interface ToastAction {
  readonly label: string;
  readonly onClick: () => void;
}

export interface Toast {
  readonly id: number;
  readonly tone: ToastTone;
  readonly title: string;
  readonly detail?: string;
  readonly action?: ToastAction;
}

export interface ToastInput {
  readonly tone: ToastTone;
  readonly title: string;
  readonly detail?: string;
  readonly action?: ToastAction;
}

const AUTO_DISMISS_MS = 5000;

/** Global toast queue — a `Notice` inside a form reacts to that form's
 * own submit; this is for confirmations that outlive the component that
 * triggered them (e.g. after navigating away on success). */
@Injectable({ providedIn: 'root' })
export class ToastService {
  private readonly _toasts = signal<Toast[]>([]);
  readonly toasts = this._toasts.asReadonly();

  private nextId = 0;

  show(input: ToastInput): number {
    const id = ++this.nextId;
    this._toasts.update((toasts) => [...toasts, { ...input, id }]);
    if (input.tone !== 'danger') {
      setTimeout(() => this.dismiss(id), AUTO_DISMISS_MS);
    }
    return id;
  }

  dismiss(id: number): void {
    this._toasts.update((toasts) => toasts.filter((toast) => toast.id !== id));
  }
}
