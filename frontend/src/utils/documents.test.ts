import { describe, it, expect } from 'vitest';
import { childrenOf, isDocument, documentsIn } from './documents';
import type { Card } from '../types/entity';

function card(over: Partial<Card>): Card {
  return {
    id: 'x', title: 'T', content: '', summary: '', format: 'markdown', level_entry_id: null,
    genesis: '', deleted_at: null, group_id: 'g1', position: 0, tags: [],
    parent_card_id: null, source_conversation_id: null, created_at: '', updated_at: '',
    review_grade: 'LGTM', review_score: null, reviewed_at: null,
    ...over,
  } as Card;
}

describe('documents helpers', () => {
  const doc = card({ id: 'doc', parent_card_id: null, updated_at: '2026-01-02' });
  const sec = card({ id: 'sec', parent_card_id: 'doc' });
  const secDeleted = card({ id: 'secx', parent_card_id: 'doc', deleted_at: '2026-01-01' });
  const leaf = card({ id: 'leaf', parent_card_id: null });
  const doc2 = card({ id: 'doc2', parent_card_id: null, updated_at: '2026-03-01' });
  const sec2 = card({ id: 'sec2', parent_card_id: 'doc2' });

  it('childrenOf returns only live children of the parent', () => {
    const kids = childrenOf([doc, sec, secDeleted, leaf], 'doc');
    expect(kids.map((c) => c.id)).toEqual(['sec']);
  });

  it('isDocument: top-level with a live child is a document', () => {
    expect(isDocument([doc, sec], doc)).toBe(true);
  });
  it('isDocument: a leaf (no children) is not a document', () => {
    expect(isDocument([leaf], leaf)).toBe(false);
  });
  it('isDocument: a top-level whose only child is deleted is not a document', () => {
    expect(isDocument([doc, secDeleted], doc)).toBe(false);
  });
  it('isDocument: a child card is not a document', () => {
    expect(isDocument([doc, sec], sec)).toBe(false);
  });
  it('isDocument: a deleted top-level is not a document', () => {
    const del = card({ id: 'd', parent_card_id: null, deleted_at: '2026-01-01' });
    const k = card({ id: 'k', parent_card_id: 'd' });
    expect(isDocument([del, k], del)).toBe(false);
  });

  it('documentsIn returns documents newest-updated first, excluding leaves', () => {
    const docs = documentsIn([doc, sec, leaf, doc2, sec2]);
    expect(docs.map((c) => c.id)).toEqual(['doc2', 'doc']);
  });
});
