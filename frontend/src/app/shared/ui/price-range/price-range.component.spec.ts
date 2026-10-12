import { TestBed } from '@angular/core/testing';
import { PriceRangeComponent, PriceRangeValue } from './price-range.component';

describe('PriceRangeComponent', () => {
  function render(value: PriceRangeValue) {
    TestBed.configureTestingModule({ imports: [PriceRangeComponent] });
    const fixture = TestBed.createComponent(PriceRangeComponent);
    fixture.componentRef.setInput('bounds', { min: 1000, max: 30000 });
    fixture.componentRef.setInput('value', value);
    fixture.detectChanges();
    const emitted: PriceRangeValue[] = [];
    fixture.componentInstance.valueChange.subscribe((v) => emitted.push(v));
    const [low, high] = Array.from((fixture.nativeElement as HTMLElement).querySelectorAll('input'));
    return { low, high, emitted };
  }

  function move(input: HTMLInputElement, value: number) {
    input.value = String(value);
    input.dispatchEvent(new Event('change'));
  }

  it('starts at the bounds when no limit is set', () => {
    const { low, high } = render({ min: null, max: null });
    expect(low.value).toBe('1000');
    expect(high.value).toBe('30000');
  });

  it('emits the picked limits, and none when a thumb goes back to its end', () => {
    const { low, high, emitted } = render({ min: null, max: null });
    move(low, 5000);
    expect(emitted.pop()).toEqual({ min: 5000, max: null });
    move(high, 30000);
    expect(emitted.pop()).toEqual({ min: null, max: null });
  });

  it('never lets the low thumb pass the high one', () => {
    const { low, emitted } = render({ min: null, max: 8000 });
    move(low, 20000);
    expect(emitted.pop()).toEqual({ min: 8000, max: 8000 });
  });
});
