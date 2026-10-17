import { isSpecies } from '../../domain/reference/species';

/** What the sell wizard remembers between visits. Photos are not kept:
 * picked files can't be stored, so the farmer re-picks them. */
export interface SellDraft {
  species: string;
  breed: string;
  ageDays: number | null;
  quantity: number;
  unitPrice: number | null;
  /** '' until the farmer picks: the section only counts once answered. */
  when: 'now' | 'date' | '';
  date: string;
  delivery: boolean;
  description: string;
}

export const EMPTY_DRAFT: SellDraft = {
  species: '',
  breed: '',
  ageDays: null,
  quantity: 10,
  unitPrice: null,
  when: '',
  date: '',
  delivery: false,
  description: ''
};

const KEY_PREFIX = 'betailo.sellDraft.';

/** Reads a saved draft, field by field, so a draft saved by an older
 * version (or edited by hand) can't put garbage into the form. */
export function loadDraft(userId: string): SellDraft | null {
  try {
    const raw = localStorage.getItem(KEY_PREFIX + userId);
    if (!raw) return null;
    const d = JSON.parse(raw) as Partial<Record<keyof SellDraft, unknown>>;
    const num = (v: unknown): number | null => (typeof v === 'number' && Number.isFinite(v) && v >= 0 ? v : null);
    const str = (v: unknown): string => (typeof v === 'string' ? v : '');
    return {
      species: typeof d.species === 'string' && isSpecies(d.species) ? d.species : '',
      breed: str(d.breed),
      ageDays: num(d.ageDays),
      quantity: num(d.quantity) || EMPTY_DRAFT.quantity,
      unitPrice: num(d.unitPrice),
      when: d.when === 'date' || d.when === 'now' ? d.when : '',
      date: /^\d{4}-\d{2}-\d{2}$/.test(str(d.date)) ? str(d.date) : '',
      delivery: d.delivery === true,
      description: str(d.description)
    };
  } catch {
    return null;
  }
}

export function saveDraft(userId: string, draft: SellDraft): void {
  try {
    localStorage.setItem(KEY_PREFIX + userId, JSON.stringify(draft));
  } catch {
    // Private mode or a full quota: the draft just isn't kept.
  }
}

export function clearDraft(userId: string): void {
  try {
    localStorage.removeItem(KEY_PREFIX + userId);
  } catch {
    // Nothing to do.
  }
}

export function isEmptyDraft(d: SellDraft): boolean {
  return JSON.stringify(d) === JSON.stringify(EMPTY_DRAFT);
}
