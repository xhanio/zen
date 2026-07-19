import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useChatSidebar } from './useChatSidebar';
import { useConversationsStore } from '../stores/conversations';

function mockSequence(responses: Array<{ status: number; body: unknown }>) {
  const fn: any = vi.fn();
  for (const r of responses) {
    fn.mockResolvedValueOnce({
      ok: r.status >= 200 && r.status < 300,
      status: r.status, json: async () => r.body, text: async () => JSON.stringify(r.body),
    });
  }
  global.fetch = fn;
}

beforeEach(() => {
  setActivePinia(createPinia());
  (global as any).WebSocket = class {
    constructor(_url: string) {}
    addEventListener() {} removeEventListener() {}
    close() {} send() {}
  };
  Object.defineProperty(window, 'location', {
    value: { protocol: 'http:', host: '127.0.0.1:5173' },
    writable: true,
  });
  // Reset singleton state between tests (the composable uses module-level refs).
  const s = useChatSidebar();
  s.close();
  s.anchorKind.value = null;
  s.anchorID.value = null;
});

describe('useChatSidebar', () => {
  it('openFor(null, null) sets open + clears anchor', async () => {
    mockSequence([
      // openFor(null, null) doesn't hit /conversations because kind+id are null
      // — but setActive(null) is still called which doesn't fetch either.
    ]);
    const s = useChatSidebar();
    await s.openFor(null, null);
    expect(s.open.value).toBe(true);
    expect(s.anchorKind.value).toBeNull();
    expect(s.pendingSelection.value).toBeNull();
  });

  it('openFor(anchor) activates the first existing conversation on that anchor', async () => {
    mockSequence([
      { status: 200, body: { conversations: [{ id: '01PRIOR', title: 'prior',
            anchor_kind: 'card', anchor_id: '01CARD',
            created_at: '', last_message_at: '' }] } },
      { status: 200, body: { id: '01PRIOR', title: 'prior',
            anchor_kind: 'card', anchor_id: '01CARD',
            created_at: '', last_message_at: '' } },
      { status: 200, body: { messages: [] } },
    ]);
    const s = useChatSidebar();
    await s.openFor('card', '01CARD');
    expect(s.anchorKind.value).toBe('card');
    expect(s.anchorID.value).toBe('01CARD');
  });

  it('openFor activates the anchored conversation even when the shared list is clobbered (race regression)', async () => {
    const store = useConversationsStore();
    // Simulate the ChatHeader's concurrent loadList tripping the sequence guard
    // so openFor's own list load is discarded (store.list stays empty). The old
    // openFor did loadList + list.find and would land on a blank
    // "New conversation"; the fix fetches the anchor directly.
    vi.spyOn(store, 'loadList').mockResolvedValue();
    const conv = {
      id: '01PRIOR', title: 'prior', anchor_kind: 'card', anchor_id: '01CARD',
      created_at: '', last_message_at: '',
    };
    mockSequence([
      { status: 200, body: { conversations: [conv] } }, // mostRecentForAnchor
      { status: 200, body: conv }, // setActive → loadConversation (get)
      { status: 200, body: { messages: [] } }, // loadConversation (messages)
    ]);
    const s = useChatSidebar();
    await s.openFor('card', '01CARD');
    expect(store.activeID).toBe('01PRIOR');
    expect(store.list.length).toBe(0); // openFor did not rely on the shared list
  });

  it('close() hides without clearing anchor', () => {
    const s = useChatSidebar();
    s.anchorKind.value = 'card';
    s.anchorID.value = '01CARD';
    s.open.value = true;
    s.close();
    expect(s.open.value).toBe(false);
    expect(s.anchorKind.value).toBe('card');
  });

  it('pendingSelection carries through openFor', async () => {
    const s = useChatSidebar();
    await s.openFor(null, null, 'selected text');
    expect(s.pendingSelection.value).toBe('selected text');
    s.clearSelection();
    expect(s.pendingSelection.value).toBeNull();
  });
});
