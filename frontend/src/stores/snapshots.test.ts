import { describe, it, expect, vi, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';

vi.mock('../api/client', () => ({
  listSnapshots: vi.fn(),
  getSnapshot: vi.fn(),
}));

import { listSnapshots, getSnapshot } from '../api/client';
import { useSnapshotsStore } from './snapshots';

const snap = (over: Record<string, unknown> = {}) => ({
  id: 's1', card_id: 'c1', card_title: 'Card One', seq: 2,
  title: 't', summary: '', content: 'body', format: 'markdown',
  actor: 'agent', conversation_id: 'conv1', change_kind: 'update',
  diff: '{"fields":[],"lines":[]}', diff_truncated: false,
  lines_added: 3, lines_removed: 1, created_at: '2026-07-25T10:00:00Z',
  ...over,
});

describe('snapshots store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it('caches records per conversation', async () => {
    (listSnapshots as ReturnType<typeof vi.fn>).mockResolvedValue({ snapshots: [snap()] });
    const store = useSnapshotsStore();

    await store.loadForConversation('conv1');

    expect(store.byConversation['conv1']).toHaveLength(1);
    expect(store.byConversation['conv1'][0].card_title).toBe('Card One');
    expect(listSnapshots).toHaveBeenCalledWith({ conversationID: 'conv1' });
  });

  it('caches records per card', async () => {
    (listSnapshots as ReturnType<typeof vi.fn>).mockResolvedValue({ snapshots: [snap()] });
    const store = useSnapshotsStore();

    await store.loadForCard('c1');

    expect(store.byCard['c1']).toHaveLength(1);
    expect(listSnapshots).toHaveBeenCalledWith({ cardID: 'c1' });
  });

  it('parses the stored diff on detail load', async () => {
    (getSnapshot as ReturnType<typeof vi.fn>).mockResolvedValue({
      snapshot: snap({ diff: '{"fields":[],"lines":[{"op":"add","text":"hi"}]}' }),
      previous: snap({ id: 's0', seq: 1, content: 'old' }),
    });
    const store = useSnapshotsStore();

    await store.loadDetail('s1');

    expect(store.detail?.parsed.lines).toEqual([{ op: 'add', text: 'hi' }]);
    expect(store.detail?.previous?.content).toBe('old');
  });

  it('yields an empty diff when the payload is truncated', async () => {
    (getSnapshot as ReturnType<typeof vi.fn>).mockResolvedValue({
      snapshot: snap({ diff: '', diff_truncated: true }),
      previous: snap({ id: 's0', seq: 1, content: 'old' }),
    });
    const store = useSnapshotsStore();

    await store.loadDetail('s1');

    expect(store.detail?.parsed.lines).toEqual([]);
    expect(store.detail?.snapshot.diff_truncated).toBe(true);
  });

  // Malformed JSON must not take the card view down with it.
  it('degrades to an empty diff on malformed json', async () => {
    (getSnapshot as ReturnType<typeof vi.fn>).mockResolvedValue({
      snapshot: snap({ diff: '{not json' }),
      previous: null,
    });
    const store = useSnapshotsStore();

    await store.loadDetail('s1');

    expect(store.detail?.parsed).toEqual({ fields: [], lines: [] });
  });

  it('clears the detail view', async () => {
    (getSnapshot as ReturnType<typeof vi.fn>).mockResolvedValue({
      snapshot: snap(), previous: null,
    });
    const store = useSnapshotsStore();
    await store.loadDetail('s1');
    store.clearDetail();
    expect(store.detail).toBeNull();
  });
});
