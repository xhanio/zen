import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';

vi.mock('../api/client', () => ({ listSnapshots: vi.fn(), getSnapshot: vi.fn() }));
import { listSnapshots } from '../api/client';
import CardSnapshotChip from './CardSnapshotChip.vue';

const row = (over: Record<string, unknown> = {}) => ({
  id: 's2', card_id: 'c1', seq: 2, actor: 'agent', conversation_id: 'conv1',
  change_kind: 'update', lines_added: 3, lines_removed: 1, diff: '',
  diff_truncated: false, title: 't', summary: '', content: '', format: 'markdown',
  created_at: '2026-07-25T10:01:00Z', ...over,
});

const baseline = () => row({ id: 's1', seq: 1, actor: 'system', change_kind: 'baseline', lines_added: 0, lines_removed: 0 });

const settle = () => new Promise((r) => setTimeout(r, 0));

describe('CardSnapshotChip', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it('lights up with a count once there is more than the baseline', async () => {
    (listSnapshots as ReturnType<typeof vi.fn>).mockResolvedValue({ snapshots: [row(), baseline()] });
    const w = mount(CardSnapshotChip, { props: { cardId: 'c1' } });
    await settle();

    expect(w.find('[data-test="snapshot-chip-count"]').text()).toBe('2');
    expect(w.find('[data-test="snapshot-chip"]').classes().join(' ')).toContain('accent');
  });

  // A card that has only ever been created has no history worth reading.
  it('stays quiet when only the baseline exists', async () => {
    (listSnapshots as ReturnType<typeof vi.fn>).mockResolvedValue({ snapshots: [baseline()] });
    const w = mount(CardSnapshotChip, { props: { cardId: 'c1' } });
    await settle();

    expect(w.find('[data-test="snapshot-chip"]').exists()).toBe(true);
    expect(w.find('[data-test="snapshot-chip-count"]').exists()).toBe(false);
  });

  it('takes no layout room until opened', async () => {
    (listSnapshots as ReturnType<typeof vi.fn>).mockResolvedValue({ snapshots: [row(), baseline()] });
    const w = mount(CardSnapshotChip, { props: { cardId: 'c1' } });
    await settle();

    expect(w.find('[data-test="snapshot-popover"]').exists()).toBe(false);
    await w.find('[data-test="snapshot-chip"]').trigger('click');
    expect(w.find('[data-test="snapshot-popover"]').exists()).toBe(true);
    expect(w.find('[data-test="snapshot-list"]').exists()).toBe(true);
  });

  it('closes on a second click, on outside click, and on Escape', async () => {
    (listSnapshots as ReturnType<typeof vi.fn>).mockResolvedValue({ snapshots: [row(), baseline()] });
    const w = mount(CardSnapshotChip, { props: { cardId: 'c1' }, attachTo: document.body });
    await settle();
    const chip = () => w.find('[data-test="snapshot-chip"]');
    const popover = () => w.find('[data-test="snapshot-popover"]');

    await chip().trigger('click');
    await chip().trigger('click');
    expect(popover().exists()).toBe(false);

    await chip().trigger('click');
    document.dispatchEvent(new MouseEvent('click'));
    await w.vm.$nextTick();
    expect(popover().exists()).toBe(false);

    await chip().trigger('click');
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    await w.vm.$nextTick();
    expect(popover().exists()).toBe(false);

    w.unmount();
  });

  it('emits open and dismisses the popover when a row is picked', async () => {
    (listSnapshots as ReturnType<typeof vi.fn>).mockResolvedValue({ snapshots: [row(), baseline()] });
    const w = mount(CardSnapshotChip, { props: { cardId: 'c1' } });
    await settle();

    await w.find('[data-test="snapshot-chip"]').trigger('click');
    await w.find('[data-test="snapshot-list-row"]').trigger('click');

    expect(w.emitted('open')?.[0]).toEqual(['s2']);
    expect(w.find('[data-test="snapshot-popover"]').exists()).toBe(false);
  });
});
