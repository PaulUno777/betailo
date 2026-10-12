import { LocaleService } from '../../../core/i18n/locale.service';

/** How long ago `iso` was, the way a queue shows it: "1 j 18 h", "22 h",
 * "35 min". Staff read it to pick the oldest request first. */
export function elapsed(iso: string, locale: LocaleService, now: number = Date.now()): string {
  const minutes = Math.max(0, Math.floor((now - new Date(iso).getTime()) / 60_000));
  const days = Math.floor(minutes / 1440);
  const hours = Math.floor((minutes % 1440) / 60);
  if (days > 0) return locale.t('elapsed.daysHours', { days, hours });
  if (hours > 0) return locale.t('elapsed.hours', { hours });
  return locale.t('elapsed.minutes', { minutes: Math.max(1, minutes) });
}

/** A request waiting longer than the review target. */
export function isLate(iso: string, targetHours: number, now: number = Date.now()): boolean {
  return now - new Date(iso).getTime() > targetHours * 3_600_000;
}
