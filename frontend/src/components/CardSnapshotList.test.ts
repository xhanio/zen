import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';

vi.mock('../api/client', () => ({ listSnapshots: vi.fn(), getSnapshot: vi.fn() }));
import { listSnapshots } from '../api/client';
import CardSnapshotList from './CardSnapshotList.vue';

const row = (over: Record<string, unknown> = {}) => ({
  id: 's2', card_id: 'c1', seq: 2, actor: 'user', conversation_id: null,
  change_kind: 'update', lines_added: 1, lines_removed: 0, diff: '',
  diff_truncated: false, title: 't', summary: '', content: '', format: 'markdown',
  created_at: '2026-07-25T10:01:00Z', ...over,
});

describe('CardSnapshotList', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  // This list exists precisely for the rows the conversation timeline cannot
  // show: hand edits and backfilled baselines.
  it('lists every snapshot, including ones with no conversation', async () => {
    (listSnapshots as ReturnType<typeof vi.fn>).mockResolvedValue({
      snapshots: [
        row(),
        row({ id: 's1', seq: 1, actor: 'system', change_kind: 'baseline' }),
      ],
    });

    const w = mount(CardSnapshotList, { props: { cardId: 'c1' } });
    await new Promise((r) => setTimeout(r, 0));

    expect(w.findAll('[data-test="snapshot-list-row"]')).toHaveLength(2);
    expect(w.text()).toContain('基线');
    expect(w.text()).toContain('#1');
    expect(w.text()).toContain('#2');
  });

  it('emits open with the snapshot id', async () => {
    (listSnapshots as ReturnType<typeof vi.fn>).mockResolvedValue({ snapshots: [row()] });
    const w = mount(CardSnapshotList, { props: { cardId: 'c1' } });
    await new Promise((r) => setTimeout(r, 0));

    await w.find('[data-test="snapshot-list-row"]').trigger('click');
    expect(w.emitted('open')?.[0]).toEqual(['s2']);
  });

  it('shows an empty state rather than nothing', async () => {
    (listSnapshots as ReturnType<typeof vi.fn>).mockResolvedValue({ snapshots: [] });
    const w = mount(CardSnapshotList, { props: { cardId: 'c1' } });
    await new Promise((r) => setTimeout(r, 0));

    expect(w.text()).toContain('暂无快照');
  });
});
