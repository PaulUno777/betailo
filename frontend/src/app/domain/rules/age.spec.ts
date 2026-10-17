import { AGE_CHOICES_DAYS, ageLabel } from './age';

describe('ageLabel', () => {
  it('says days for the first week', () => {
    expect(ageLabel(1)).toEqual({ key: 'age.days', count: 1 });
  });

  it('says weeks up to twelve weeks', () => {
    expect(ageLabel(7)).toEqual({ key: 'age.weeks', count: 1 });
    expect(ageLabel(84)).toEqual({ key: 'age.weeks', count: 12 });
  });

  it('says months, then years', () => {
    expect(ageLabel(122)).toEqual({ key: 'age.months', count: 4 });
    expect(ageLabel(274)).toEqual({ key: 'age.months', count: 9 });
    expect(ageLabel(365)).toEqual({ key: 'age.years', count: 1 });
    expect(ageLabel(730)).toEqual({ key: 'age.years', count: 2 });
  });

  it('gives every wizard choice a distinct label', () => {
    const labels = AGE_CHOICES_DAYS.map((d) => JSON.stringify(ageLabel(d)));
    expect(new Set(labels).size).toBe(labels.length);
  });
});
