import { ChangeDetectionStrategy, Component, ElementRef, HostListener, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { SellerSpaceService } from '../../../core/auth/seller-space.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { farmStatusEntry } from '../../../domain/rules/status-registry';
import { AvatarComponent } from '../avatar/avatar.component';
import { IconComponent } from '../icon/icon.component';

/** The farm card at the foot of the seller sidebar, per the portal
 * design: which farm the portal is working on, with its review status.
 * With several farms it opens a list to switch between them; either way
 * it leads to "Mes fermes". Renders on the sidebar's dark surface. */
@Component({
  selector: 'bt-farm-switcher',
  imports: [RouterLink, AvatarComponent, IconComponent, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (seller.currentFarm(); as farm) {
      <div class="relative">
        @if (open()) {
          <div
            role="menu"
            [attr.aria-label]="'farmSwitcher.label' | t"
            class="absolute inset-x-0 bottom-full z-40 mb-2 rounded-lg border border-line bg-surface-card py-2 text-ink"
            style="box-shadow: var(--shadow-dialog)"
          >
            @for (f of seller.farms(); track f.id) {
              <button
                type="button"
                role="menuitemradio"
                [attr.aria-checked]="f.id === farm.id"
                (click)="choose(f.id)"
                class="flex w-full items-center gap-3 px-3 py-2 text-left hover:bg-surface-sunken"
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
            <a routerLink="/portal/farms" role="menuitem" (click)="open.set(false)" class="flex items-center gap-3 px-3 py-2.5 text-body hover:bg-surface-sunken">
              <bt-icon name="settings" [size]="18" class="text-ink-muted" />
              {{ 'farmSwitcher.manage' | t }}
            </a>
          </div>
        }
        <button
          type="button"
          (click)="open.set(!open())"
          [attr.aria-expanded]="open()"
          aria-haspopup="menu"
          [attr.aria-label]="'farmSwitcher.current' | t: { farm: farm.name }"
          class="flex w-full items-center gap-2.5 rounded-lg bg-on-forest-deep/5 p-3 text-left transition hover:bg-on-forest-deep/10"
        >
          <bt-avatar [name]="farm.name" [size]="38" shape="square" tone="clay" />
          <span class="flex min-w-0 flex-1 flex-col">
            <span class="truncate text-body-strong text-on-forest-deep">{{ farm.name }}</span>
            <span class="truncate text-caption font-medium text-on-forest-deep/70">{{ statusKey(farm.status) | t }} · {{ farm.location }}</span>
          </span>
          <bt-icon name="chevron-down" [size]="16" class="shrink-0 rotate-180 text-on-forest-deep/70" />
        </button>
      </div>
    }
  `
})
export class FarmSwitcherComponent {
  protected readonly seller = inject(SellerSpaceService);
  private readonly elementRef = inject(ElementRef<HTMLElement>);

  protected readonly open = signal(false);
  protected readonly several = computed(() => this.seller.farms().length > 1);

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
