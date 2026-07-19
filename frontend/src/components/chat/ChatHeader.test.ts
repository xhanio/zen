import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount, RouterLinkStub } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ChatHeader from './ChatHeader.vue';
import { useConversationsStore } from '../../stores/conversations';
import type { Conversation } from '../../types/entity';

vi.mock('../../composables/useChatSidebar', () => ({
  useChatSidebar: () => ({ anchorKind: { value: 'card' }, anchorID: { value: 'k1' }, close: vi.fn() }),
}));

const mountHeader = (opts: { attachTo?: Element } = {}) =>
  mount(ChatHeader, {
    attachTo: opts.attachTo,
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        'router-link': RouterLinkStub,
        ThreadSwitcher: true,
        ThreadActionsMenu: true,
      },
    },
  });

const conv = (id: string, title: string): Conversation =>
  ({ id, title, anchor_kind: 'card', anchor_id: 'k1', created_at: '', last_message_at: '' } as Conversation);

describe('ChatHeader', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    // The header loads the anchored thread list on mount; keep it off the network.
    vi.spyOn(useConversationsStore(), 'loadList').mockResolvedValue();
  });

  it('shows the active conversation title or a new-conversation label', () => {
    const store = useConversationsStore();
    store.activeID = null;
    expect(mountHeader().find('[data-test="chat-title"]').text()).toMatch(/new conversation/i);
    store.byID = { c1: conv('c1', 'My thread') };
    store.activeID = 'c1';
    expect(mountHeader().find('[data-test="chat-title"]').text()).toContain('My thread');
  });

  it('opens the THREAD switcher from the title', async () => {
    const store = useConversationsStore();
    store.list = [conv('a', 'T1'), conv('b', 'T2')]; // >1 so the trigger shows
    store.activeID = 'a';
    const w = mountHeader();
    expect(w.find('[data-test="thread-switch-pop"]').exists()).toBe(false);
    await w.find('[data-test="thread-switch-trigger"]').trigger('click');
    expect(w.find('[data-test="thread-switch-pop"]').exists()).toBe(true);
  });

  it('closes the thread switcher on an outside click', async () => {
    const store = useConversationsStore();
    store.list = [conv('a', 'T1'), conv('b', 'T2')];
    store.activeID = 'a';
    const w = mountHeader({ attachTo: document.body });
    await w.find('[data-test="thread-switch-trigger"]').trigger('click');
    expect(w.find('[data-test="thread-switch-pop"]').exists()).toBe(true);
    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await w.vm.$nextTick();
    expect(w.find('[data-test="thread-switch-pop"]').exists()).toBe(false);
    w.unmount();
  });

  it('emits close', async () => {
    const w = mountHeader();
    await w.find('[data-test="chat-close"]').trigger('click');
    expect(w.emitted('close')).toBeTruthy();
  });
});
