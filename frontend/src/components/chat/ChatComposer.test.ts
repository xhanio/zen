import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ChatComposer from './ChatComposer.vue';
import { useConversationsStore } from '../../stores/conversations';
import { usePresenceStore } from '../../stores/presence';
import type { ChannelSession } from '../../types/entity';

// Mirrors the real singleton's shape: the composer reads pendingRange and
// pendingSeq alongside pendingSelection, and a partial double would throw
// inside the send handler rather than fail an assertion.
const sidebarState = {
  anchorKind: { value: 'card' as string | null },
  anchorID: { value: 'k1' as string | null },
  pendingSelection: { value: null as string | null },
  pendingRange: { value: null as { start: number; end: number } | null },
  pendingSeq: { value: null as number | null },
  clearSelection: vi.fn(),
};
vi.mock('../../composables/useChatSidebar', () => ({
  useChatSidebar: () => sidebarState,
}));

function pick() {
  const p = usePresenceStore();
  p.sessions = [{ instance_id: 'i', session_id: 's1', cwd: '/home/x/repo',
    started_at: '', client_name: '', client_version: '', connected_at: '' } as ChannelSession];
  p.select('s1');
}

// optimisticPost now returns the created Message so the composer can paint the
// selection without refetching; the mock has to honour that contract.
const sentMsg = {
  id: 'm1', conversation_id: 'c1', role: 'user', content: 'hello',
  selection_text: null, selection_start: null, selection_end: null,
  selection_seq: null, created_at: '2026-07-27T00:00:00Z',
} as never;

describe('ChatComposer', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it('Enter sends, Shift+Enter does not', async () => {
    const store = useConversationsStore(); store.activeID = 'c1'; pick();
    const spy = vi.spyOn(store, 'optimisticPost').mockResolvedValue(sentMsg);
    const w = mount(ChatComposer);
    const ta = w.find('[data-test="composer-input"]');
    await ta.setValue('hello');
    await ta.trigger('keydown', { key: 'Enter', shiftKey: true });
    expect(spy).not.toHaveBeenCalled();
    await ta.trigger('keydown', { key: 'Enter' });
    expect(spy).toHaveBeenCalledWith('hello', null, null, null);
  });

  // The captured range must reach the post: it is the whole point of the
  // capture chain, and nothing downstream can reconstruct it.
  it('forwards the pending selection range to the post', async () => {
    const store = useConversationsStore(); store.activeID = 'c1'; pick();
    const spy = vi.spyOn(store, 'optimisticPost').mockResolvedValue(sentMsg);
    sidebarState.pendingSelection.value = 'quick';
    sidebarState.pendingRange.value = { start: 4, end: 9 };
    sidebarState.pendingSeq.value = 2;

    const w = mount(ChatComposer);
    const ta = w.find('[data-test="composer-input"]');
    await ta.setValue('tighten this');
    await ta.trigger('keydown', { key: 'Enter' });

    expect(spy).toHaveBeenCalledWith('tighten this', 'quick', { start: 4, end: 9 }, 2);

    sidebarState.pendingSelection.value = null;
    sidebarState.pendingRange.value = null;
    sidebarState.pendingSeq.value = null;
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
