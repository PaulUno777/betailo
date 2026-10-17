import { changedFields, marksOf, parsePaste, refId } from './reference-ops';
import { Change } from './reference.types';

describe('reference-ops', () => {
  it('reads a paste: names, optional English, spreadsheet rows with keys', () => {
    expect(parsePaste('Bamendjou\nBatié ; Batie\n\n')).toEqual([
      { key: 'bamendjou', fr: 'Bamendjou', en: 'Bamendjou' },
      { key: 'batie', fr: 'Batié', en: 'Batie' }
    ]);
    expect(parsePaste('key;fr;en\noies;Oies;Geese')).toEqual([{ key: 'oies', fr: 'Oies', en: 'Geese' }]);
    expect(parsePaste('Thiès\tThies')).toEqual([{ key: 'thies', fr: 'Thiès', en: 'Thies' }]);
  });

  it('sends only the fields an edit changed', () => {
    const before = { label: { fr: 'Volaille', en: 'Poultry' }, icon: 'hen', active: true };
    expect(changedFields(before, { ...before, label: { fr: 'Volaille', en: 'Fowl' } })).toEqual({ label: { fr: 'Volaille', en: 'Fowl' } });
    expect(changedFields(before, { ...before })).toBeNull();
  });

  it('marks each entry by what the pending changes did to it', () => {
    const change = (kind: 'add' | 'update' | 'move', key: string): Change =>
      ({ id: key + kind, op: { kind, ref: { entity: 'city', country: 'CM', key } }, by: { id: '', name: '' }, at: '' }) as Change;
    const marks = marksOf([change('add', 'mbe'), change('update', 'mbe'), change('update', 'dschang'), change('move', 'dschang'), change('move', 'buea')]);
    expect(marks.get(refId({ entity: 'city', country: 'CM', key: 'mbe' }))).toBe('added');
    expect(marks.get('city:CM:dschang')).toBe('edited');
    expect(marks.get('city:CM:buea')).toBe('moved');
  });
});
