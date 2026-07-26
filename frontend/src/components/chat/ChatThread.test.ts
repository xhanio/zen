import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ChatThread from './ChatThread.vue';
import { nextTick } from 'vue';
import { useConversationsStore } from '../../stores/conversations';
import { useSnapshotsStore } from '../../stores/snapshots';
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

describe('ChatThread snapshot records', () => {
  it('renders a record between the messages it sits between, in time order', async () => {
    const store = useConversationsStore();
    store.messagesByConv = {
      '01CONV': [
        m({ id: '01A', created_at: '2026-07-25T10:00:00Z' }),
        m({ id: '01B', role: 'assistant', created_at: '2026-07-25T10:05:00Z' }),
      ],
    };
    store.activeID = '01CONV';
    const snapshots = useSnapshotsStore();
    snapshots.byConversation = {
      '01CONV': [{
        id: '01SNAP', card_id: '01CARD', card_title: 'Target card', seq: 3,
        title: 't', summary: '', content: '', format: 'markdown',
        actor: 'agent', conversation_id: '01CONV', change_kind: 'update',
        diff: '', diff_truncated: false, lines_added: 4, lines_removed: 2,
        created_at: '2026-07-25T10:02:00Z',
      }],
    };

    const w = mount(ChatThread, {
      props: { conversationId: '01CONV' },
      global: { stubs: { MarkdownBody: true, RouterLink: true } },
    });
    await nextTick();

    const record = w.find('[data-test="snapshot-record"]');
    expect(record.exists()).toBe(true);
    expect(record.text()).toContain('Target card');
    expect(record.text()).toContain('#2→#3');
    // A record must never be mistaken for something Claude said.
    expect(w.findAll('[data-test="turn"]')).toHaveLength(2);
  });
});

describe('ChatThread empty state', () => {
  it('says nothing-yet only when the timeline is truly empty', async () => {
    const store = useConversationsStore();
    store.messagesByConv = { '01CONV': [] };
    store.activeID = '01CONV';

    const w = mount(ChatThread, {
      props: { conversationId: '01CONV' },
      global: { stubs: { MarkdownBody: true } },
    });
    await nextTick();
    expect(w.text()).toContain('No messages yet.');
  });

  // Reachable for real: an agent edit can cite a conversation that has no
  // messages. Claiming emptiness above a visible record contradicts itself.
  it('does not claim emptiness when records exist without messages', async () => {
    const store = useConversationsStore();
    store.messagesByConv = { '01CONV': [] };
    store.activeID = '01CONV';
    const snapshots = useSnapshotsStore();
    snapshots.byConversation = {
      '01CONV': [{
        id: '01SNAP', card_id: '01CARD', card_title: 'Edited card', seq: 2,
        title: 't', summary: '', content: '', format: 'markdown',
        actor: 'agent', conversation_id: '01CONV', change_kind: 'update',
        diff: '', diff_truncated: false, lines_added: 1, lines_removed: 0,
        created_at: '2026-07-25T10:00:00Z',
      }],
    };

    const w = mount(ChatThread, {
      props: { conversationId: '01CONV' },
      global: { stubs: { MarkdownBody: true } },
    });
    await nextTick();

    expect(w.find('[data-test="snapshot-record"]').exists()).toBe(true);
    expect(w.text()).not.toContain('No messages yet.');
  });
});
