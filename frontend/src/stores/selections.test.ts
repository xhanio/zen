import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useSelectionsStore } from './selections';

vi.mock('../api/client', () => ({
  listCardSelections: vi.fn(),
}));
import { listCardSelections } from '../api/client';

describe('selections store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.mocked(listCardSelections).mockReset();
  });

  it('caches selections per card', async () => {
    vi.mocked(listCardSelections).mockResolvedValue({
      selections: [
        {
          message_id: 'm1', conversation_id: 'c1', selection_text: 'beta',
          selection_start: 6, selection_end: 10, selection_seq: 2,
          created_at: '2026-07-27T00:00:00Z',
        },
      ],
    });
    const store = useSelectionsStore();
    await store.load('card1');
    expect(store.byCard['card1']).toHaveLength(1);
    expect(store.byCard['card1'][0].message_id).toBe('m1');
  });

  // Underlines are a secondary surface: a failure must not blank the card.
  it('degrades to an empty list when the request fails', async () => {
    vi.mocked(listCardSelections).mockRejectedValue(new Error('boom'));
    const store = useSelectionsStore();
    await store.load('card1');
    expect(store.byCard['card1']).toEqual([]);
  });

  it('degrades to an empty list on a malformed payload', async () => {
    vi.mocked(listCardSelections).mockResolvedValue({} as never);
    const store = useSelectionsStore();
    await store.load('card1');
    expect(store.byCard['card1']).toEqual([]);
  });

  it('records stale message ids', () => {
    const store = useSelectionsStore();
    store.markStale(['m1', 'm2']);
    expect(store.stale['m1']).toBe(true);
    expect(store.stale['m2']).toBe(true);
    expect(store.stale['m3']).toBeUndefined();
  });

  it('adds a just-sent selection locally, without fetching', () => {
    const store = useSelectionsStore();
    store.add('card1', {
      id: 'm9', conversation_id: 'c1', role: 'user', content: 'q',
      selection_text: 'brave', selection_start: 6, selection_end: 11,
      selection_seq: 2, created_at: '2026-07-27T00:00:00Z',
    } as never);
    expect(store.byCard['card1']).toHaveLength(1);
    expect(store.byCard['card1'][0].message_id).toBe('m9');
    expect(store.byCard['card1'][0].selection_start).toBe(6);
    expect(listCardSelections).not.toHaveBeenCalled();
  });

  it('ignores a message with no range — those can never paint', () => {
    const store = useSelectionsStore();
    store.add('card1', {
      id: 'm10', conversation_id: 'c1', role: 'user', content: 'q',
      selection_text: 'brave', selection_start: null, selection_end: null,
      selection_seq: null, created_at: '2026-07-27T00:00:00Z',
    } as never);
    expect(store.byCard['card1'] ?? []).toHaveLength(0);
  });

  it('does not double-add the same message', () => {
    const store = useSelectionsStore();
    const m = {
      id: 'm11', conversation_id: 'c1', role: 'user', content: 'q',
      selection_text: 'brave', selection_start: 6, selection_end: 11,
      selection_seq: 1, created_at: '2026-07-27T00:00:00Z',
    } as never;
    store.add('card1', m);
    store.add('card1', m);
    expect(store.byCard['card1']).toHaveLength(1);
  });
});
