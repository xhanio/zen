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
});
