import { describe, it, expect, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useLevelFilterStore } from './levelFilter';

beforeEach(() => setActivePinia(createPinia()));

describe('useLevelFilterStore', () => {
  it('ensure() seeds defaults with every entry checked + misc on', () => {
    const s = useLevelFilterStore();
    const state = s.ensure('g1', [
      { id: 'A', weight: 0, name: '原则' },
      { id: 'B', weight: 1, name: '细节' },
    ]);
    expect(state.selectedEntryIds).toEqual(['A', 'B']);
    expect(state.showMisc).toBe(true);
  });

  it('ensure() is idempotent on a known groupId with unchanged catalog', () => {
    const s = useLevelFilterStore();
    s.ensure('g1', [{ id: 'A', weight: 0, name: 'a' }]);
    s.setSelectedEntryIds('g1', []);
    const state = s.ensure('g1', [{ id: 'A', weight: 0, name: 'a' }]);
    expect(state.selectedEntryIds).toEqual([]);
  });

  it('ensure() auto-checks newly appended entries (catalog grew)', () => {
    const s = useLevelFilterStore();
    s.ensure('g1', [{ id: 'A', weight: 0, name: 'a' }]);
    s.setSelectedEntryIds('g1', []);
    const state = s.ensure('g1', [
      { id: 'A', weight: 0, name: 'a' },
      { id: 'B', weight: 0.5, name: 'b' },
    ]);
    expect(state.selectedEntryIds).toEqual(['B']);
  });

  it('setShowMisc + setSelectedEntryIds mutate state', () => {
    const s = useLevelFilterStore();
    s.ensure('g1', [{ id: 'A', weight: 0, name: 'a' }]);
    s.setShowMisc('g1', false);
    s.setSelectedEntryIds('g1', ['A']);
    expect(s.byGroup['g1']).toEqual({ selectedEntryIds: ['A'], showMisc: false });
  });

  it('same-weight entries have independent toggle state', () => {
    const s = useLevelFilterStore();
    s.ensure('g1', [
      { id: 'A', weight: 2, name: '决策' },
      { id: 'B', weight: 2, name: '细节' },
    ]);
    s.setSelectedEntryIds('g1', ['A']);
    // B is off, A is on — legend / column would show only A's column
    expect(s.byGroup['g1'].selectedEntryIds).toEqual(['A']);
  });
});
