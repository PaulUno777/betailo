import { PhotoRejected, preparePhoto } from './photo-upload.service';

describe('preparePhoto', () => {
  it('scales a large photo to 1600px with a 640px small copy, both light JPEGs', async () => {
    const canvas = document.createElement('canvas');
    canvas.width = 2400;
    canvas.height = 1800;
    const ctx = canvas.getContext('2d')!;
    const gradient = ctx.createLinearGradient(0, 0, canvas.width, canvas.height);
    gradient.addColorStop(0, '#3a7d44');
    gradient.addColorStop(1, '#f2c14e');
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    const png = await new Promise<Blob>((resolve) => canvas.toBlob((b) => resolve(b!), 'image/png'));

    const { photo, thumb } = await preparePhoto(new File([png], 'big.png', { type: 'image/png' }), true);
    const full = await createImageBitmap(photo);
    const small = await createImageBitmap(thumb!);

    expect(photo.type).toBe('image/jpeg');
    expect(photo.size).toBeLessThanOrEqual(800 * 1024);
    expect(Math.max(full.width, full.height)).toBe(1600);
    expect(thumb!.size).toBeLessThanOrEqual(120 * 1024);
    expect(Math.max(small.width, small.height)).toBe(640);
  }, 30000);

  it('makes no small copy unless asked', async () => {
    const canvas = document.createElement('canvas');
    canvas.width = canvas.height = 10;
    const png = await new Promise<Blob>((resolve) => canvas.toBlob((b) => resolve(b!), 'image/png'));

    const out = await preparePhoto(new File([png], 'tiny.png', { type: 'image/png' }), false);

    expect(out.thumb).toBeUndefined();
  });

  it('refuses a file the browser cannot decode instead of sending it raw', async () => {
    const file = new File(['not an image'], 'doc.pdf', { type: 'application/pdf' });

    await expectAsync(preparePhoto(file, false)).toBeRejectedWith(new PhotoRejected('photo.unsupported'));
  });
});
