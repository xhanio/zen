import { describe, it, expect, beforeEach, vi, type Mock } from 'vitest';

vi.mock('../api/client', () => ({
  getCard: vi.fn(),
  listChildren: vi.fn(),
}));
// Keep the real serializer/filename; only stub the impure download.
vi.mock('../utils/exportCard', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../utils/exportCard')>();
  return { ...actual, downloadText: vi.fn() };
});

import { getCard, listChildren } from '../api/client';
import { downloadText } from '../utils/exportCard';
import { BackendError } from '../types/api';
import { useCardExport } from './useCardExport';
import type { Card } from '../types/entity';

function card(overrides: Partial<Card> = {}): Card {
  return {
    id: 'c1', title: 'T', summary: '', content: '', format: 'markdown',
    level_entry_id: null, genesis: '', deleted_at: null, group_id: 'g1',
    position: 0, tags: [], parent_card_id: null, source_conversation_id: null,
    created_at: '', updated_at: '', review_grade: 'LGTM', review_score: null,
    reviewed_at: null, ...overrides,
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  const { exporting, error } = useCardExport();
  exporting.value = false;
  error.value = null;
});

describe('useCardExport.exportCard', () => {
  it('fetches the live subtree, serializes, and downloads with the right filename/mime', async () => {
    const root = card({ id: 'c1', title: 'Doc', content: 'Preamble', format: 'markdown' });
    const s1 = card({ id: 's1', title: 'Alpha', content: 'alpha body', parent_card_id: 'c1', position: 0 });
    const s2 = card({ id: 's2', title: 'Beta', content: '', parent_card_id: 'c1', position: 1 });
    const s2a = card({ id: 's2a', title: 'Beta One', content: 'nested', parent_card_id: 's2', position: 0 });
    (getCard as Mock).mockResolvedValue(root);
    const kids: Record<string, Card[]> = { c1: [s1, s2], s1: [], s2: [s2a], s2a: [] };
    (listChildren as Mock).mockImplementation(async (id: string) => kids[id] ?? []);

    await useCardExport().exportCard('c1');

    expect(getCard).toHaveBeenCalledWith('c1');
    expect(listChildren).toHaveBeenCalledWith('c1', false); // live only
    expect(downloadText).toHaveBeenCalledTimes(1);
    const [filename, mime, text] = (downloadText as Mock).mock.calls[0];
    expect(filename).toBe('Doc.md');
    expect(mime).toBe('text/markdown');
    expect(text).toBe('# Doc\n\nPreamble\n\n## Alpha\n\nalpha body\n\n## Beta\n\n### Beta One\n\nnested\n');
  });

  it('orders sections by position regardless of fetch order', async () => {
    const root = card({ id: 'c1', title: 'Doc', content: '' });
    const b = card({ id: 'b', title: 'B', content: 'b', position: 1 });
    const a = card({ id: 'a', title: 'A', content: 'a', position: 0 });
    (getCard as Mock).mockResolvedValue(root);
    const kids: Record<string, Card[]> = { c1: [b, a], a: [], b: [] }; // returned out of order
    (listChildren as Mock).mockImplementation(async (id: string) => kids[id] ?? []);

    await useCardExport().exportCard('c1');
    const text = (downloadText as Mock).mock.calls[0][2];
    expect(text.indexOf('## A')).toBeLessThan(text.indexOf('## B'));
  });

  it('on failure sets error, resets exporting, and does not download', async () => {
    (getCard as Mock).mockRejectedValue(new BackendError(404, 'not_found', 'nope'));
    const api = useCardExport();
    await api.exportCard('c1');
    expect(api.error.value).toBe('nope');
    expect(api.exporting.value).toBe(false);
    expect(downloadText).not.toHaveBeenCalled();
  });

  it('is re-entrancy guarded (a second call while exporting is a no-op)', async () => {
    const api = useCardExport();
    api.exporting.value = true; // simulate in-flight
    await api.exportCard('c1');
    expect(getCard).not.toHaveBeenCalled();
  });
});
