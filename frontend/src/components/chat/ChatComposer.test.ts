import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ChatComposer from './ChatComposer.vue';
import { useConversationsStore } from '../../stores/conversations';
import { usePresenceStore } from '../../stores/presence';
import type { ChannelSession } from '../../types/entity';

vi.mock('../../composables/useChatSidebar', () => ({
  useChatSidebar: () => ({
    anchorKind: { value: 'card' }, anchorID: { value: 'k1' },
    pendingSelection: { value: null }, clearSelection: vi.fn(),
  }),
}));

function pick() {
  const p = usePresenceStore();
  p.sessions = [{ instance_id: 'i', session_id: 's1', cwd: '/home/x/repo',
    started_at: '', client_name: '', client_version: '', connected_at: '' } as ChannelSession];
  p.select('s1');
}

describe('ChatComposer', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it('Enter sends, Shift+Enter does not', async () => {
    const store = useConversationsStore(); store.activeID = 'c1'; pick();
    const spy = vi.spyOn(store, 'optimisticPost').mockResolvedValue();
    const w = mount(ChatComposer);
    const ta = w.find('[data-test="composer-input"]');
    await ta.setValue('hello');
    await ta.trigger('keydown', { key: 'Enter', shiftKey: true });
    expect(spy).not.toHaveBeenCalled();
    await ta.trigger('keydown', { key: 'Enter' });
    expect(spy).toHaveBeenCalledWith('hello', null);
  });

  it('renders the session picker left of send and toggles the switcher upward', async () => {
    const store = useConversationsStore(); store.activeID = 'c1'; pick();
    const w = mount(ChatComposer, { global: { stubs: { SessionSwitcher: true } } });
    // The picker trigger lives in the composer now, not the header.
    expect(w.find('[data-test="presence-pill"]').exists()).toBe(true);
    expect(w.find('[data-test="composer-session-pop"]').exists()).toBe(false);
    await w.find('[data-test="presence-pill"]').trigger('click');
    expect(w.find('[data-test="composer-session-pop"]').exists()).toBe(true);
  });
});
