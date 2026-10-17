import { relativeTime } from './relative-time';

describe('relativeTime', () => {
  const now = Date.parse('2026-10-17T12:00:00Z');
  const ago = (ms: number) => new Date(now - ms).toISOString();

  it('says minutes, then hours, within a day', () => {
    expect(relativeTime(ago(5 * 60_000), 'en', now)).toBe('5 minutes ago');
    expect(relativeTime(ago(3 * 3_600_000), 'en', now)).toBe('3 hours ago');
  });

  it('says yesterday, then days, weeks, months and years', () => {
    expect(relativeTime(ago(26 * 3_600_000), 'en', now)).toBe('yesterday');
    expect(relativeTime(ago(3 * 86_400_000), 'en', now)).toBe('3 days ago');
    expect(relativeTime(ago(14 * 86_400_000), 'en', now)).toBe('2 weeks ago');
    expect(relativeTime(ago(90 * 86_400_000), 'en', now)).toBe('3 months ago');
    expect(relativeTime(ago(800 * 86_400_000), 'en', now)).toBe('2 years ago');
  });

  it('speaks the reader’s language', () => {
    expect(relativeTime(ago(3 * 3_600_000), 'fr', now)).toBe('il y a 3 heures');
  });
});
