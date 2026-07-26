import { describe, it, expect } from 'vitest';
import { buildTimeline } from './timeline';
import type { CardSnapshot, Message } from '../types/entity';

const msg = (id: string, at: string): Message => ({
  id, conversation_id: 'c', role: 'user', content: id,
  selection_text: null, session_id: null, session_cwd: null, created_at: at,
});

const snap = (id: string, at: string, over: Partial<CardSnapshot> = {}): CardSnapshot => ({
  id, card_id: 'card', card_title: 'Card', seq: 2, title: 't', summary: '',
  content: '', format: 'markdown', actor: 'agent', conversation_id: 'c',
  change_kind: 'update', diff: '', diff_truncated: false,
  lines_added: 2, lines_removed: 1, created_at: at, ...over,
});

describe('buildTimeline', () => {
  it('interleaves records with messages by created_at', () => {
    const items = buildTimeline(
      [msg('m1', '2026-07-25T10:00:00Z'), msg('m2', '2026-07-25T10:05:00Z')],
      [snap('s1', '2026-07-25T10:02:00Z')],
    );
    expect(items.map((i) => i.kind)).toEqual(['message', 'snapshot', 'message']);
  });

  it('leaves a run of three records unfolded', () => {
    const items = buildTimeline(
      [msg('m1', '2026-07-25T10:00:00Z')],
      ['s1', 's2', 's3'].map((id, i) => snap(id, `2026-07-25T10:0${i + 1}:00Z`)),
    );
    expect(items.filter((i) => i.kind === 'snapshot')).toHaveLength(3);
    expect(items.some((i) => i.kind === 'snapshot-group')).toBe(false);
  });

  it('folds a run of four or more and sums their counts', () => {
    const items = buildTimeline(
      [msg('m1', '2026-07-25T10:00:00Z')],
      ['s1', 's2', 's3', 's4'].map((id, i) => snap(id, `2026-07-25T10:0${i + 1}:00Z`)),
    );
    const group = items.find((i) => i.kind === 'snapshot-group');
    expect(group).toBeDefined();
    if (group?.kind !== 'snapshot-group') throw new Error('unreachable');
    expect(group.snapshots).toHaveLength(4);
    expect(group.linesAdded).toBe(8);
    expect(group.linesRemoved).toBe(4);
  });

  it('a message between records splits the run so neither side folds', () => {
    const items = buildTimeline(
      [msg('m1', '2026-07-25T10:00:00Z'), msg('m2', '2026-07-25T10:03:00Z')],
      [
        snap('s1', '2026-07-25T10:01:00Z'), snap('s2', '2026-07-25T10:02:00Z'),
        snap('s3', '2026-07-25T10:04:00Z'), snap('s4', '2026-07-25T10:05:00Z'),
      ],
    );
    expect(items.some((i) => i.kind === 'snapshot-group')).toBe(false);
    expect(items.filter((i) => i.kind === 'snapshot')).toHaveLength(4);
  });

  it('breaks created_at ties by id so ordering is stable', () => {
    const at = '2026-07-25T10:00:00Z';
    const items = buildTimeline([msg('m1', at)], [snap('s1', at)]);
    expect(items.map((i) => i.kind)).toEqual(['message', 'snapshot']);
  });

  // The reason no message-bus event is needed: an edit that happened before
  // the reply sorts before it regardless of which arrived at the client first.
  it('sorts an edit above the reply that announced it', () => {
    const items = buildTimeline(
      [msg('m1', '2026-07-25T10:00:00Z'), msg('m2-reply', '2026-07-25T10:09:00Z')],
      [snap('s1', '2026-07-25T10:08:00Z')],
    );
    expect(items.map((i) => i.id)).toEqual(['m1', 's1', 'm2-reply']);
  });

  it('handles an empty thread', () => {
    expect(buildTimeline([], [])).toEqual([]);
  });
});
