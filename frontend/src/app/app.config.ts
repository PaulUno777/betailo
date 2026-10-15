import { ApplicationConfig, inject, provideAppInitializer, provideBrowserGlobalErrorListeners, provideZonelessChangeDetection } from '@angular/core';
import { provideRouter, withInMemoryScrolling, withViewTransitions } from '@angular/router';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';

import { routes } from './app.routes';
import { ReferenceService } from './core/reference/reference.service';
import { AuthService } from './core/auth/auth.service';
import { authInterceptor } from './core/http/auth.interceptor';
import { errorInterceptor } from './core/http/error.interceptor';
import { refreshInterceptor } from './core/http/refresh.interceptor';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideZonelessChangeDetection(),
    provideRouter(
      routes,
      withViewTransitions(),
      withInMemoryScrolling({ anchorScrolling: 'enabled', scrollPositionRestoration: 'enabled' })
    ),
    // refreshInterceptor must come after errorInterceptor — see its own
    // comment for why array order maps to response-handling order in
    // reverse (it needs the raw HttpErrorResponse, before
    // errorInterceptor turns a 401 into an AppError).
    provideHttpClient(withInterceptors([authInterceptor, errorInterceptor, refreshInterceptor])),
    // Restores the session from the httpOnly refresh cookie on page
    // load (see docs/adr/0008) — without this, a reload would show a
    // signed-out UI until the first 401 triggered refreshInterceptor.
    // refreshSession() never throws (see AuthService), so this always
    // resolves and never blocks bootstrap on a slow or failed network
    // call beyond that one request.
    provideAppInitializer(() => firstValueFrom(inject(AuthService).refreshSession())),
    // The reference data (categories, countries, tags…) every form and
    // filter offers, loaded in parallel with the session; a failed load
    // falls back to the copy from the last visit.
    provideAppInitializer(() => firstValueFrom(inject(ReferenceService).load(), { defaultValue: undefined }))
  ]
};
