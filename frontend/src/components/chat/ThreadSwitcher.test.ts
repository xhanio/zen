import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ThreadSwitcher from './ThreadSwitcher.vue';
import { useConversationsStore } from '../../stores/conversations';
import type { Conversation } from '../../types/entity';

vi.mock('../../composables/useChatSidebar', () => ({
  useChatSidebar: () => ({ anchorKind: { value: 'card' }, anchorID: { value: 'k1' } }),
}));

const conv = (id: string): Conversation =>
  ({ id, title: 'T-' + id, anchor_kind: 'card', anchor_id: 'k1', created_at: '', last_message_at: '' } as Conversation);

describe('ThreadSwitcher', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it('lists the anchor threads and marks the active one', () => {
    const store = useConversationsStore();
    store.list = [conv('a'), conv('b')];
    store.activeID = 'a';
    const w = mount(ThreadSwitcher);
    const rows = w.findAll('[data-test="thread-row"]');
    expect(rows.length).toBe(2);
    expect(rows[0].classes().join(' ')).toContain('bg-accent-bg');
  });


  it('creates a new thread on the anchor and activates it', async () => {
    const store = useConversationsStore();
    const created = conv('new');
    const spyCreate = vi.spyOn(store, 'create').mockResolvedValue(created);
    const spyActive = vi.spyOn(store, 'setActive').mockResolvedValue();
    const w = mount(ThreadSwitcher);
    await w.find('[data-test="thread-new"]').trigger('click');
    expect(spyCreate).toHaveBeenCalledWith({ title: '', anchor_kind: 'card', anchor_id: 'k1' });
    expect(spyActive).toHaveBeenCalledWith('new');
  });
  it('loads the anchor list, NOT the global pending set', () => {
    // pending:true hits ListPendingConversations, which ignores the anchor and
    // returns the global pending list — the switcher would then show the wrong
    // threads (the browser caught this: zero rows on a card with a thread).
    const store = useConversationsStore();
    const spy = vi.spyOn(store, 'loadList').mockResolvedValue();
    mount(ThreadSwitcher);
    expect(spy).toHaveBeenCalledWith({ anchorKind: 'card', anchorID: 'k1' });
    expect(spy.mock.calls[0][0]).not.toHaveProperty('pending', true);
  });
});

