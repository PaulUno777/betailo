import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  ElementRef,
  effect,
  inject,
  input,
  output,
  signal,
  viewChild
} from '@angular/core';
import { DOCUMENT } from '@angular/common';
import { IconComponent } from '../icon/icon.component';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';

/** How far (px) a downward drag on the handle must travel before release
 * dismisses the sheet rather than snapping it back. */
const DISMISS_DRAG_PX = 96;

/** The mobile-first container for filters, menus, short forms and
 * confirmations: docked to the bottom edge on phones (thumb reach, the
 * design's filter and "Plus" sheets), a centered card from `sm` up.
 *
 * Built on the native `<dialog>` for the focus trap, Escape handling and
 * top-layer stacking — the same reasoning as `Dialog`. Page scroll is
 * locked while open (showModal alone doesn't stop the body scrolling
 * behind the backdrop on iOS), the backdrop closes it, and the handle
 * can be dragged down to dismiss. */
@Component({
  selector: 'bt-bottom-sheet',
  imports: [IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  // Backdrop clicks bubble here from the <dialog>; keyboard users already
  // get the equivalent from the dialog's native Escape handling.
  host: { '(click)': 'onDialogClick($event)' },
  styles: `
    dialog[open] {
      animation: bt-sheet-in 220ms cubic-bezier(0.2, 0.8, 0.2, 1);
    }
    dialog[open]::backdrop {
      animation: bt-fade-in 220ms ease-out;
    }
    @keyframes bt-sheet-in {
      from {
        transform: translateY(24px);
        opacity: 0;
      }
    }
    @keyframes bt-fade-in {
      from {
        opacity: 0;
      }
    }
  `,
  template: `
    <dialog
      #dialogEl
      [attr.aria-labelledby]="titleId"
      (close)="onNativeClose()"
      [style.transform]="dragOffset() ? 'translateY(' + dragOffset() + 'px)' : null"
      class="fixed inset-x-0 bottom-0 top-auto m-0 hidden open:flex max-h-[88dvh] w-full max-w-none flex-col rounded-t-xl bg-surface-card p-0 text-ink backdrop:bg-black/50 sm:inset-0 sm:m-auto sm:max-h-[80dvh] sm:max-w-lg sm:rounded-xl"
      style="box-shadow: var(--shadow-dialog)"
    >
      <div
        class="flex shrink-0 touch-none justify-center pb-1 pt-3 sm:hidden"
        (pointerdown)="onDragStart($event)"
        (pointermove)="onDragMove($event)"
        (pointerup)="onDragEnd()"
        (pointercancel)="onDragEnd()"
      >
        <span class="h-1.5 w-10 rounded-full bg-line-strong"></span>
      </div>
      <header class="flex shrink-0 items-center justify-between gap-3 px-5 pb-3 pt-2 sm:pt-5">
        <h2 [id]="titleId" class="text-heading-4">{{ title() }}</h2>
        <button
          type="button"
          (click)="close()"
          [attr.aria-label]="'common.close' | t"
          class="-mr-2 inline-flex h-control-md w-11 items-center justify-center rounded-full text-ink-muted hover:bg-surface-sunken"
        >
          <bt-icon name="x" [size]="20" />
        </button>
      </header>
      <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-5 pb-5">
        <ng-content />
      </div>
      <ng-content select="[sheetFooter]" />
    </dialog>
  `
})
export class BottomSheetComponent {
  private static nextId = 0;
  protected readonly titleId = `bt-sheet-title-${++BottomSheetComponent.nextId}`;

  private readonly document = inject(DOCUMENT);

  readonly open = input.required<boolean>();
  readonly title = input.required<string>();
  /** Emitted however the sheet was dismissed — close button, backdrop,
   * Escape or drag — so the parent can sync its `open` state. */
  readonly closed = output<void>();

  protected readonly dragOffset = signal(0);
  private dragStartY: number | null = null;

  private readonly dialogRef = viewChild.required<ElementRef<HTMLDialogElement>>('dialogEl');

  constructor() {
    effect(() => {
      const element = this.dialogRef().nativeElement;
      if (this.open() && !element.open) {
        element.showModal();
        this.lockScroll(true);
      } else if (!this.open() && element.open) {
        element.close();
      }
    });
    inject(DestroyRef).onDestroy(() => this.lockScroll(false));
  }

  close(): void {
    this.dialogRef().nativeElement.close();
  }

  protected onNativeClose(): void {
    this.lockScroll(false);
    this.dragOffset.set(0);
    this.closed.emit();
  }

  /** A click whose target is the <dialog> itself landed on the backdrop
   * (the sheet's content always sits in a child element). */
  protected onDialogClick(event: MouseEvent): void {
    if (event.target === this.dialogRef().nativeElement) {
      this.close();
    }
  }

  protected onDragStart(event: PointerEvent): void {
    this.dragStartY = event.clientY;
    (event.target as HTMLElement).setPointerCapture(event.pointerId);
  }

  protected onDragMove(event: PointerEvent): void {
    if (this.dragStartY !== null) {
      this.dragOffset.set(Math.max(0, event.clientY - this.dragStartY));
    }
  }

  protected onDragEnd(): void {
    if (this.dragStartY === null) {
      return;
    }
    this.dragStartY = null;
    if (this.dragOffset() > DISMISS_DRAG_PX) {
      this.close();
    } else {
      this.dragOffset.set(0);
    }
  }

  private lockScroll(locked: boolean): void {
    this.document.documentElement.style.overflow = locked ? 'hidden' : '';
  }
}
