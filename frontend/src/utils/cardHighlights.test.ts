import { describe, it, expect } from 'vitest';
import { buildCardHighlights } from './cardHighlights';
import type { Card, CardSelection, Reference } from '../types/entity';

const ref = (over: Partial<Reference> = {}): Reference => ({
  id: 'r1', source_card_id: 'c1', derived_card_id: 'd1', conversation_id: null,
  selection_text: 'beta', selection_start: 6, selection_end: 10, selection_seq: 1,
  created_at: '2026-07-27T00:00:00Z', ...over,
});

const sel = (over: Partial<CardSelection> = {}): CardSelection => ({
  message_id: 'm1', conversation_id: 'v1', selection_text: 'gamma',
  selection_start: 11, selection_end: 16, selection_seq: 2,
  created_at: '2026-07-27T00:00:00Z', ...over,
});

const card = (refs: Reference[]): Card => ({ id: 'c1', references: refs } as Card);

describe('buildCardHighlights', () => {
  it('puts references before message selections', () => {
    const got = buildCardHighlights(card([ref()]), [sel()]);
    expect(got.map((h) => h.kind)).toEqual(['reference', 'message']);
  });

  it('sets requireRange on messages only', () => {
    const got = buildCardHighlights(card([ref()]), [sel()]);
    expect(got[0].requireRange).toBeUndefined();
    expect(got[1].requireRange).toBe(true);
  });

  it('keys a reference by its own id and a selection by its message id', () => {
    const got = buildCardHighlights(card([ref({ id: 'REF' })]), [sel({ message_id: 'MSG' })]);
    expect(got.map((h) => h.id)).toEqual(['REF', 'MSG']);
  });

  it('carries the offsets through unchanged', () => {
    const got = buildCardHighlights(card([ref()]), [sel()]);
    expect(got[0]).toMatchObject({ text: 'beta', start: 6, end: 10 });
    expect(got[1]).toMatchObject({ text: 'gamma', start: 11, end: 16 });
  });

  // A section whose data has not loaded yet must yield an empty array, never
  // undefined — CardBody passes this straight to the painter.
  it('returns an empty array for an undefined card and undefined selections', () => {
    expect(buildCardHighlights(undefined, undefined)).toEqual([]);
  });

  it('tolerates a card with no references field at all', () => {
    expect(buildCardHighlights({ id: 'c1' } as Card, [])).toEqual([]);
  });
});
