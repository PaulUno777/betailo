import { Component, inject } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { API_BASE_URL } from '../../../core/config/api.config';
import { errorInterceptor } from '../../../core/http/error.interceptor';
import { LocaleService } from '../../../core/i18n/locale.service';
import { TEST_CATALOG, useTestReference } from '../../../testing/reference.fixture';
import { BulkAddDialogComponent } from './bulk-add-dialog.component';
import { CategoriesViewComponent } from './categories-view.component';
import { ChangesPanelComponent } from './changes-panel.component';
import { EntryDrawerComponent } from './entry-drawer.component';
import { ReferenceStore } from './reference.store';
import { Change, Changeset } from './reference.types';

/** The views, the drawer, the bulk dialog and the changes panel, with
 * the real store, as the layout puts them together. */
@Component({
  selector: 'bt-test-host',
  imports: [CategoriesViewComponent, ChangesPanelComponent, EntryDrawerComponent, BulkAddDialogComponent],
  providers: [ReferenceStore],
  template: `
    <bt-categories-view />
    <bt-changes-panel />
    @if (store.editing(); as r) {
      <bt-entry-drawer [request]="r" (closed)="store.editing.set(null)" />
    }
    @if (store.bulking(); as b) {
      <bt-bulk-add-dialog [target]="b" (closed)="store.bulking.set(null)" />
    }
  `
})
class HostComponent {
  readonly store = inject(ReferenceStore);
}

const API = 'https://api.test';
const changeset = (rev: number, changes: Change[] = [], draft = TEST_CATALOG): Changeset => ({
  rev,
  baseVersions: TEST_CATALOG.versions,
  touched: changes.length ? ['animals'] : [],
  ready: changes.length > 0,
  problems: [],
  changes,
  draft
});

describe('reference editor flows', () => {
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HostComponent],
      providers: [provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting(), { provide: API_BASE_URL, useValue: API }]
    });
    http = TestBed.inject(HttpTestingController);
    useTestReference();
    TestBed.inject(LocaleService).setLocale('en');
  });

  async function render() {
    const fixture = TestBed.createComponent(HostComponent);
    const store = fixture.componentInstance.store;
    const loading = store.load();
    fixture.detectChanges();
    http.expectOne(`${API}/admin/reference/changeset`).flush(changeset(0));
    await loading;
    // Answer everything the page asks for: a pending request keeps
    // Angular from settling.
    http.match(`${API}/reference`).forEach((r) => r.flush(TEST_CATALOG));
    http.match(`${API}/admin/reference/usage`).forEach((r) => r.flush({ 'species:poussins': 7 }));
    http.match(`${API}/admin/reference/versions`).forEach((r) => r.flush([]));
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    const el: HTMLElement = fixture.nativeElement;
    const button = (text: string, within: ParentNode = el) => Array.from(within.querySelectorAll('button')).find((b) => b.textContent?.trim() === text)!;
    const settle = async () => {
      await new Promise((resolve) => setTimeout(resolve));
      http.match((r) => r.url.startsWith(`${API}/admin/reference/schema/`)).forEach((r) => r.flush({ type: 'object' }));
      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();
    };
    return { fixture, el, button, settle };
  }

  it('deactivates from the row, shows it pending, and undoes it', async () => {
    const { el, button, settle } = await render();
    expect(el.textContent).toContain('Day-old chicks');
    expect(el.textContent).toContain('7'); // usage

    button('Deactivate', el.querySelector('section')!).click();
    const req = http.expectOne(`${API}/admin/reference/changeset/ops`);
    expect(req.request.body).toEqual({ rev: 0, ops: [{ kind: 'update', ref: { entity: 'category', key: 'volaille' }, value: { active: false } }] });
    const change: Change = { id: 'c1', op: req.request.body.ops[0], before: { active: true }, by: { id: 'u', name: 'Awa' }, at: '2026-10-15T10:00:00Z' };
    req.flush(changeset(1, [change]));
    await settle();
    const panel = el.querySelector('bt-changes-panel')!;
    expect(panel.textContent).toContain('Deactivated');
    expect(panel.textContent).toContain('Awa');
    expect(el.querySelector('bt-categories-view')!.textContent).toContain('Edited');

    button('Undo', panel).click();
    http.expectOne(`${API}/admin/reference/changeset/ops/c1?rev=1`).flush(changeset(2));
    await settle();
    expect(panel.textContent).toContain('No pending changes');
  });

  it('pastes many kinds, flags a duplicate, then queues them under their category', async () => {
    const { el, button, settle } = await render();
    button('Paste a list', el.querySelector('section')!).click();
    await settle();
    const dialog = el.querySelector('bt-bulk-add-dialog')!;
    const area = dialog.querySelector('textarea')!;
    area.value = 'Oies ; Geese\nOies';
    area.dispatchEvent(new Event('input'));
    await settle();
    expect(dialog.textContent).toContain('Twice in the list');
    const queue = () => Array.from(dialog.querySelectorAll('button')).find((b) => b.textContent?.includes('Add 2'))!;
    expect(queue().disabled).toBeTrue();

    area.value = 'Oies ; Geese\nCailles japonaises';
    area.dispatchEvent(new Event('input'));
    await settle();
    queue().click();
    const req = http.expectOne(`${API}/admin/reference/changeset/ops`);
    expect(req.request.body.ops.map((o: { ref: { parent: string }; value: { key: string } }) => [o.ref.parent, o.value.key])).toEqual([
      ['volaille', 'oies'],
      ['volaille', 'cailles-japonaises']
    ]);
    req.flush(changeset(1));
    await settle();
    expect(el.querySelector('bt-bulk-add-dialog')).toBeNull();
  });

  it('edits one field in the drawer and sends only that field', async () => {
    const { el, button, settle } = await render();
    button('Edit', el.querySelector('section')!).click();
    await settle();
    const drawer = el.querySelector('bt-entry-drawer')!;
    const english = Array.from(drawer.querySelectorAll('label')).find((l) => l.textContent?.includes('English name'))!.querySelector('input')!;
    english.value = 'Fowl';
    english.dispatchEvent(new Event('input'));
    await settle();
    button('Save to the changes', drawer).click();
    const req = http.expectOne(`${API}/admin/reference/changeset/ops`);
    expect(req.request.body.ops).toEqual([{ kind: 'update', ref: { entity: 'category', key: 'volaille' }, value: { label: { fr: 'Volaille', en: 'Fowl' } } }]);
    req.flush(changeset(1));
    await settle();
  });
});
