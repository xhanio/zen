import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import ResizableSplitter from './ResizableSplitter.vue';

beforeEach(() => {
  // jsdom's pointer-capture isn't functional; stub so drag handlers don't throw.
  Element.prototype.setPointerCapture = vi.fn();
  Element.prototype.releasePointerCapture = vi.fn();
});

function mountSplitter(props: Record<string, unknown> = {}) {
  return mount(ResizableSplitter, {
    props: { width: 300, min: 200, max: 500, defaultWidth: 256, side: 'right', ...props },
  });
}

describe('ResizableSplitter', () => {
  it('widens when a right-side handle is dragged right', async () => {
    const w = mountSplitter({ side: 'right' });
    await w.trigger('pointerdown', { clientX: 100 });
    await w.trigger('pointermove', { clientX: 150 });
    expect(w.emitted('update:width')?.at(-1)).toEqual([350]);
  });

  it('inverts the direction for a left-side handle', async () => {
    const w = mountSplitter({ side: 'left' });
    await w.trigger('pointerdown', { clientX: 100 });
    await w.trigger('pointermove', { clientX: 50 }); // dragging left widens
    expect(w.emitted('update:width')?.at(-1)).toEqual([350]);
  });

  it('clamps to [min, max]', async () => {
    const w = mountSplitter({ width: 480, side: 'right' });
    await w.trigger('pointerdown', { clientX: 0 });
    await w.trigger('pointermove', { clientX: 200 }); // 680 → 500
    expect(w.emitted('update:width')?.at(-1)).toEqual([500]);
  });

  it('ignores pointermove with no active drag', async () => {
    const w = mountSplitter();
    await w.trigger('pointermove', { clientX: 999 });
    expect(w.emitted('update:width')).toBeUndefined();
  });

  it('nudges with arrow keys and clamps', async () => {
    const w = mountSplitter({ width: 300, nudge: 16 });
    await w.trigger('keydown', { key: 'ArrowRight' });
    expect(w.emitted('update:width')?.at(-1)).toEqual([316]);
    await w.trigger('keydown', { key: 'ArrowLeft' });
    expect(w.emitted('update:width')?.at(-1)).toEqual([284]);
  });

  it('resets to defaultWidth on Home and double-click', async () => {
    const w = mountSplitter({ width: 480, defaultWidth: 256 });
    await w.trigger('keydown', { key: 'Home' });
    expect(w.emitted('update:width')?.at(-1)).toEqual([256]);
    await w.trigger('dblclick');
    expect(w.emitted('update:width')?.at(-1)).toEqual([256]);
  });

  it('exposes separator semantics', () => {
    const w = mountSplitter({ width: 300, min: 200, max: 500 });
    expect(w.attributes('role')).toBe('separator');
    expect(w.attributes('aria-orientation')).toBe('vertical');
    expect(w.attributes('aria-valuenow')).toBe('300');
    expect(w.attributes('aria-valuemin')).toBe('200');
    expect(w.attributes('aria-valuemax')).toBe('500');
  });
});
