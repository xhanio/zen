import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ChatThread from './ChatThread.vue';
import { useConversationsStore } from '../../stores/conversations';
import { usePresenceStore } from '../../stores/presence';
import type { ChannelSession, Message } from '../../types/entity';

const userMsg = {
  id: '01MSG', conversation_id: '01CONV', role: 'user' as const,
  content: 'hello', selection_text: null, created_at: '',
};

function m(over: Partial<Message>): Message {
  return {
    id: 'x', conversation_id: '01CONV', role: 'user', content: 'c',
    selection_text: null, created_at: '', ...over,
  } as Message;
}

function seed() {
  const store = useConversationsStore();
  store.messagesByConv = { '01CONV': [userMsg] };
  store.activeID = '01CONV';
  return store;
}

function pickSession(): ChannelSession {
  const s: ChannelSession = {
    instance_id: 'i1', session_id: 's1', cwd: '/repo',
    started_at: '', client_name: '', client_version: '', connected_at: '',
  };
  const presence = usePresenceStore();
  presence.sessions = [s];
  presence.select('s1');
  return s;
}

beforeEach(() => {
  setActivePinia(createPinia());
  localStorage.clear();
  // resend POSTs to the dispatch endpoint; 204 No Content.
  global.fetch = vi.fn(async () => ({
    ok: true, status: 204,
    json: async () => ({}),
    text: async () => '',
  })) as any;
});

describe('ChatThread transcript', () => {
  it('renders a turn per message with the right speaker', () => {
    seed();
    const w = mount(ChatThread, { props: { conversationId: '01CONV' },
      global: { stubs: { MarkdownBody: true } } });
    expect(w.findAll('[data-test="turn"]').length).toBe(1);
    expect(w.text()).toContain('You');
  });

  it('passes the message delivery state through to the turn', () => {
    seed(); pickSession();
    usePresenceStore().markDelivered('01MSG');
    const w = mount(ChatThread, { props: { conversationId: '01CONV' },
      global: { stubs: { MarkdownBody: true } } });
    expect(w.find('[data-test="turn-state"]').text()).toMatch(/Claude Code has it/i);
  });

  it('resending a turn calls store.resend with the message id', async () => {
    const store = seed(); pickSession();
    store.undelivered = { '01MSG': true };
    const spy = vi.spyOn(store, 'resend').mockResolvedValue();
    const w = mount(ChatThread, { props: { conversationId: '01CONV' },
      global: { stubs: { MarkdownBody: true } } });
    await w.find('[data-test="turn-resend"]').trigger('click');
    expect(spy).toHaveBeenCalledWith('01MSG');
  });

  it('draws a divider when the session changes between turns', () => {
    const store = useConversationsStore();
    store.messagesByConv = { '01CONV': [
      m({ id: '01A', session_id: 'sess-A', session_cwd: '/x/alpha' }),
      m({ id: '01B', role: 'assistant', session_id: 'sess-A', session_cwd: '/x/alpha' }),
      m({ id: '01C', session_id: 'sess-B', session_cwd: '/y/beta' }),
    ] };
    store.activeID = '01CONV';
    const w = mount(ChatThread, { props: { conversationId: '01CONV' },
      global: { stubs: { MarkdownBody: true } } });
    expect(w.findAll('[data-test="session-divider"]').length).toBe(1);
  });

  it('appends a short-id suffix when two sessions share a cwd basename', () => {
    const store = useConversationsStore();
    store.messagesByConv = { '01CONV': [
      m({ id: '01A', session_id: 'aaaa1111', session_cwd: '/x/zen' }),
      m({ id: '01B', session_id: 'bbbb2222', session_cwd: '/y/zen' }),
    ] };
    store.activeID = '01CONV';
    const w = mount(ChatThread, { props: { conversationId: '01CONV' },
      global: { stubs: { MarkdownBody: true } } });
    const tags = w.findAll('[data-test="turn-session"]').map((n) => n.text());
    expect(tags[0]).toContain('#1111');
    expect(tags[1]).toContain('#2222');
  });

  it('labels an assistant turn by its own session, not the selected one', () => {
    pickSession(); // selects s1, cwd /repo
    const store = useConversationsStore();
    store.messagesByConv = { '01CONV': [
      m({ id: '01B', role: 'assistant', session_id: 'sess-Z', session_cwd: '/home/x/otherproj' }),
    ] };
    store.activeID = '01CONV';
    const w = mount(ChatThread, { props: { conversationId: '01CONV' },
      global: { stubs: { MarkdownBody: true } } });
    expect(w.text()).toContain('otherproj');
    expect(w.text()).not.toContain('repo');
  });
});
