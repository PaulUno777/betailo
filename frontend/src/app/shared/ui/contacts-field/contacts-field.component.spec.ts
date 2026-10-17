import { filledContacts } from './contacts-field.component';

describe('filledContacts', () => {
  it('trims numbers, drops blank rows and keeps each WhatsApp flag', () => {
    expect(
      filledContacts([
        { phone: ' 690 12 34 56 ', whatsapp: true },
        { phone: '   ', whatsapp: true },
        { phone: '+237677001122', whatsapp: false }
      ])
    ).toEqual([
      { phone: '690 12 34 56', whatsapp: true },
      { phone: '+237677001122', whatsapp: false }
    ]);
  });
});
