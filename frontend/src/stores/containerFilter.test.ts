import { describe, it, expect, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useContainerFilterStore } from './containerFilter';

describe('containerFilter', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it('toggleCard folds and unfolds a single card', () => {
    const s = useContainerFilterStore();
    s.toggleCard('c1');
    expect(s.isCollapsed('c1')).toBe(true);
    expect(s.isCollapsed('c2')).toBe(false);
    s.toggleCard('c1');
    expect(s.isCollapsed('c1')).toBe(false);
  });

  it('setCollapsed folds/unfolds a batch of cards (legend bulk toggle)', () => {
    const s = useContainerFilterStore();
    s.setCollapsed(['a', 'b', 'c'], true);
    expect(s.isCollapsed('a')).toBe(true);
    expect(s.isCollapsed('b')).toBe(true);
    expect(s.isCollapsed('c')).toBe(true);
    s.setCollapsed(['a', 'c'], false);
    expect(s.isCollapsed('a')).toBe(false);
    expect(s.isCollapsed('b')).toBe(true); // untouched
    expect(s.isCollapsed('c')).toBe(false);
  });

  it('clear resets state', () => {
    const s = useContainerFilterStore();
    s.setCollapsed(['a', 'b'], true);
    s.clear();
    expect(s.isCollapsed('a')).toBe(false);
    expect(s.isCollapsed('b')).toBe(false);
  });
});
