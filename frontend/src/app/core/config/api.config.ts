import { InjectionToken } from '@angular/core';

// Set at build time: `ng build --define "BETAILO_API_URL='https://…'"`
// (see frontend/vercel.json). Absent in `ng serve` and in the tests.
declare const BETAILO_API_URL: string | undefined;

/** Without a built-in address (development), the API is on port 8080 of
 * the machine that served the page: localhost on the computer itself,
 * its LAN address (http://192.168.x.x:8080) from a phone on the network. */
function localApiUrl(): string {
  const host = typeof location !== 'undefined' && location.hostname ? location.hostname : 'localhost';
  return `http://${host.includes(':') ? `[${host}]` : host}:8080`;
}

export const API_BASE_URL = new InjectionToken<string>('API_BASE_URL', {
  providedIn: 'root',
  factory: () => (typeof BETAILO_API_URL === 'string' && BETAILO_API_URL.replace(/\/+$/, '')) || localApiUrl()
});
