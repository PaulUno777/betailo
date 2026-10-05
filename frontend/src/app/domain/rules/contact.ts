/** A WhatsApp click-to-chat link for an E.164 number, with a prefilled
 * message. WhatsApp is how buyers and farmers in Cameroon actually talk;
 * on a phone without it, wa.me falls back to WhatsApp Web. */
export function whatsappUrl(e164: string, message: string): string {
  return `https://wa.me/${e164.replace(/\D/g, '')}?text=${encodeURIComponent(message)}`;
}

/** `tel:` link for the same number, for a plain call. */
export function telUrl(e164: string): string {
  return `tel:${e164.replace(/[^\d+]/g, '')}`;
}
