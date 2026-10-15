import { TestBed } from '@angular/core/testing';
import { Review } from '../../../domain/models';
import { LocaleService } from '../../../core/i18n/locale.service';
import { ReviewCardComponent } from './review-card.component';
import { useTestReference } from '../../../testing/reference.fixture';

const REVIEW: Review = {
  id: 'rv1',
  rating: 4,
  comment: 'Très bonne souche.',
  criteria: { health: 5 },
  tags: ['responsive_farmer'],
  photos: ['https://cdn.test/a.jpg'],
  thumbs: ['https://cdn.test/a.thumb.jpg'],
  helpfulCount: 7,
  votedByMe: false,
  mine: false,
  reply: { text: 'Merci Mireille !', at: '2026-05-03T10:00:00Z' },
  authorName: 'Mireille N.',
  reservationRef: 'RSV-1702',
  createdAt: '2026-05-01T10:00:00Z'
};

describe('ReviewCardComponent', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({ imports: [ReviewCardComponent] });
    TestBed.inject(LocaleService).setLocale('fr');
    useTestReference();
  });

  function render(review: Review, canVote = true) {
    const fixture = TestBed.createComponent(ReviewCardComponent);
    fixture.componentRef.setInput('review', review);
    fixture.componentRef.setInput('canVote', canVote);
    fixture.detectChanges();
    return fixture;
  }

  it('shows the public name, tags, verified badge, photo and the farmer reply', () => {
    const text = (render(REVIEW).nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Mireille N.');
    expect(text).toContain('Éleveur réactif');
    expect(text).toContain('Avis vérifié · RSV-1702');
    expect(text).toContain('Merci Mireille !');
    expect(text).toContain('Utile · 7');
  });

  it('emits a helpful vote, but never offers one on your own review', () => {
    const fixture = render(REVIEW);
    let votes = 0;
    fixture.componentInstance.helpful.subscribe(() => votes++);
    const button = (fixture.nativeElement as HTMLElement).querySelector<HTMLButtonElement>('button[aria-pressed]')!;
    button.click();
    expect(votes).toBe(1);

    const own = render({ ...REVIEW, mine: true });
    expect((own.nativeElement as HTMLElement).querySelector<HTMLButtonElement>('button[aria-pressed]')!.disabled).toBeTrue();
  });

  it("shows the author's profile photo, or their initials without one", () => {
    const withPhoto = render({ ...REVIEW, authorAvatarUrl: 'https://cdn.test/me.thumb.jpg' }).nativeElement as HTMLElement;
    expect(withPhoto.querySelector('bt-avatar img')?.getAttribute('src')).toBe('https://cdn.test/me.thumb.jpg');

    const without = render(REVIEW).nativeElement as HTMLElement;
    expect(without.querySelector('bt-avatar img')).toBeNull();
    expect(without.querySelector('bt-avatar')?.textContent?.trim()).toBe('MN');
  });
});
