import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useConversationsStore } from './conversations';
import { usePresenceStore } from './presence';
import { useCardsStore } from './cards';
import { BackendError } from '../types/api';
import type { ChannelSession, Conversation, Message } from '../types/entity';

class MockWS {
  static instances: MockWS[] = [];
  url: string;
  readyState = 1; // OPEN
  listeners: Record<string, ((e: any) => void)[]> = {};
  constructor(url: string) {
    this.url = url;
    MockWS.instances.push(this);
  }
  addEventListener(kind: string, cb: (e: any) => void) {
    (this.listeners[kind] ??= []).push(cb);
  }
  removeEventListener() { /* noop */ }
  send(_data: string) { /* noop */ }
  close() { this.readyState = 3; (this.listeners.close ?? []).forEach((c) => c({})); }
  open() { (this.listeners.open ?? []).forEach((c) => c({})); }
  static last(): MockWS { return MockWS.instances[MockWS.instances.length - 1]; }
  simulate(payload: unknown) {
    (this.listeners.message ?? []).forEach((c) => c({ data: JSON.stringify(payload) }));
  }
}

beforeEach(() => {
  setActivePinia(createPinia());
  (global as any).WebSocket = MockWS;
  MockWS.instances = [];
  Object.defineProperty(window, 'location', {
    value: { protocol: 'http:', host: '127.0.0.1:5173' },
    writable: true,
  });
});

function fakeSession(id: string): ChannelSession {
  return {
    instance_id: 'i-' + id, session_id: id, cwd: '/repo',
    started_at: '', client_name: '', client_version: '', connected_at: '',
  };
}

// The store posts, so the request body is the last fetch call's init.body.
function lastRequestBody(): any {
  const fn = global.fetch as any;
  const init = fn.mock.calls.at(-1)![1];
  return JSON.parse(init.body as string);
}

const CONV = { id: '01CONV', title: 't', anchor_kind: null, anchor_id: null, created_at: '', last_message_at: '' } as Conversation;
const MSG = { id: '01MSG', conversation_id: '01CONV', role: 'user', content: 'hello', created_at: '' } as Message;

function mockSequence(responses: Array<{ status: number; body: unknown }>) {
  const fn: any = vi.fn();
  for (const r of responses) {
    fn.mockResolvedValueOnce({
      ok: r.status >= 200 && r.status < 300,
      status: r.status,
      json: async () => r.body,
      text: async () => JSON.stringify(r.body),
    });
  }
  global.fetch = fn;
}

