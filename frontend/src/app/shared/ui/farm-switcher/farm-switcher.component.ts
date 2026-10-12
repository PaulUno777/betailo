import { ChangeDetectionStrategy, Component, ElementRef, HostListener, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { SellerSpaceService } from '../../../core/auth/seller-space.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { farmStatusEntry } from '../../../domain/rules/status-registry';
import { AvatarComponent } from '../avatar/avatar.component';
import { IconComponent } from '../icon/icon.component';

/** The farm switcher at the top of the seller sidebar, per the portal
 * design: which farm the portal is working on, with its review status.
 * It opens the list of the account's farms to switch between them, how
 * many the account may still open, and the way to "Mes fermes". Renders
 * on the sidebar's dark surface. */
@Component({
  selector: 'bt-farm-switcher',
  imports: [RouterLink, AvatarComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (seller.currentFarm(); as farm) {
      <div class="relative">
        <button
          type="button"
          (click)="open.set(!open())"
          [attr.aria-expanded]="open()"
          aria-haspopup="menu"
          [attr.aria-label]="'farmSwitcher.current' | t: { farm: farm.name }"
          class="flex min-h-14 w-full items-center gap-3 rounded-2xl border border-on-forest-deep/16 bg-on-forest-deep/8 py-2 pr-3 pl-2 text-left text-on-forest-deep transition hover:bg-on-forest-deep/12"
        >
          <bt-avatar [name]="farm.name" [size]="38" shape="square" tone="forest" class="shrink-0 !rounded-[10px]" />
          <span class="flex min-w-0 flex-1 flex-col leading-tight">
            <b class="truncate text-small">{{ farm.name }}</b>
            <span class="truncate text-caption text-on-forest-deep/70">{{ statusKey(farm.status) | t }} · {{ farm.location }}</span>
          </span>
          <bt-icon name="chevron-down" [size]="16" class="shrink-0 text-on-forest-deep/70 transition" [class.rotate-180]="open()" />
        </button>
        @if (open()) {
          <div
            role="menu"
            [attr.aria-label]="'farmSwitcher.label' | t"
            class="absolute inset-x-0 top-full z-40 mt-2 rounded-2xl border border-line bg-surface-card py-2 text-ink"
            style="box-shadow: var(--shadow-dialog)"
          >
            @for (f of seller.farms(); track f.id) {
              <button
                type="button"
                role="menuitemradio"
                [attr.aria-checked]="f.id === farm.id"
                (click)="choose(f.id)"
                class="flex min-h-12 w-full items-center gap-3 px-3 py-2 text-left hover:bg-surface-sunken"
              >
                <bt-avatar [name]="f.name" [size]="32" shape="square" tone="forest" />
                <span class="flex min-w-0 flex-1 flex-col">
                  <span class="truncate text-body-strong">{{ f.name }}</span>
                  <span class="truncate text-small text-ink-muted">{{ statusKey(f.status) | t }}</span>
                </span>
                @if (f.id === farm.id) {
                  <bt-icon name="check" [size]="18" class="shrink-0 text-forest" />
                }
              </button>
            }
            <div class="my-1 border-t border-line"></div>
            <p class="px-3 py-1.5 text-caption text-ink-muted">{{ 'farmQuota.used' | t: { count: seller.farms().length, max: seller.farmLimit() } }}</p>
            <a routerLink="/portal/farms" role="menuitem" (click)="open.set(false)" class="flex min-h-11 items-center gap-3 px-3 py-2 text-body hover:bg-surface-sunken">
              <bt-icon name="leaf" [size]="18" class="text-ink-muted" />
              {{ 'farmSwitcher.manage' | t }}
            </a>
            @if (seller.canAddFarm()) {
              <a routerLink="/sell" role="menuitem" (click)="open.set(false)" class="flex min-h-11 items-center gap-3 px-3 py-2 text-body hover:bg-surface-sunken">
                <bt-icon name="plus" [size]="18" class="text-ink-muted" />
                {{ 'farmSwitcher.create' | t }}
              </a>
            }
          </div>
        }
      </div>
    }
  `
})
export class FarmSwitcherComponent {
  protected readonly seller = inject(SellerSpaceService);
  private readonly elementRef = inject(ElementRef<HTMLElement>);

  protected readonly open = signal(false);

  protected statusKey(status: Parameters<typeof farmStatusEntry>[0]): string {
    return farmStatusEntry(status).labelKey;
  }

  protected choose(farmId: string): void {
    this.seller.select(farmId);
    this.open.set(false);
  }

  @HostListener('document:click', ['$event'])
  protected onDocumentClick(event: MouseEvent): void {
    if (this.open() && !this.elementRef.nativeElement.contains(event.target as Node)) {
      this.open.set(false);
    }
  }

  @HostListener('document:keydown.escape')
  protected onEscape(): void {
    this.open.set(false);
  }
}
