import { describe, it, expect, vi, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useCardsStore } from './cards';

vi.mock('../api/client', () => ({
  listCards: vi.fn(),
  getCard: vi.fn(),
  createCard: vi.fn(),
  updateCard: vi.fn(),
  deleteCard: vi.fn(),
  restoreCard: vi.fn(),
  purgeCard: vi.fn(),
  listChildren: vi.fn(),
  reorderCard: vi.fn(),
  reviewCard: vi.fn(),
}));
import { restoreCard, purgeCard, getCard, listChildren, reorderCard, reviewCard } from '../api/client';

describe('useCardsStore.restore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it('upserts the returned live card into byID and byGroup', async () => {
    const store = useCardsStore();
    (getCard as any).mockResolvedValue({
      id: 'c1', title: 'A', content: 'x', format: 'markdown', level: null,
      group_id: 'g1', position: 0, tags: [],
      genesis: '', deleted_at: '2026-06-28T00:00:00Z',
      parent_card_id: null, source_conversation_id: null,
      created_at: 'x', updated_at: 'x',
    });
    await store.loadOne('c1');
    expect(store.byID['c1'].deleted_at).not.toBeNull();
    expect(store.byGroup['g1']).toBeUndefined();

    (restoreCard as any).mockResolvedValue({
      id: 'c1', title: 'A', content: 'x', format: 'markdown', level: null,
      group_id: 'g1', position: 0, tags: [],
      genesis: '', deleted_at: null,
      parent_card_id: null, source_conversation_id: null,
      created_at: 'x', updated_at: 'x',
    });

    await store.restore('c1');

    expect(restoreCard).toHaveBeenCalledWith('c1');
    expect(store.byID['c1'].deleted_at).toBeNull();
    expect(store.byGroup['g1']).toBeDefined();
    expect(store.byGroup['g1'].map((c) => c.id)).toContain('c1');
  });
});

describe('useCardsStore.purge', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it('removes the card from byID and byGroup', async () => {
    const store = useCardsStore();
    (getCard as any).mockResolvedValue({
      id: 'c1', title: 'A', content: 'x', format: 'markdown', level: null,
      group_id: 'g1', position: 0, tags: [],
      genesis: '', deleted_at: '2026-06-28T00:00:00Z',
      parent_card_id: null, source_conversation_id: null,
      created_at: 'x', updated_at: 'x',
    });
    await store.loadOne('c1');
    store.byGroup['g1'] = [{ ...store.byID['c1'], deleted_at: null }];

    (purgeCard as any).mockResolvedValue(undefined);

    await store.purge('c1');

    expect(purgeCard).toHaveBeenCalledWith('c1');
    expect(store.byID['c1']).toBeUndefined();
    expect(store.byGroup['g1']).toEqual([]);
  });
});

describe('useCardsStore.loadChildren', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it('populates byChildren and indexes byID', async () => {
    (listChildren as any).mockResolvedValue([
      { id: 'a', title: 'A', content: 'x', summary: '', format: 'markdown', level: null, genesis: '', deleted_at: null, group_id: 'g1', position: 0, tags: [], parent_card_id: 'p', source_conversation_id: null, created_at: '', updated_at: '' },
      { id: 'b', title: 'B', content: 'y', summary: '', format: 'markdown', level: null, genesis: '', deleted_at: null, group_id: 'g1', position: 1, tags: [], parent_card_id: 'p', source_conversation_id: null, created_at: '', updated_at: '' },
    ]);
    const s = useCardsStore();
    await s.loadChildren('p');
    expect(s.byChildren['p'].map((c) => c.id)).toEqual(['a', 'b']);
    expect(s.byID['a'].title).toBe('A');
    expect(listChildren).toHaveBeenCalledWith('p', false);
  });

  it('passes include_trashed through', async () => {
    (listChildren as any).mockResolvedValue([]);
    const s = useCardsStore();
    await s.loadChildren('p', true);
    expect(listChildren).toHaveBeenCalledWith('p', true);
  });
});

describe('useCardsStore.reorderChild', () => {
  const parent = 'P';
  function seedThreeChildren() {
    setActivePinia(createPinia());
    vi.clearAllMocks();
    const store = useCardsStore();
    const mk = (id: string, pos: number) => ({
      id, title: id.toUpperCase(), content: '', summary: '', format: 'markdown' as const,
      level_entry_id: null, genesis: '', deleted_at: null, group_id: 'g1', position: pos,
      tags: [], parent_card_id: parent, source_conversation_id: null,
      created_at: '', updated_at: '',
      review_grade: 'LGTM' as const, review_score: null, reviewed_at: null,
    });
    store.byChildren[parent] = [mk('a', 0), mk('b', 1), mk('c', 2)];
    store.byID['a'] = store.byChildren[parent][0];
    store.byID['b'] = store.byChildren[parent][1];
    store.byID['c'] = store.byChildren[parent][2];
    return store;
  }

  it('optimistically reindexes local children then calls reorderCard', async () => {
    const store = seedThreeChildren();
    (reorderCard as any).mockResolvedValue({ ...store.byID['c'], position: 0 });
    await store.reorderChild('c', 0);
    expect(reorderCard).toHaveBeenCalledWith('c', { position: 0 });
    const ids = store.byChildren[parent].map((x) => x.id);
    expect(ids).toEqual(['c', 'a', 'b']);
    expect(store.byChildren[parent].map((x) => x.position)).toEqual([0, 1, 2]);
  });

  it('is a no-op when target equals current position', async () => {
    const store = seedThreeChildren();
    (reorderCard as any).mockClear();
    await store.reorderChild('b', 1);
    expect(reorderCard).not.toHaveBeenCalled();
    expect(store.byChildren[parent].map((x) => x.id)).toEqual(['a', 'b', 'c']);
  });

  it('rolls back by reloading children on server error', async () => {
    const store = seedThreeChildren();
    (listChildren as any).mockResolvedValue([]);
    (reorderCard as any).mockRejectedValue(new Error('boom'));
    await expect(store.reorderChild('c', 0)).rejects.toThrow('boom');
    expect(listChildren).toHaveBeenCalledWith(parent, false);
  });
});

