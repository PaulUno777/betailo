import { telUrl, whatsappUrl } from './contact';

describe('contact links', () => {
  it('builds a wa.me link from an E.164 number with an encoded message', () => {
    expect(whatsappUrl('+237690123456', 'Bonjour « Poussins »')).toBe(
      'https://wa.me/237690123456?text=Bonjour%20%C2%AB%20Poussins%20%C2%BB'
    );
  });

  it('builds a tel: link keeping the leading +', () => {
    expect(telUrl('+237690123456')).toBe('tel:+237690123456');
  });
});
