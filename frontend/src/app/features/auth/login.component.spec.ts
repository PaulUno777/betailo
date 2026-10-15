import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { AuthApi } from '../../data-access/auth/auth.api';
import { LoginComponent } from './login.component';

describe('LoginComponent', () => {
  const calls: string[] = [];
  const fakeApi = {
    login: () => of({ mfaRequired: true, mfaToken: 'challenge', methods: ['totp', 'recovery'] }),
    verifyMfa: (mfaToken: string, code: { code?: string }) => {
      calls.push(`${mfaToken}:${code.code}`);
      return of();
    }
  };

  beforeEach(() => {
    calls.length = 0;
    TestBed.configureTestingModule({
      imports: [LoginComponent],
      providers: [provideRouter([]), { provide: AuthApi, useValue: fakeApi }]
    });
  });

  it('sends the authenticator code from the second step without reloading the page', () => {
    const fixture = TestBed.createComponent(LoginComponent);
    fixture.detectChanges();
    const page = fixture.nativeElement as HTMLElement;

    const [email, password] = Array.from(page.querySelectorAll('input:not([type=checkbox])')) as HTMLInputElement[];
    email.value = 'root@betailo.test';
    email.dispatchEvent(new Event('input'));
    password.value = 'secret-password';
    password.dispatchEvent(new Event('input'));
    page.querySelector('form')!.dispatchEvent(new Event('submit', { cancelable: true }));
    fixture.detectChanges();

    const code = page.querySelector('input') as HTMLInputElement;
    code.value = '123456';
    code.dispatchEvent(new Event('input'));
    fixture.detectChanges();
    const submit = new Event('submit', { cancelable: true });
    page.querySelector('form')!.dispatchEvent(submit);

    expect(calls).toEqual(['challenge:123456']);
    // A form Angular doesn't handle would let the browser submit it,
    // reloading the page and dropping the pending sign-in.
    expect(submit.defaultPrevented).toBeTrue();
  });
});