describe('conversations store', () => {
  it('reports posted before any delivery frame', async () => {
    const presence = usePresenceStore();
    presence.sessions = [fakeSession('sess-A')];
    presence.select('sess-A');
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
      { status: 201, body: MSG },
    ]);

    const store = useConversationsStore();
    await store.setActive('01CONV');
    await store.optimisticPost('hello');

    expect(store.threadStatus?.state).toBe('posted');
  });

  it('escalates a section-anchored conversation to GRILLED on a user post', async () => {
    const CARD_CONV = { id: '01CONV', title: 't', anchor_kind: 'card', anchor_id: 'sect1', created_at: '', last_message_at: '' } as Conversation;
    mockSequence([
      { status: 200, body: CARD_CONV },
      { status: 200, body: { messages: [] } },
      { status: 201, body: MSG },
    ]);
    const cards = useCardsStore();
    cards.byID['sect1'] = { id: 'sect1', parent_card_id: 'doc1', review_grade: '' } as never;
    const esc = vi.spyOn(cards, 'escalateReviewGrade').mockResolvedValue();

    const store = useConversationsStore();
    await store.setActive('01CONV');
    await store.optimisticPost('a question');

    expect(esc).toHaveBeenCalledWith('sect1', 'GRILLED');
  });

  it('does not grade on a standalone (unanchored) post', async () => {
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
      { status: 201, body: MSG },
    ]);
    const cards = useCardsStore();
    const esc = vi.spyOn(cards, 'escalateReviewGrade').mockResolvedValue();

    const store = useConversationsStore();
    await store.setActive('01CONV');
    await store.optimisticPost('hello');

    expect(esc).not.toHaveBeenCalled();
  });

  it('carries session_id and session_cwd from a live event onto the message', async () => {
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');

    MockWS.last().simulate({
      conversation_id: '01CONV',
      message_id: '01ASST',
      role: 'assistant',
      content: 'reply',
      session_id: 'sess-A',
      session_cwd: '/home/x/zen',
      created_at: '',
    });

    const m = store.messagesByConv['01CONV'].find((x) => x.id === '01ASST');
    expect(m?.session_id).toBe('sess-A');
    expect(m?.session_cwd).toBe('/home/x/zen');
  });

  it('reports delivered once the delivery frame arrives', async () => {
    const presence = usePresenceStore();
    presence.sessions = [fakeSession('sess-A')];
    presence.select('sess-A');
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
      { status: 201, body: MSG },
    ]);

    const store = useConversationsStore();
    await store.setActive('01CONV');
    await store.optimisticPost('hello');
    presence.markDelivered('01MSG');

    expect(store.threadStatus?.state).toBe('delivered');
  });

  // The message reached nobody. Two seconds is long enough that a healthy
  // channel has always acked, and short enough that the user is still looking.
  it('reports undelivered after 2s with no ack', async () => {
    vi.useFakeTimers();
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
      { status: 201, body: MSG },
    ]);

    const store = useConversationsStore();
    await store.setActive('01CONV');
    await store.optimisticPost('hello');
    expect(store.threadStatus?.state).toBe('posted');

    await vi.advanceTimersByTimeAsync(2000);
    expect(store.threadStatus?.state).toBe('undelivered');
    vi.useRealTimers();
  });

  // A late ack still wins: delivered is the truth, the timer was only a guess.
  it('a delivery frame after the timer still reports delivered', async () => {
    vi.useFakeTimers();
    const presence = usePresenceStore();
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
      { status: 201, body: MSG },
    ]);

    const store = useConversationsStore();
    await store.setActive('01CONV');
    await store.optimisticPost('hello');
    await vi.advanceTimersByTimeAsync(2000);
    expect(store.threadStatus?.state).toBe('undelivered');

    presence.markDelivered('01MSG');
    expect(store.threadStatus?.state).toBe('delivered');
    vi.useRealTimers();
  });

  // Once the assistant answers there is nothing to report.
  it('reports nothing once the assistant has replied', async () => {
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
      { status: 201, body: MSG },
    ]);

    const store = useConversationsStore();
    await store.setActive('01CONV');
    await store.optimisticPost('hello');

    MockWS.instances[0].simulate({
      conversation_id: '01CONV', message_id: '01REPLY',
      role: 'assistant', content: 'hi back', created_at: 'now',
    });

    expect(store.threadStatus).toBeNull();
  });

  // The bus dropped the assistant reply and nothing followed it, so no event can
  // ever reveal the gap. Regaining focus reconciles against the database.
  it('reconciles on visibilitychange when the tab becomes visible', async () => {
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [{ ...MSG, id: '01A' }] } },
      { status: 200, body: { messages: [{ ...MSG, id: '01B', role: 'assistant', content: 'the reply you never saw' }] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');

    Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    await vi.waitFor(() => {
      expect(store.messagesByConv['01CONV'].map((m) => m.id)).toEqual(['01A', '01B']);
    });
    const url = (global.fetch as any).mock.calls.at(-1)![0] as string;
    expect(url).toContain('after=01A');
  });

  it('does not reconcile when the tab is hidden', async () => {
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [{ ...MSG, id: '01A' }] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');

    const before = (global.fetch as any).mock.calls.length;
    Object.defineProperty(document, 'visibilityState', { value: 'hidden', configurable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    expect((global.fetch as any).mock.calls.length).toBe(before);
  });

  it('does not reconcile when no conversation is active', async () => {
    const store = useConversationsStore();
    global.fetch = vi.fn() as any;

    Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    expect((global.fetch as any).mock.calls.length).toBe(0);
    void store;
  });

  it('reconnects the conversation socket after it closes', async () => {
    vi.useFakeTimers();
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');
    expect(MockWS.instances.length).toBe(1);

    MockWS.last().close();
    await vi.advanceTimersByTimeAsync(1000);
    expect(MockWS.instances.length).toBe(2);
    vi.useRealTimers();
  });

  it('backs off exponentially across repeated failures', async () => {
    vi.useFakeTimers();
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');

    MockWS.last().close();
    await vi.advanceTimersByTimeAsync(1000);
    expect(MockWS.instances.length).toBe(2);

    MockWS.last().close();
    await vi.advanceTimersByTimeAsync(1000);
    expect(MockWS.instances.length).toBe(2); // still waiting: 2s, not 1s
    await vi.advanceTimersByTimeAsync(1000);
    expect(MockWS.instances.length).toBe(3);
    vi.useRealTimers();
  });

  // Reconnecting is gap detection with an empty socket: whatever arrived while
  // we were away is fetched from the database.
  it('catches up when the socket reopens', async () => {
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [{ ...MSG, id: '01A' }] } },
      { status: 200, body: { messages: [{ ...MSG, id: '01B', content: 'missed' }] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');

    MockWS.last().open();

    await vi.waitFor(() => {
      expect(store.messagesByConv['01CONV'].map((m) => m.id)).toEqual(['01A', '01B']);
    });
    const url = (global.fetch as any).mock.calls.at(-1)![0] as string;
    expect(url).toContain('after=01A');
  });

  // setActive(null) is a deliberate close; it must not reconnect.
  it('does not reconnect after setActive(null)', async () => {
    vi.useFakeTimers();
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');
    await store.setActive(null);

    await vi.advanceTimersByTimeAsync(5000);
    expect(MockWS.instances.length).toBe(1);
    vi.useRealTimers();
  });

  it('applies an event whose prev matches the cursor, without refetching', async () => {
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [{ ...MSG, id: '01A' }] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');

    const before = (global.fetch as any).mock.calls.length;
    MockWS.instances[0].simulate({
      conversation_id: '01CONV', message_id: '01B', prev_message_id: '01A',
      role: 'assistant', content: 'two', created_at: '',
    });

    expect(store.messagesByConv['01CONV'].map((m) => m.id)).toEqual(['01A', '01B']);
    expect((global.fetch as any).mock.calls.length).toBe(before);
  });

  // The bus dropped 01B. The event for 01C names it, the cursor says 01A, so the
  // store refetches the span and ends up with all three, in order.
  it('refetches the span when prev does not match the cursor', async () => {
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [{ ...MSG, id: '01A' }] } },
      { status: 200, body: { messages: [
        { ...MSG, id: '01B', content: 'dropped' },
        { ...MSG, id: '01C', content: 'three' },
      ] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');

    MockWS.instances[0].simulate({
      conversation_id: '01CONV', message_id: '01C', prev_message_id: '01B',
      role: 'assistant', content: 'three', created_at: '',
    });
    await vi.waitFor(() => {
      expect(store.messagesByConv['01CONV'].map((m) => m.id)).toEqual(['01A', '01B', '01C']);
    });

    const url = (global.fetch as any).mock.calls.at(-1)![0] as string;
    expect(url).toContain('after=01A');
  });

  // An empty prev means "no gap possible" — the first message of a thread, and
  // every dispatched re-publish.
  it('never refetches on an empty prev', async () => {
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [{ ...MSG, id: '01A' }] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');

    const before = (global.fetch as any).mock.calls.length;
    MockWS.instances[0].simulate({
      conversation_id: '01CONV', message_id: '01Z', prev_message_id: '',
      role: 'user', content: 'redispatched', created_at: '',
    });

    expect((global.fetch as any).mock.calls.length).toBe(before);
  });

  // A message we already hold is not a gap, whatever its prev claims.
  it('does not refetch for a message it already has', async () => {
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [{ ...MSG, id: '01A' }] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');

    const before = (global.fetch as any).mock.calls.length;
    MockWS.instances[0].simulate({
      conversation_id: '01CONV', message_id: '01A', prev_message_id: '01ZZZ',
      role: 'user', content: 'hello', created_at: '',
    });

    expect((global.fetch as any).mock.calls.length).toBe(before);
  });

  it('sends the selected session as the target', async () => {
    const presence = usePresenceStore();
    presence.sessions = [fakeSession('sess-A')];
    presence.select('sess-A');

    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
      { status: 201, body: MSG },
    ]);

    const store = useConversationsStore();
    await store.setActive('01CONV');
    await store.optimisticPost('hello');

    expect(lastRequestBody()).toMatchObject({
      role: 'user',
      content: 'hello',
      target_session_id: 'sess-A',
    });
  });

  it('omits the target when no session is selected', async () => {
    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
      { status: 201, body: MSG },
    ]);

    const store = useConversationsStore();
    await store.setActive('01CONV');
    await store.optimisticPost('hello');

    expect(lastRequestBody().target_session_id).toBeUndefined();
  });

  // A 409 means the chosen session died between the picker and the post.
  // Clear it so the picker re-prompts rather than sending into the void again.
  it('clears the selection on a dead-target 409', async () => {
    const presence = usePresenceStore();
    presence.sessions = [fakeSession('sess-A')];
    presence.select('sess-A');

    mockSequence([
      { status: 200, body: CONV },
      { status: 200, body: { messages: [] } },
      { status: 409, body: { kind: 'Conflict', message: 'session sess-A is not connected' } },
    ]);

    const store = useConversationsStore();
    await store.setActive('01CONV');
    await expect(store.optimisticPost('hello')).rejects.toBeInstanceOf(BackendError);

    expect(presence.selectedSessionID).toBeNull();
  });

  it('setActive opens a WS to the conversation', async () => {
    mockSequence([
      { status: 200, body: { id: '01CONV', title: 't', anchor_kind: null, anchor_id: null, created_at: '', last_message_at: '' } as Conversation },
      { status: 200, body: { messages: [] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');
    expect(MockWS.instances.length).toBe(1);
    expect(MockWS.instances[0].url).toContain('/api/v1/conversations/01CONV/ws');
  });

  it('applies WS events into messagesByConv', async () => {
    mockSequence([
      { status: 200, body: { id: '01CONV', title: 't', anchor_kind: null, anchor_id: null, created_at: '', last_message_at: '' } as Conversation },
      { status: 200, body: { messages: [] } },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');

    MockWS.instances[0].simulate({
      conversation_id: '01CONV', message_id: '01MSG',
      role: 'assistant', content: 'hi back', created_at: 'now',
    });

    expect(store.messagesByConv['01CONV']?.length).toBe(1);
    expect(store.messagesByConv['01CONV'][0].content).toBe('hi back');
  });

  it('optimisticPost de-duplicates against WS echo', async () => {
    mockSequence([
      { status: 200, body: { id: '01CONV', title: 't', anchor_kind: null, anchor_id: null, created_at: '', last_message_at: '' } as Conversation },
      { status: 200, body: { messages: [] } },
      { status: 201, body: { id: '01MSG', conversation_id: '01CONV', role: 'user', content: 'hi', selection_text: null, created_at: 'now' } as Message },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');
    await store.optimisticPost('hi');
    expect(store.messagesByConv['01CONV'].length).toBe(1);

    MockWS.instances[0].simulate({
      conversation_id: '01CONV', message_id: '01MSG',
      role: 'user', content: 'hi', created_at: 'now',
    });
    expect(store.messagesByConv['01CONV'].length).toBe(1);
  });

  it('pendingAssistant is true after a user message and false after assistant arrives', async () => {
    mockSequence([
      { status: 200, body: { id: '01CONV', title: 't', anchor_kind: null, anchor_id: null, created_at: '', last_message_at: '' } as Conversation },
      { status: 200, body: { messages: [] } },
      // Fake ids must sort like real ULIDs: monotonic, so older < newer.
      // The store orders the thread by id, as the database does.
      { status: 201, body: { id: '01AAAUSER', conversation_id: '01CONV', role: 'user', content: 'q', selection_text: null, created_at: 'now' } as Message },
    ]);
    const store = useConversationsStore();
    await store.setActive('01CONV');
    await store.optimisticPost('q');
    expect(store.pendingAssistant).toBe(true);

    MockWS.instances[0].simulate({
      conversation_id: '01CONV', message_id: '01BBBASS',
      role: 'assistant', content: 'a', created_at: 'now',
    });
    expect(store.pendingAssistant).toBe(false);
  });
});

describe('rename', () => {
  it('optimistically updates the title and PUTs it', async () => {
    const store = useConversationsStore();
    store.byID = { c1: { id: 'c1', title: 'old', anchor_kind: 'card', anchor_id: 'k1', created_at: '', last_message_at: '' } as Conversation };
    mockSequence([{ status: 200, body: { id: 'c1', title: 'new', anchor_kind: 'card', anchor_id: 'k1', created_at: '', last_message_at: '' } }]);
    await store.rename('c1', 'new');
    expect(store.byID['c1'].title).toBe('new');
    expect(lastRequestBody()).toEqual({ title: 'new' });
  });

  it('rolls the title back if the PUT fails', async () => {
    const store = useConversationsStore();
    store.byID = { c1: { id: 'c1', title: 'old', anchor_kind: 'card', anchor_id: 'k1', created_at: '', last_message_at: '' } as Conversation };
    mockSequence([{ status: 500, body: { message: 'boom' } }]);
    await expect(store.rename('c1', 'new')).rejects.toBeTruthy();
    expect(store.byID['c1'].title).toBe('old');
  });
});

describe('deliveryState', () => {
  it('is sent for a posted message with no ack', () => {
    const store = useConversationsStore();
    expect(store.deliveryState('m1')).toBe('sent');
  });
  it('is delivered once the presence store acks it', () => {
    const store = useConversationsStore();
    usePresenceStore().markDelivered('m1');
    expect(store.deliveryState('m1')).toBe('delivered');
  });
  it('is undelivered when the timer fired and there is no ack', () => {
    const store = useConversationsStore();
    store.undelivered = { m1: true };
    expect(store.deliveryState('m1')).toBe('undelivered');
  });
  it('delivered beats an earlier undelivered flag (late ack)', () => {
    const store = useConversationsStore();
    store.undelivered = { m1: true };
    usePresenceStore().markDelivered('m1');
    expect(store.deliveryState('m1')).toBe('delivered');
  });
});

describe('linked conversations (card view)', () => {
  const conv = (over: Partial<Conversation>): Conversation => ({
    id: 'x', title: 't', anchor_kind: 'card', anchor_id: 'a',
    created_at: '', last_message_at: '', ...over,
  });

  it('linkedFor merges origin + discussions, origin first, discussions newest first', () => {
    const store = useConversationsStore();
    store.byID = { o1: conv({ id: 'o1', title: 'origin', anchor_kind: null, anchor_id: null }) };
    store.byAnchor = {
      'card:a': [
        conv({ id: 'd1', last_message_at: '2026-07-01T00:00:00Z' }),
        conv({ id: 'd2', last_message_at: '2026-07-10T00:00:00Z' }),
      ],
    };
    const { items } = store.linkedFor('a', 'o1');
    expect(items.map((i) => [i.kind, i.conversation.id])).toEqual([
      ['origin', 'o1'], ['discussion', 'd2'], ['discussion', 'd1'],
    ]);
  });

  it('linkedFor dedups an origin that is also anchored, keeping it as origin', () => {
    const store = useConversationsStore();
    store.byID = { o1: conv({ id: 'o1' }) };
    store.byAnchor = { 'card:a': [conv({ id: 'o1' }), conv({ id: 'd1' })] };
    const { items } = store.linkedFor('a', 'o1');
    expect(items.filter((i) => i.conversation.id === 'o1')).toHaveLength(1);
    expect(items[0].kind).toBe('origin');
  });

  it('linkedFor omits a missing origin (not in byID)', () => {
    const store = useConversationsStore();
    store.byAnchor = { 'card:a': [conv({ id: 'd1' })] };
    const { items } = store.linkedFor('a', 'gone');
    expect(items.map((i) => i.conversation.id)).toEqual(['d1']);
  });

  it('loadForAnchor caches anchored conversations by anchor and by id', async () => {
    mockSequence([{ status: 200, body: { conversations: [conv({ id: 'd1' }), conv({ id: 'd2' })] } }]);
    const store = useConversationsStore();
    await store.loadForAnchor('card', 'a');
    expect(store.byAnchor['card:a'].map((c) => c.id)).toEqual(['d1', 'd2']);
    expect(store.byID['d1'].id).toBe('d1');
  });

  it('ensureConversation swallows a missing origin', async () => {
    mockSequence([{ status: 404, body: { message: 'gone' } }]);
    const store = useConversationsStore();
    await store.ensureConversation('gone');
    expect(store.byID['gone']).toBeUndefined();
  });
});
