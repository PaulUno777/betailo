import { primaryWhatsapp, telUrl, whatsappUrl } from './contact';

describe('contact links', () => {
  it('builds a wa.me link from an E.164 number with an encoded message', () => {
    expect(whatsappUrl('+237690123456', 'Bonjour « Poussins »')).toBe(
      'https://wa.me/237690123456?text=Bonjour%20%C2%AB%20Poussins%20%C2%BB'
    );
  });

  it('builds a tel: link keeping the leading +', () => {
    expect(telUrl('+237690123456')).toBe('tel:+237690123456');
  });

  it('picks the first WhatsApp number, skipping call-only ones', () => {
    expect(
      primaryWhatsapp([
        { phone: '+237677001122', whatsapp: false },
        { phone: '+237690123456', whatsapp: true },
        { phone: '+237699999999', whatsapp: true }
      ])
    ).toBe('+237690123456');
    expect(primaryWhatsapp([{ phone: '+237677001122', whatsapp: false }])).toBeUndefined();
    expect(primaryWhatsapp(undefined)).toBeUndefined();
  });
});
