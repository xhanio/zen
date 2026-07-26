import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import SnapshotRecord from './SnapshotRecord.vue';
import type { CardSnapshot } from '../../types/entity';

const snap = (over: Partial<CardSnapshot> = {}): CardSnapshot => ({
  id: 's1', card_id: 'c1', card_title: 'Decision Two', seq: 8,
  title: 't', summary: '', content: '', format: 'markdown',
  actor: 'agent', conversation_id: 'conv1', change_kind: 'update',
  diff: '', diff_truncated: false, lines_added: 5, lines_removed: 3,
  created_at: '2026-07-25T10:00:00Z', ...over,
});

describe('SnapshotRecord', () => {
  it('shows the card title, the seq pair, and the counts', () => {
    const w = mount(SnapshotRecord, { props: { snapshot: snap() } });
    expect(w.text()).toContain('Decision Two');
    expect(w.text()).toContain('#7→#8');
    expect(w.text()).toContain('+5');
    expect(w.text()).toContain('−3');
  });

  it('labels a decompose record as a restructure, not a deletion', () => {
    const w = mount(SnapshotRecord, { props: { snapshot: snap({ change_kind: 'decompose' }) } });
    expect(w.find('[data-test="snapshot-kind"]').text()).toBe('拆分为章节');
  });

  it('omits the seq pair on a baseline', () => {
    const w = mount(SnapshotRecord, { props: { snapshot: snap({ seq: 1, change_kind: 'baseline' }) } });
    expect(w.text()).not.toContain('#0');
  });

  it('is not a conversation turn', () => {
    const w = mount(SnapshotRecord, { props: { snapshot: snap() } });
    expect(w.find('[data-test="turn"]').exists()).toBe(false);
    expect(w.find('[data-test="snapshot-record"]').exists()).toBe(true);
  });

  it('emits open with the snapshot id when clicked', async () => {
    const w = mount(SnapshotRecord, { props: { snapshot: snap() } });
    await w.find('[data-test="snapshot-record"]').trigger('click');
    expect(w.emitted('open')?.[0]).toEqual(['s1']);
  });

  it('renders a folded group as one row with summed counts', () => {
    const w = mount(SnapshotRecord, {
      props: {
        group: {
          snapshots: [snap(), snap({ id: 's2', card_title: 'Decision Four' })],
          linesAdded: 10,
          linesRemoved: 6,
        },
      },
    });
    expect(w.text()).toContain('2 张卡片被修改');
    expect(w.text()).toContain('+10');
    expect(w.text()).toContain('−6');
  });

  // Folding must not lose access to the individual records.
  it('expands a group into its records, preserving order', async () => {
    const w = mount(SnapshotRecord, {
      props: {
        group: {
          snapshots: [snap({ id: 'a', card_title: 'First' }), snap({ id: 'b', card_title: 'Second' })],
          linesAdded: 4,
          linesRemoved: 2,
        },
      },
    });
    await w.find('[data-test="snapshot-record"]').trigger('click');

    const rows = w.findAll('[data-test="snapshot-record"]');
    expect(rows).toHaveLength(3); // the group row plus its two children
    expect(w.text()).toContain('First');
    expect(w.text()).toContain('Second');

    await rows[1].trigger('click');
    expect(w.emitted('open')?.[0]).toEqual(['a']);
  });
});
