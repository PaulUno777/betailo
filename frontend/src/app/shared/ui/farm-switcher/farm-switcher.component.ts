import { NgTemplateOutlet } from '@angular/common';
import { ChangeDetectionStrategy, Component, ElementRef, HostListener, booleanAttribute, inject, input, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { SellerSpaceService } from '../../../core/auth/seller-space.service';
import { TranslatePipe } from '../../../core/i18n/translate.pipe';
import { PlacePipe } from '../../../core/i18n/place.pipe';
import { farmStatusEntry } from '../../../domain/rules/status-registry';
import { AvatarComponent } from '../avatar/avatar.component';
import { BottomSheetComponent } from '../bottom-sheet/bottom-sheet.component';
import { IconComponent } from '../icon/icon.component';
import { StatusBadgeComponent } from '../status-badge/status-badge.component';

/** The farm switcher at the top of the seller sidebar, per the portal
 * design: which farm the portal is working on, with its review status.
 * It opens the list of the account's farms to switch between them, how
 * many the account may still open, and the way to "Mes fermes". With
 * several farms the first choice is "Toutes mes fermes", the combined
 * view. Renders on the sidebar's dark surface, or `light` at the top of
 * the seller home on phones. */
@Component({
  selector: 'bt-farm-switcher',
  imports: [NgTemplateOutlet, RouterLink, AvatarComponent, BottomSheetComponent, IconComponent, StatusBadgeComponent, TranslatePipe, PlacePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (seller.currentFarm(); as farm) {
      <div class="relative">
        <button
          type="button"
          (click)="open.set(!open())"
          [attr.aria-expanded]="open()"
          aria-haspopup="menu"
          [attr.aria-label]="'farmSwitcher.current' | t: { farm: seller.allFarms() ? ('farmSwitcher.all' | t) : farm.name }"
          class="flex w-full items-center gap-3 text-left transition"
          [class]="
            light()
              ? 'min-h-[60px] rounded-[18px] border border-line bg-surface-card py-2 pr-3.5 pl-2 text-ink'
              : 'min-h-14 rounded-2xl border border-on-forest-deep/16 bg-on-forest-deep/8 py-2 pr-3 pl-2 text-on-forest-deep hover:bg-on-forest-deep/12'
          "
        >
          @if (seller.allFarms()) {
            <span class="flex shrink-0 items-center justify-center rounded-[10px] bg-forest-soft font-display font-extrabold text-forest" [class]="light() ? 'h-11 w-11 text-[15px]' : 'h-[38px] w-[38px] text-small'" aria-hidden="true">Σ</span>
            <span class="flex min-w-0 flex-1 flex-col leading-tight">
              <b class="truncate" [class]="light() ? 'font-display text-[19px]' : 'text-small'">{{ 'farmSwitcher.all' | t }}</b>
              <span class="truncate text-caption" [class]="light() ? 'text-ink-muted' : 'text-on-forest-deep/70'">{{ 'farmSwitcher.allHint' | t: { count: seller.farms().length } }}</span>
            </span>
          } @else {
            <bt-avatar [name]="farm.name" [size]="light() ? 44 : 38" shape="square" tone="forest" class="shrink-0 !rounded-[10px]" />
            <span class="flex min-w-0 flex-1 flex-col leading-tight">
              <b class="truncate" [class]="light() ? 'font-display text-[19px]' : 'text-small'">{{ farm.name }}</b>
              <span class="truncate text-caption" [class]="light() ? 'text-ink-muted' : 'text-on-forest-deep/70'">{{ statusKey(farm.status) | t }} · {{ farm.location | place }}</span>
            </span>
          }
          <bt-icon name="chevron-down" [size]="16" class="shrink-0 transition" [class]="light() ? 'text-ink' : 'text-on-forest-deep/70'" [class.rotate-180]="open() && !light()" />
        </button>

        <ng-template #choices>
          @if (seller.farms().length > 1) {
            <button
              type="button"
              role="menuitemradio"
              [attr.aria-checked]="seller.allFarms()"
              (click)="chooseAll()"
              class="flex min-h-12 w-full items-center gap-3 rounded-[14px] px-3 py-2 text-left"
              [class]="seller.allFarms() ? 'bg-forest-soft' : 'hover:bg-surface-sunken'"
            >
              <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-[10px] bg-forest-soft font-display text-small font-extrabold text-forest" aria-hidden="true">Σ</span>
              <span class="flex min-w-0 flex-1 flex-col">
                <span class="truncate text-body-strong">{{ 'farmSwitcher.all' | t }}</span>
                <span class="truncate text-small text-ink-muted">{{ 'farmSwitcher.allHint' | t: { count: seller.farms().length } }}</span>
              </span>
            </button>
          }
          @for (f of seller.farms(); track f.id) {
            <button
              type="button"
              role="menuitemradio"
              [attr.aria-checked]="!seller.allFarms() && f.id === farm.id"
              (click)="choose(f.id)"
              class="flex min-h-12 w-full items-center gap-3 rounded-[14px] px-3 py-2 text-left"
              [class]="!seller.allFarms() && f.id === farm.id ? 'bg-forest-soft' : 'hover:bg-surface-sunken'"
            >
              <bt-avatar [name]="f.name" [size]="32" shape="square" tone="forest" />
              <span class="flex min-w-0 flex-1 flex-col">
                <span class="truncate text-body-strong">{{ f.name }}</span>
                <span class="truncate text-small text-ink-muted">{{ f.location | place }}</span>
              </span>
              <bt-status-badge kind="farm" [status]="f.status" />
            </button>
          }
          <div class="mt-1 flex items-center justify-between gap-3 border-t border-line px-3 pt-3 pb-1 text-caption text-ink-muted">
            <span>{{ 'farmQuota.used' | t: { count: seller.farms().length, max: seller.farmLimit() } }}</span>
            @if (!seller.canAddFarm()) {
              <span>{{ 'farmSwitcher.limitReached' | t }}</span>
            }
          </div>
          <a routerLink="/portal/farms" role="menuitem" (click)="open.set(false)" class="flex min-h-11 items-center gap-3 rounded-[14px] px-3 py-2 text-body hover:bg-surface-sunken">
            <bt-icon name="leaf" [size]="18" class="text-ink-muted" />
            {{ 'farmSwitcher.manage' | t }}
          </a>
          @if (seller.canAddFarm()) {
            <a routerLink="/sell" role="menuitem" (click)="open.set(false)" class="flex min-h-11 items-center gap-3 rounded-[14px] px-3 py-2 text-body hover:bg-surface-sunken">
              <bt-icon name="plus" [size]="18" class="text-ink-muted" />
              {{ 'farmSwitcher.create' | t }}
            </a>
          }
        </ng-template>

        @if (light()) {
          <bt-bottom-sheet [open]="open()" [title]="'farmSwitcher.label' | t" (closed)="open.set(false)">
            <div role="menu" class="flex flex-col gap-1">
              <ng-container [ngTemplateOutlet]="choices" />
            </div>
          </bt-bottom-sheet>
        } @else if (open()) {
          <div
            role="menu"
            [attr.aria-label]="'farmSwitcher.label' | t"
            class="absolute inset-x-0 top-full z-40 mt-2 flex w-[300px] flex-col gap-0.5 rounded-2xl border border-line bg-surface-card p-1.5 text-ink"
            style="box-shadow: var(--shadow-dialog)"
          >
            <ng-container [ngTemplateOutlet]="choices" />
          </div>
        }
      </div>
    }
  `
})
export class FarmSwitcherComponent {
  protected readonly seller = inject(SellerSpaceService);
  private readonly elementRef = inject(ElementRef<HTMLElement>);

  /** On the page's light surface (phones), where the choices open as a
   * bottom sheet; the default is the sidebar's dark surface. */
  readonly light = input(false, { transform: booleanAttribute });

  protected readonly open = signal(false);

  protected statusKey(status: Parameters<typeof farmStatusEntry>[0]): string {
    return farmStatusEntry(status).labelKey;
  }

  protected chooseAll(): void {
    this.seller.selectAll();
    this.open.set(false);
  }

  protected choose(farmId: string): void {
    this.seller.select(farmId);
    this.open.set(false);
  }

  @HostListener('document:click', ['$event'])
  protected onDocumentClick(event: MouseEvent): void {
    // The sheet closes itself; only the sidebar's popover needs this.
    if (this.open() && !this.light() && !this.elementRef.nativeElement.contains(event.target as Node)) {
      this.open.set(false);
    }
  }

  @HostListener('document:keydown.escape')
  protected onEscape(): void {
    this.open.set(false);
  }
}
