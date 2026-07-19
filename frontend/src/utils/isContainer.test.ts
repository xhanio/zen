import { describe, it, expect } from 'vitest';
import { isContainer } from './isContainer';
import type { Card } from '../types/entity';

function stub(overrides: Partial<Card> = {}): Card {
  return {
    id: 'c1',
    title: 'T',
    content: '',
    summary: '',
    format: 'markdown',
    level: null,
    genesis: '',
    deleted_at: null,
    group_id: 'g1',
    position: 0,
    tags: [],
    parent_card_id: null,
    source_conversation_id: null,
    created_at: '',
    updated_at: '',
    ...overrides,
  } as Card;
}

describe('isContainer', () => {
  it('is true when there are live children (empty content)', () => {
    expect(isContainer(stub({ content: '' }), 3)).toBe(true);
  });

  it('is true when there are live children even if content is non-empty (preamble)', () => {
    expect(isContainer(stub({ content: 'Date: 2026-07-11' }), 3)).toBe(true);
  });

  it('is false when there are no live children (even if content empty)', () => {
    expect(isContainer(stub({ content: '' }), 0)).toBe(false);
  });

  it('is false for null/undefined card', () => {
    expect(isContainer(null, 1)).toBe(false);
    expect(isContainer(undefined, 1)).toBe(false);
  });
});
