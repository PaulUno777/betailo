import { TestBed } from '@angular/core/testing';
import { ReservationView } from '../../domain/models';
import { LocaleService } from '../../core/i18n/locale.service';
import { ReservationCardComponent } from './reservation-card.component';

function view(overrides: Partial<ReservationView>): ReservationView {
  return {
    id: 'r1',
    reference: 'RSV-2041',
    listingId: 'l1',
    buyerId: 'b1',
    quantity: 30,
    unitPrice: 2500,
    serviceFee: 1500,
    totalAmount: 76500,
    status: 'confirmed',
    priority: true,
    createdAt: '2026-09-18T08:00:00Z',
    payment: { status: 'held', amount: 76500 },
    listing: { id: 'l1', title: 'Poussins Brahma' },
    farm: { id: 'f1', name: 'Ferme des Collines', location: 'Dschang' },
    cycle: { id: 'c1', reference: 'CYC-0918', status: 'in_progress', startsAt: '2026-09-02T08:00:00Z' },
    group: 'active',
    steps: [
      { key: 'paid', state: 'done' },
      { key: 'delivered', state: 'current' },
      { key: 'confirmed', state: 'upcoming' },
      { key: 'released', state: 'upcoming' }
    ],
    disputeOpen: false,
    reviewed: false,
    actions: ['dispute'],
    ...overrides
  };
}

describe('ReservationCardComponent', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({ imports: [ReservationCardComponent] });
    TestBed.inject(LocaleService).setLocale('fr');
  });

  function render(v: ReservationView) {
    const fixture = TestBed.createComponent(ReservationCardComponent);
    fixture.componentRef.setInput('view', v);
    fixture.detectChanges();
    return fixture;
  }

  it('shows exactly the actions the API allows, and emits the chosen one', () => {
    const fixture = render(view({ status: 'delivered', actions: ['confirm', 'dispute'], deliveredAt: '2026-09-20T08:00:00Z' }));
    const el: HTMLElement = fixture.nativeElement;
    const labels = Array.from(el.querySelectorAll('button')).map((b) => b.textContent?.trim());
    expect(labels).toEqual(['Confirmer la réception', 'Litige']);

    let emitted: string | undefined;
    fixture.componentInstance.action.subscribe((a) => (emitted = a));
    el.querySelector('button')!.click();
    expect(emitted).toBe('confirm');
  });

  it('turns the review action into the "Votre avis compte" band, not a plain button', () => {
    const el: HTMLElement = render(view({ group: 'done', actions: ['review'] })).nativeElement;
    expect(el.textContent).toContain('Votre avis compte');
    expect(el.textContent).toContain('Libéré');
  });

  it('counts down to the auto-release on a delivered reservation', () => {
    const soon = new Date(Date.now() + 47.5 * 3_600_000).toISOString();
    const el: HTMLElement = render(
      view({
        status: 'delivered',
        autoReleaseAt: soon,
        actions: ['confirm', 'dispute'],
        steps: [
          { key: 'paid', state: 'done' },
          { key: 'delivered', state: 'done' },
          { key: 'confirmed', state: 'current' },
          { key: 'released', state: 'upcoming' }
        ]
      })
    ).nativeElement;
    expect(el.textContent).toContain('Livré · à confirmer');
    expect(el.textContent).toContain('dans 47 h');
  });

  it('marks a refunded reservation as such and drops the progress bar', () => {
    const el: HTMLElement = render(view({ group: 'cancelled', status: 'cancelled', payment: { status: 'refunded', amount: 76500 }, actions: [] }))
      .nativeElement;
    expect(el.textContent).toContain('Remboursée');
    expect(el.querySelector('[role="img"]')).toBeNull();
  });
});
