import { BadgeTone } from './status-registry';

/** A forum category is free text (any user can type one), so there is
 * no fixed category → colour map to look up — this just hashes the
 * string to one of a few tones, deterministically, purely so the same
 * category always reads as the same colour and different categories are
 * easier to tell apart at a glance. */
const TONES: readonly BadgeTone[] = ['clay', 'forest', 'sky', 'amber', 'plum', 'alert', 'stone'];

export function categoryTone(category: string): BadgeTone {
  let hash = 0;
  for (let i = 0; i < category.length; i++) {
    hash = (hash * 31 + category.charCodeAt(i)) | 0;
  }
  return TONES[Math.abs(hash) % TONES.length];
}
