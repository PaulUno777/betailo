import { InjectionToken } from '@angular/core';

// Set at build time: `ng build --define "BETAILO_API_URL='https://…'"`
// (see frontend/vercel.json). Absent in `ng serve` and in the tests.
declare const BETAILO_API_URL: string | undefined;

const LOCAL_API_URL = 'http://localhost:8080';

export const API_BASE_URL = new InjectionToken<string>('API_BASE_URL', {
  providedIn: 'root',
  factory: () =>
    (typeof BETAILO_API_URL === 'string' && BETAILO_API_URL.replace(/\/+$/, '')) || LOCAL_API_URL
});
