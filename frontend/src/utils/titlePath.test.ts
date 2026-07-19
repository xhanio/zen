import { describe, it, expect } from 'vitest';
import { ancestorsOf, truncateTail } from './titlePath';
import type { Card } from '../types/entity';

const card = (id: string, parent: string | null): Card =>
  ({ id, title: id, parent_card_id: parent ?? undefined }) as Card;

describe('ancestorsOf', () => {
  it('returns ancestors root-first, excluding self', () => {
    const byID = { a: card('a', null), b: card('b', 'a'), c: card('c', 'b') };
    expect(ancestorsOf(byID.c, byID).map((x) => x.id)).toEqual(['a', 'b']);
  });
  it('is empty for a top-level card', () => {
    const byID = { a: card('a', null) };
    expect(ancestorsOf(byID.a, byID)).toEqual([]);
  });
  it('stops at a parent missing from the store', () => {
    const byID = { c: card('c', 'gone') };
    expect(ancestorsOf(byID.c, byID)).toEqual([]);
  });
  it('guards against a cycle', () => {
    const byID = { a: card('a', 'b'), b: card('b', 'a') };
    expect(ancestorsOf(byID.a, byID).map((x) => x.id)).toEqual(['b']);
  });
});

describe('truncateTail', () => {
  it('keeps the last max and flags overflow', () => {
    expect(truncateTail(['a', 'b', 'c', 'd'], 2)).toEqual({ items: ['c', 'd'], overflow: true });
  });
  it('no overflow within max', () => {
    expect(truncateTail(['a', 'b'], 2)).toEqual({ items: ['a', 'b'], overflow: false });
  });
});
