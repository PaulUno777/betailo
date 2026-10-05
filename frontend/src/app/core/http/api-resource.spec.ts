import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Injector, runInInjectionContext } from '@angular/core';
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
});
