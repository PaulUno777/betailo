import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Injector, runInInjectionContext, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { ApiResource, apiResource } from './api-resource';

/** Lets the resource process a flushed response before it's read. */
async function settle(): Promise<void> {
  TestBed.tick();
  await new Promise((resolve) => setTimeout(resolve));
}

describe('apiResource', () => {
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpMock.verify());

  function create(): ApiResource<{ n: number }> {
    return runInInjectionContext(TestBed.inject(Injector), () => apiResource<{ n: number }>(() => '/thing'));
  }

  it('exposes the loaded value', async () => {
    const res = create();
    TestBed.tick();
    httpMock.expectOne('/thing').flush({ n: 1 });
    await settle();

    expect(res.value()).toEqual({ n: 1 });
    expect(res.hasValue()).toBeTrue();
  });

  it('reads as undefined instead of throwing once the request fails', async () => {
    const res = create();
    TestBed.tick();
    httpMock.expectOne('/thing').flush('boom', { status: 500, statusText: 'Server Error' });
    await settle();

    expect(() => res.value()).not.toThrow();
    expect(res.value()).toBeUndefined();
    expect(res.error()).toBeTruthy();
    expect(res.hasValue()).toBeFalse();
  });

  it('fetches again only when the URL itself changes', async () => {
    const farm = signal({ id: 'f1', region: 'west' });
    const species = signal('chicks');
    const res = runInInjectionContext(TestBed.inject(Injector), () =>
      apiResource<{ low: number }>(() => `/listings/price-hint?species=${species()}&region=${farm().region}`)
    );
    TestBed.tick();
    httpMock.expectOne('/listings/price-hint?species=chicks&region=west').flush({ low: 1 });
    await settle();
    expect(res.value()).toEqual({ low: 1 });

    // A refreshed farm object, same region: same URL, no new request.
    farm.set({ id: 'f1', region: 'west' });
    TestBed.tick();
    httpMock.expectNone(() => true);

    species.set('broilers');
    TestBed.tick();
    httpMock.expectOne('/listings/price-hint?species=broilers&region=west').flush({ low: 2 });
    await settle();
    expect(res.value()).toEqual({ low: 2 });
  });

  it('waits while the URL is undefined', () => {
    const id = signal<string | undefined>(undefined);
    runInInjectionContext(TestBed.inject(Injector), () => apiResource(() => (id() ? `/forum/posts/${id()}` : undefined)));
    TestBed.tick();
    httpMock.expectNone(() => true);
    id.set('p1');
    TestBed.tick();
    httpMock.expectOne('/forum/posts/p1').flush({});
  });
});
