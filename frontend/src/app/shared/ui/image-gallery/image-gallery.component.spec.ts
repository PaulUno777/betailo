import { TestBed } from '@angular/core/testing';
import { ImageGalleryComponent } from './image-gallery.component';

describe('ImageGalleryComponent', () => {
  const photos = ['https://cdn.test/a.jpg', 'https://cdn.test/b.jpg'];

  function render() {
    const fixture = TestBed.createComponent(ImageGalleryComponent);
    fixture.componentRef.setInput('photos', photos);
    fixture.componentRef.setInput('alt', 'Poulets');
    fixture.detectChanges();
    return fixture;
  }

  it('shows each full-size photo whole, over a blurred copy of itself', () => {
    const el: HTMLElement = render().nativeElement;
    const shown = [...el.querySelectorAll<HTMLImageElement>('img.object-contain')];
    expect(shown.map((img) => img.getAttribute('src'))).toEqual(photos);
    expect(shown[0].alt).toBe('Poulets (1/2)');
    const backdrops = el.querySelectorAll<HTMLImageElement>('img[aria-hidden="true"]');
    expect(backdrops.length).toBe(2);
    expect(backdrops[1].getAttribute('src')).toBe(photos[1]);
  });

  it('opens the tapped photo fullscreen and closes on the close button', async () => {
    const fixture = render();
    const el: HTMLElement = fixture.nativeElement;
    el.querySelectorAll<HTMLButtonElement>('[role="region"] > button')[1].click();
    fixture.detectChanges();
    const dialog = el.querySelector('dialog')!;
    expect(dialog.open).toBeTrue();
    expect(document.documentElement.style.overflow).toBe('hidden');
    await new Promise((r) => requestAnimationFrame(r));
    fixture.detectChanges();
    expect(dialog.querySelector('[aria-live]')?.textContent?.trim()).toBe('2 / 2');
    expect([...dialog.querySelectorAll('img')].map((i) => i.getAttribute('src'))).toEqual(photos);

    // The dialog's close event is queued, not fired synchronously.
    const closed = new Promise((r) => dialog.addEventListener('close', r, { once: true }));
    dialog.querySelector<HTMLButtonElement>('button')!.click();
    expect(dialog.open).toBeFalse();
    await closed;
    expect(document.documentElement.style.overflow).toBe('');
  });
});
