/** The brand kit's animal drawings (design/betailo-brand/icons/animaux):
 * 64x64, 2.5px round stroke, `currentColor`. Inner SVG markup only. The
 * names match the API's domain.AnimalIcons: the reference data says
 * which drawing a category or species wears. */
export const ANIMAL_NAMES = [
  'poule', 'coq', 'poussin', 'dinde', 'canard', 'pintade', 'chevre', 'mouton', 'vache', 'porc', 'lapin', 'poisson'
] as const;

export type AnimalName = (typeof ANIMAL_NAMES)[number];

export const ANIMAL_PATHS: Readonly<Record<AnimalName, string>> = {
  poule:
    '<path d="M46 26C50 32 52 38 49 44C46 50 39 53 31 53C21 53 14 47 13 38L11 22L19 28L21 19L27 29C31 31 36 31 39 28L41 24"/><circle cx="45" cy="20" r="6"/><path d="M41 14C40 10 44 9 45 13C46 9 50 10 49 14"/><path d="M51 18L56 20.5L51 23"/><path d="M47 26C47 29 50 30 50 26"/><path d="M24 38C28 45 37 45 41 38"/><path d="M29 53L28 59M25 60L28 59L31 60M37 53L38 59M35 60L38 59L41 60"/><circle cx="46.5" cy="19" r="1.3" fill="currentColor" stroke="none"/>',
  coq:
    '<path d="M42 25C47 32 48 42 42 48C37 53 29 53 24 50C19 47 17 42 18 36"/><path d="M18 36C10 32 7 22 11 13C12.5 10 15 10 15.5 12.5M19.5 34.5C17 27 20 19 27 16.5C29 16 30 17.5 29 19M19.5 34.5C26 31 32 28 36.5 23"/><circle cx="41" cy="19.5" r="5.5"/><path d="M36 15C35 10 38 9 39.5 12.5C39.5 8 43.5 8 43.5 12C44.5 8 48.5 10 46.5 15"/><path d="M46.5 18L51.5 20.5L46.5 22.5"/><path d="M43 25C43 30 47 30 46.5 24"/><path d="M27 40C31 46 38 45 40 39"/><path d="M30 52L29 59M26 60L29 59L32 60M37 52L38 59M35 60L38 59L41 60M37.5 56L35 55"/><circle cx="42.5" cy="18.5" r="1.3" fill="currentColor" stroke="none"/>',
  poussin:
    '<circle cx="38" cy="23" r="9"/><path d="M31 29C21 31 15 38 16 46C18 53 26 56 34 55C44 54 50 46 48 38C47 34 46 32 44 30"/><path d="M47 21L52.5 23.5L47 26"/><path d="M35 14C34 10 37 9 38.5 11.5C39 9 42 9 41.5 13"/><path d="M23 42C27 49 35 49 38 43"/><path d="M30 55V60M27.5 61L30 60L32.5 61M37 55V60M34.5 61L37 60L39.5 61"/><circle cx="40.5" cy="21.5" r="1.4" fill="currentColor" stroke="none"/>',
  dinde:
    '<path d="M24.1 50.1A19 19 0 1 1 36.5 14.1"/><path d="M22.2 39.8A11 11 0 0 1 32.8 21.4"/><path d="M19.7 35.8L12.1 38.5M19.7 28.2L12.1 25.5M24.5 22.5L20.5 15.5M31.9 21.2L33.3 13.3"/><path d="M41 20.5C38 25 33 28 28 32C22 37 22 47 28 51C34 54 44 52 46 44C47 38 46 30 45.5 21"/><circle cx="44" cy="17" r="4"/><path d="M47.5 15.5L52 17L47.5 19"/><path d="M42.5 13C41 10 43 7 44.5 9.5"/><path d="M47 19.5C48.5 23.5 47.5 27 46 26.5"/><path d="M31 40C34 45 40 45 42 40"/><path d="M32 53L31.5 59M29 60L31.5 59L34 60M40 52L41 59M38.5 60L41 59L43.5 60"/><circle cx="45" cy="16.3" r="1.2" fill="currentColor" stroke="none"/>',
  canard:
    '<circle cx="42" cy="18" r="6"/><path d="M47.5 16C52 15 56 16 56.5 18.5C56.5 21 52 21.5 47.5 21"/><path d="M45.5 23C44.5 28 45 32 49 35"/><path d="M37 20.5C37 26 38 29 36 32C28 33 19 31 11 26C11 30 12.5 33 14 35"/><path d="M14 35C16 44 24 48 33 48C43 48 51 45 52 40C52 37.5 51 36 49 35"/><path d="M22 37C27 43 37 43 41 37C35 37.5 28 36 22 37Z"/><path d="M9 55C13 53 17 57 21 55S29 53 33 55S41 57 45 55S53 53 57 55"/><circle cx="43.5" cy="16.5" r="1.3" fill="currentColor" stroke="none"/>',
  pintade:
    '<path d="M45 30C47 25 48 20 48 16"/><path d="M41 27C35 24 26 24 20 28C14 32 11 38 12 44"/><path d="M12 44C18 50 40 51 48 44C51 40 51 35 49 32C48 31 46 30 45 30"/><circle cx="49" cy="13" r="4"/><path d="M47 9.5L48.5 5L51 9.5"/><path d="M52.5 12L57 13.5L52.5 15.5"/><path d="M48 17C47 20 49 21 50.5 17"/><g fill="currentColor" stroke="none"><circle cx="22" cy="35" r="1.4"/><circle cx="29" cy="31" r="1.4"/><circle cx="36" cy="30" r="1.4"/><circle cx="27" cy="39" r="1.4"/><circle cx="34" cy="37" r="1.4"/><circle cx="41" cy="35" r="1.4"/><circle cx="20" cy="42" r="1.4"/><circle cx="32" cy="44" r="1.4"/><circle cx="40" cy="42" r="1.4"/><circle cx="49.5" cy="12.3" r="1.1"/></g><path d="M26 48L25 57M22.5 58L25 57L27.5 58M37 48L38 57M35.5 58L38 57L40.5 58"/>',
  chevre:
    '<path d="M15 27C22 25 34 25 41 26L45 18"/><path d="M45 18C47 14 51 13 53.5 15.5L58 23C58.5 26 55.5 27.5 53 26L49.5 24"/><path d="M53 26.5L52 32"/><path d="M48 14.5C46 9 42 6.5 38 6"/><path d="M46.5 19L41.5 21.5"/><path d="M49.5 24L45 30V38C45 41 43 42 40 42H21C17.5 42 15 40 15 36V27L11.5 23"/><path d="M19.5 41V56M25 42V56M36 42V56M41.5 41V56"/><circle cx="51" cy="18.5" r="1.3" fill="currentColor" stroke="none"/>',
  mouton:
    '<path d="M14 36A5 5 0 0 1 19 28A6 6 0 0 1 29 24A6 6 0 0 1 39 25A5 5 0 0 1 44 31A5 5 0 0 1 43 40A5 5 0 0 1 36 45A6 6 0 0 1 26 45A6 6 0 0 1 18 44A5 5 0 0 1 14 36Z"/><path d="M44 28C46 24 51 23 54 26C57 29 57 35 54 38C52 40 48 40 46 38"/><path d="M47 27C45 25 42 25 40.5 26.5"/><path d="M20 45V56M26 46V56M35 46V56M40 44V56"/><circle cx="51.5" cy="30" r="1.3" fill="currentColor" stroke="none"/>',
  vache:
    '<path d="M14 28C19 26.5 25 26.5 29 27C31 20 38 20 40 26.5L45 28.5"/><path d="M45 28.5L50 26.5C54 26.5 56 30 58 36C58.5 39.5 55 40.5 52.5 38.5L48.5 35"/><path d="M49 26.5C49 21.5 52 19 56 19M47 28C44.5 24.5 41.5 23.5 39 24.5"/><path d="M48.5 35C47 40 45 42 42 42H18C15.5 42 14 40 14 37V28"/><path d="M14 30C10.5 34 10 39 10.5 45M9 45.5L10.5 48L12 45.5"/><path d="M18.5 42V56M24 42V56M36 42V56M41.5 42V56"/><path d="M45 30.5L41.5 32"/><circle cx="52" cy="30.5" r="1.3" fill="currentColor" stroke="none"/>',
  porc:
    '<path d="M50 30C48 26 42 22 32 22C22 22 14 27 14 36C14 44 22 48 32 48C42 48 48 44 50 38"/><path d="M50 30H54C56 30 57.5 32 57.5 34S56 38 54 38H50Z"/><path d="M40 24L44 17L48 26"/><path d="M14 34C10 34.5 9.5 30 12 30C14 30 13.5 33.5 10.5 33.5"/><path d="M20 46V54M26 47.5V54M38 47.5V54M44 45.5V54"/><g fill="currentColor" stroke="none"><circle cx="43.5" cy="30.5" r="1.3"/><circle cx="53" cy="32.5" r="1"/><circle cx="53" cy="35.5" r="1"/></g>',
  lapin:
    '<circle cx="43" cy="30" r="8"/><path d="M40 22.5C37 15 37 8 41 4.5C44 8 45 15 44.5 22"/><path d="M46.5 23C48.5 15 52 10 55.5 9.5C56 15 53.5 20 50 24.5"/><path d="M37 34.5C34 31 28 30 23 32C16 35 13 43 17 50H44C46 50 46 47 44 46L40 44.5"/><path d="M40 44.5C42 42 42 39 40 37.5"/><circle cx="14.5" cy="44.5" r="3.5"/><g fill="currentColor" stroke="none"><circle cx="45.5" cy="28.5" r="1.3"/><circle cx="50.5" cy="31.5" r="1.1"/></g>',
  poisson:
    '<path d="M53 32C45 21 27 19 17 28.5L8.5 21V43L17 35.5C27 45 45 43 53 32Z"/><path d="M24 23L26 15L30 21L34 14L37 20.5L40.5 16.5L42 23"/><path d="M38 26C36 29.5 36 34.5 38 38"/><path d="M28 41L30 47L34 42"/><path d="M24 31C26 33 28 33 30 31M28 35C30 37 32 37 34 35"/><circle cx="45" cy="30" r="2.2"/><circle cx="57" cy="22" r="1.6"/><circle cx="59.5" cy="16" r="1.1"/>'
};

export function isAnimalName(name: string): name is AnimalName {
  return (ANIMAL_NAMES as readonly string[]).includes(name);
}