describe('useCardsStore.setReviewGrade', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  function makeLeaf(over: Record<string, unknown> = {}) {
    return {
      id: 'card-1', title: 't', content: 'body', summary: '', format: 'html' as const,
      level_entry_id: null, genesis: '', deleted_at: null, group_id: 'g1', position: 0,
      tags: [] as string[], parent_card_id: 'parent-1' as string | null, source_conversation_id: null,
      created_at: '', updated_at: '',
      review_grade: 'LGTM' as const, review_score: null as number | null, reviewed_at: null as string | null,
      ...over,
    };
  }

  it('optimistically updates local state before the network resolves', async () => {
    const store = useCardsStore();
    const leaf = makeLeaf();
    store.byID[leaf.id] = leaf as any;
    let resolve!: (v: any) => void;
    (reviewCard as any).mockReturnValue(new Promise((r) => { resolve = r; }));
    (getCard as any).mockResolvedValue({ ...leaf, id: 'parent-1', review_score: 50 });

    const promise = store.setReviewGrade('card-1', 'GRILLED');
    expect(store.byID['card-1'].review_grade).toBe('GRILLED');
    resolve({ ...leaf, review_grade: 'GRILLED' });
    await promise;
    expect(store.byID['card-1'].review_grade).toBe('GRILLED');
  });

  it('rolls back and reloads children on API failure', async () => {
    const store = useCardsStore();
    const leaf = makeLeaf({ review_grade: 'LGTM' });
    store.byID[leaf.id] = leaf as any;
    store.byChildren['parent-1'] = [leaf as any];
    (reviewCard as any).mockRejectedValue(new Error('boom'));
    (listChildren as any).mockResolvedValue([]);

    await expect(store.setReviewGrade('card-1', 'GRILLED')).rejects.toThrow('boom');
    expect(store.byID['card-1'].review_grade).toBe('LGTM');
    expect(listChildren).toHaveBeenCalledWith('parent-1', false);
  });

  it('reloads the parent card on success so ancestor review_score refreshes', async () => {
    const store = useCardsStore();
    const leaf = makeLeaf();
    store.byID[leaf.id] = leaf as any;
    (reviewCard as any).mockResolvedValue({ ...leaf, review_grade: 'GRILLED' });
    (getCard as any).mockResolvedValue({ ...leaf, id: 'parent-1', parent_card_id: null, review_score: 50 });

    await store.setReviewGrade('card-1', 'GRILLED');
    expect(getCard).toHaveBeenCalledWith('parent-1');
  });

  it('reloads the card itself when it has no parent (top-level)', async () => {
    const store = useCardsStore();
    const leaf = makeLeaf({ parent_card_id: null });
    store.byID[leaf.id] = leaf as any;
    (reviewCard as any).mockResolvedValue({ ...leaf, review_grade: 'DIGESTED' });
    (getCard as any).mockResolvedValue({ ...leaf, review_grade: 'DIGESTED', review_score: 50 });

    await store.setReviewGrade('card-1', 'DIGESTED');
    expect(getCard).toHaveBeenCalledWith('card-1');
  });
});

describe('useCardsStore.escalateReviewGrade', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  function card(over: Record<string, unknown> = {}) {
    return {
      id: 'card-1', title: 't', content: '', summary: '', format: 'html' as const,
      level_entry_id: null, genesis: '', deleted_at: null, group_id: 'g1', position: 0,
      tags: [] as string[], parent_card_id: 'parent-1' as string | null, source_conversation_id: null,
      created_at: '', updated_at: '',
      review_grade: '' as unknown as 'LGTM', review_score: null as number | null, reviewed_at: null as string | null,
      ...over,
    };
  }

  it('escalates a lower grade (ungraded → DIGESTED)', async () => {
    const store = useCardsStore();
    store.byID['card-1'] = card({ review_grade: '' }) as any;
    (reviewCard as any).mockResolvedValue(card({ review_grade: 'DIGESTED' }));
    (getCard as any).mockResolvedValue(card({ id: 'parent-1', parent_card_id: null }));
    await store.escalateReviewGrade('card-1', 'DIGESTED');
    expect(reviewCard).toHaveBeenCalledWith('card-1', { grade: 'DIGESTED' });
  });

  it('does not lower a higher grade (GRILLED stays on a DIGESTED escalate)', async () => {
    const store = useCardsStore();
    store.byID['card-1'] = card({ review_grade: 'GRILLED' }) as any;
    await store.escalateReviewGrade('card-1', 'DIGESTED');
    expect(reviewCard).not.toHaveBeenCalled();
    expect(store.byID['card-1'].review_grade).toBe('GRILLED');
  });

  it('no-ops for an unknown card', async () => {
    const store = useCardsStore();
    await store.escalateReviewGrade('nope', 'GRILLED');
    expect(reviewCard).not.toHaveBeenCalled();
  });
});
