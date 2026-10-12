export type PasswordStrength = 'weak' | 'medium' | 'strong';

/** A UI-only heuristic (length + character variety) driving the register
 * form's strength meter — never a substitute for the backend's own
 * password rules, which stay authoritative over what's actually accepted. */
export function passwordStrength(password: string): PasswordStrength {
  let score = 0;
  if (password.length >= 8) score++;
  if (password.length >= 12) score++;
  if (/[0-9]/.test(password)) score++;
  if (/[a-z]/.test(password) && /[A-Z]/.test(password)) score++;
  if (/[^A-Za-z0-9]/.test(password)) score++;
  if (score <= 2) return 'weak';
  if (score <= 3) return 'medium';
  return 'strong';
}
