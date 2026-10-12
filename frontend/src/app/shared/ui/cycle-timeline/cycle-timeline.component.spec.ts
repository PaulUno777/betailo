import { TestBed } from '@angular/core/testing';
import { TimelineStep } from '../../../domain/models';
import { LocaleService } from '../../../core/i18n/locale.service';
import { CycleTimelineComponent } from './cycle-timeline.component';

const STEPS: TimelineStep[] = [
  { kind: 'start', date: '2026-09-02T08:00:00Z', state: 'done' },
  { kind: 'health', date: '2026-09-14T08:00:00Z', title: 'Mirage J+12', detail: '162 œufs fécondés', state: 'current' },
  { kind: 'end', date: '2026-09-27T08:00:00Z', state: 'upcoming' }
];

describe('CycleTimelineComponent', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({ imports: [CycleTimelineComponent] });
    TestBed.inject(LocaleService).setLocale('fr');
  });

  it('renders the server steps in order, with health entries in the farmer’s words', () => {
    const fixture = TestBed.createComponent(CycleTimelineComponent);
    fixture.componentRef.setInput('steps', STEPS);
    fixture.detectChanges();
    const items = Array.from((fixture.nativeElement as HTMLElement).querySelectorAll('li')).map((li) => li.textContent ?? '');

    expect(items.length).toBe(3);
    expect(items[0]).toContain('Démarrage du cycle');
    expect(items[1]).toContain('Mirage J+12');
    expect(items[1]).toContain('162 œufs fécondés');
    expect(items[1]).toContain('en cours');
    expect(items[2]).toContain('Fin du cycle');
    expect(items[2]).toContain('à venir');
  });

  it('flags a cancelled cycle', () => {
    const fixture = TestBed.createComponent(CycleTimelineComponent);
    fixture.componentRef.setInput('steps', STEPS);
    fixture.componentRef.setInput('cancelled', true);
    fixture.detectChanges();
    expect((fixture.nativeElement as HTMLElement).textContent).toContain('Cycle annulé');
  });
});
