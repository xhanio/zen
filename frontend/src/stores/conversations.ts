import { ref, computed } from 'vue';
import { defineStore } from 'pinia';
import {
  listConversations as apiList,
  getConversation as apiGet,
  createConversation as apiCreate,
  updateConversationTitle as apiRename,
  deleteConversation as apiDelete,
  listMessages as apiListMessages,
  appendMessage as apiAppendMessage,
  dispatchMessage as apiDispatchMessage,
  getCard as apiGetCard,
  getGroup as apiGetGroup,
} from '../api/client';
import { BackendError, type AppendMessageRequest, type CreateConversationRequest } from '../types/api';
import { usePresenceStore } from './presence';
import { useCardsStore } from './cards';
import type { Conversation, ConversationEvent, Message } from '../types/entity';

// Two seconds is long enough that a healthy channel has always acked, and short
// enough that the user is still looking at the thread.
const UNDELIVERED_AFTER_MS = 2000;

// The same ladder the presence store and the channel's subscriber already use.
const WS_DIAL_DELAY_MS = 1000;
const WS_MAX_DELAY_MS = 30000;

// The visibilitychange listener is bound to `document`, which outlives any one
// store instance. Production has exactly one store, but a new Pinia (every test,
// and a hot reload) builds another — so keep a module-level handle and replace
// the old listener instead of stacking a second one onto a dead store.
let boundVisibilityHandler: (() => void) | null = null;

export type ThreadState = 'posted' | 'delivered' | 'undelivered';

export const useConversationsStore = defineStore('conversations', () => {
  const list = ref<Conversation[]>([]);
  const listLoading = ref(false);
  const listError = ref<BackendError | null>(null);

  const byID = ref<Record<string, Conversation>>({});
  const messagesByConv = ref<Record<string, Message[]>>({});

  const activeID = ref<string | null>(null);

  // Delivery lives in the presence store, which owns the control stream. Grab
  // the handle once: presence does not import conversations, so there is no
  // cycle.
  const presence = usePresenceStore();

  // Message ids we posted and have not seen delivered. Purely local: the
  // backend has no opinion about this, and a refresh forgets it.
  const undelivered = ref<Record<string, true>>({});
  const undeliveredTimers = new Map<string, ReturnType<typeof setTimeout>>();

  function armUndeliveredTimer(messageID: string) {
    const existing = undeliveredTimers.get(messageID);
    if (existing) clearTimeout(existing);
    const timer = setTimeout(() => {
      undeliveredTimers.delete(messageID);
      undelivered.value = { ...undelivered.value, [messageID]: true };
    }, UNDELIVERED_AFTER_MS);
    undeliveredTimers.set(messageID, timer);
  }

  const listSeq = { current: 0 };
  const convSeq: Record<string, number> = {};

  // Optimistically-posted message IDs (server-assigned). The composer pushes
  // the user message into the thread immediately after the REST POST returns;
  // the matching WS echo (~ms later) is suppressed by this set.
  const pendingIDs = new Set<string>();

  let ws: WebSocket | null = null;
  let wsRetryTimer: ReturnType<typeof setTimeout> | null = null;
  let wsBackoff = WS_DIAL_DELAY_MS;
  let wsStopped = false;

  function wsURL(id: string): string {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${proto}//${window.location.host}/api/v1/conversations/${encodeURIComponent(id)}/ws`;
  }

  // The newest message id the store holds for a conversation. The database is
  // the source of truth; the socket is only a wake-up hint.
  function cursorOf(conversationID: string): string {
    const arr = messagesByConv.value[conversationID] ?? [];
    return arr.length ? arr[arr.length - 1].id : '';
  }

  function mergeMessage(m: Message) {
    const arr = messagesByConv.value[m.conversation_id] ?? [];
    if (arr.some((x) => x.id === m.id)) return;
    arr.push(m);
    arr.sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0)); // ULIDs sort by time
    messagesByConv.value[m.conversation_id] = [...arr];
  }

  // Fetch everything after our cursor and merge it in. This is the one recovery
  // path: reconnect, laptop sleep, backend restart, and a bus drop all reduce to
  // "my cursor is behind the server".
  async function catchUp(conversationID: string): Promise<void> {
    try {
      const resp = await apiListMessages(conversationID, { after: cursorOf(conversationID) });
      for (const m of resp.messages) mergeMessage(m);
    } catch { /* a failed catch-up retries on the next event or focus */ }
  }

  // Gap detection needs a *next* event to see a gap, so a drop at the tail of
  // the stream is invisible to it — a dropped assistant reply with nothing after
  // it would leave the thread spinning until a refresh. Regaining focus is the
  // same "my cursor is behind" story with a different trigger, and costs no
  // polling.
  function onVisibilityChange() {
    if (typeof document === 'undefined') return;
    if (document.visibilityState !== 'visible') return;
    const id = activeID.value;
    if (!id) return;
    void catchUp(id);
  }

  if (typeof document !== 'undefined') {
    if (boundVisibilityHandler) {
      document.removeEventListener('visibilitychange', boundVisibilityHandler);
    }
    boundVisibilityHandler = onVisibilityChange;
    document.addEventListener('visibilitychange', boundVisibilityHandler);
  }

  function applyEvent(ev: ConversationEvent) {
    const arr = messagesByConv.value[ev.conversation_id] ?? [];

    // Already held: not a gap, whatever prev claims. Also covers the WS echo of
    // our own optimistic post.
    if (pendingIDs.has(ev.message_id)) {
      pendingIDs.delete(ev.message_id);
      return;
    }
    if (arr.some((m) => m.id === ev.message_id)) return;

    // An empty prev means no gap is possible. Otherwise a prev that is not our
    // cursor means the bus dropped something in between.
    const cursor = cursorOf(ev.conversation_id);
    if (ev.prev_message_id && cursor && ev.prev_message_id !== cursor) {
      void catchUp(ev.conversation_id);
      return; // catchUp brings this message too
    }

    mergeMessage({
      id: ev.message_id,
      conversation_id: ev.conversation_id,
      role: ev.role,
      content: ev.content,
      selection_text: ev.selection_text ?? null,
      session_id: ev.session_id ?? null,
      session_cwd: ev.session_cwd ?? null,
      created_at: ev.created_at,
    });
  }

  function scheduleWSReconnect(id: string) {
    if (wsStopped || wsRetryTimer) return;
    const delay = wsBackoff;
    wsBackoff = Math.min(wsBackoff * 2, WS_MAX_DELAY_MS);
    wsRetryTimer = setTimeout(() => {
      wsRetryTimer = null;
      if (!wsStopped && activeID.value === id) openWS(id);
    }, delay);
  }

  // A deliberate close: stop retrying. setActive(null) and $reset use this.
  function closeWS() {
    wsStopped = true;
    if (wsRetryTimer) {
      clearTimeout(wsRetryTimer);
      wsRetryTimer = null;
    }
    if (ws) {
      try { ws.close(); } catch { /* ignore */ }
      ws = null;
    }
  }

  function openWS(id: string) {
    if (typeof window === 'undefined') return;
    wsStopped = false;
    if (ws) {
      try { ws.close(); } catch { /* ignore */ }
      ws = null;
    }
    ws = new WebSocket(wsURL(id));
    ws.addEventListener('open', () => {
      wsBackoff = WS_DIAL_DELAY_MS;   // a healthy connection resets the ladder
      void catchUp(id);               // whatever we missed while away
    });
    ws.addEventListener('message', (evt) => {
      try {
        const data = JSON.parse((evt as MessageEvent).data as string) as ConversationEvent;
        if (data && data.conversation_id === id) applyEvent(data);
      } catch { /* ignore malformed */ }
    });
    ws.addEventListener('close', () => {
      ws = null;
      scheduleWSReconnect(id);
    });
  }

  async function loadList(opts: { anchorKind?: string; anchorID?: string; pending?: boolean; limit?: number } = {}) {
    const local = ++listSeq.current;
    listLoading.value = true;
    listError.value = null;
    try {
      const resp = await apiList({
        anchorKind: opts.anchorKind, anchorID: opts.anchorID,
        pending: opts.pending, limit: opts.limit,
      });
      if (local !== listSeq.current) return;
      list.value = resp.conversations;
      for (const c of resp.conversations) byID.value[c.id] = c;
    } catch (e) {
      listError.value = e instanceof BackendError ? e : new BackendError(0, '', String(e));
    } finally {
      listLoading.value = false;
    }
  }

  // Fetch the most-recent conversation anchored to (kind, id), independent of
  // the shared `list` and its sequence guard. openFor uses this so a
  // concurrent loadList (e.g. the ChatHeader's own list load) can't clobber
  // the result and leave the panel on an empty "New conversation".
  async function mostRecentForAnchor(kind: string, id: string): Promise<Conversation | null> {
    const resp = await apiList({ anchorKind: kind, anchorID: id, limit: 1 });
    const c = resp.conversations[0] ?? null;
    if (c) byID.value[c.id] = c;
    return c;
  }

  async function loadConversation(id: string) {
    const local = (convSeq[id] = (convSeq[id] ?? 0) + 1);
    try {
      const [conv, msgs] = await Promise.all([apiGet(id), apiListMessages(id)]);
      if (local !== convSeq[id]) return;
      byID.value[id] = conv;
      messagesByConv.value[id] = msgs.messages;
    } catch (e) {
      listError.value = e instanceof BackendError ? e : new BackendError(0, '', String(e));
    }
  }

  async function create(req: CreateConversationRequest): Promise<Conversation> {
    const conv = await apiCreate(req);
    byID.value[conv.id] = conv;
    list.value = [conv, ...list.value];
    listSeq.current++;
    messagesByConv.value[conv.id] = [];
    return conv;
  }

  async function deleteOne(id: string): Promise<void> {
    await apiDelete(id);
    delete byID.value[id];
    delete messagesByConv.value[id];
    list.value = list.value.filter((c) => c.id !== id);
    if (activeID.value === id) await setActive(null);
  }

  async function rename(id: string, title: string): Promise<void> {
    const conv = byID.value[id];
    if (!conv) return;
    const prev = conv.title;
    conv.title = title; // optimistic
    try {
      const updated = await apiRename(id, { title });
      Object.assign(conv, updated);
    } catch (e) {
      conv.title = prev; // rollback
      throw e;
    }
  }

  // Per-message delivery state. Three states, because the transport has one
  // delivery ack — "delivered" already means the session has it and is working.
  // A late ack beats the undelivered timer, which was only a guess.
  function deliveryState(messageID: string): 'sent' | 'delivered' | 'undelivered' {
    if (presence.isDelivered(messageID)) return 'delivered';
    if (undelivered.value[messageID]) return 'undelivered';
    return 'sent';
  }

  async function setActive(id: string | null): Promise<void> {
    if (activeID.value === id) return;
    activeID.value = id;
    if (id === null) {
      closeWS();
      return;
    }
    if (!messagesByConv.value[id]) await loadConversation(id);
    openWS(id);
  }

  async function optimisticPost(content: string, selectionText: string | null = null): Promise<void> {
    const id = activeID.value;
    if (!id) throw new Error('no active conversation');

    const req: AppendMessageRequest = { role: 'user', content };
    if (selectionText) req.selection_text = selectionText;
    if (presence.selectedSessionID) req.target_session_id = presence.selectedSessionID;

    let msg: Message;
    try {
      msg = await apiAppendMessage(id, req);
    } catch (e) {
      // 409 means the chosen session died between the picker and the post.
      // Drop the stale selection so the picker re-prompts instead of sending
      // into the void again.
      if (e instanceof BackendError && e.status === 409) presence.select(null);
      throw e;
    }

    pendingIDs.add(msg.id);
    const arr = messagesByConv.value[id] ?? [];
    // Defensive: if the WS echo beat us here, don't double-append.
    if (!arr.some((m) => m.id === msg.id)) arr.push(msg);
    messagesByConv.value[id] = [...arr];
    armUndeliveredTimer(msg.id);

    // Asking a question about a section escalates it to GRILLED. Only a user
    // post (this function) to a conversation anchored to a section (a card that
    // has a parent) triggers it; escalateReviewGrade never lowers a grade.
    const conv = byID.value[id];
    if (conv?.anchor_kind === 'card' && conv.anchor_id) {
      const cardsStore = useCardsStore();
      const card = cardsStore.byID[conv.anchor_id];
      if (card?.parent_card_id) void cardsStore.escalateReviewGrade(conv.anchor_id, 'GRILLED');
    }
  }

  const anchorTitleCache = ref<Record<string, string>>({});
  // A card anchor's genesis (its provenance note), captured while resolving the
  // title so the chat page can show it as a tooltip on the source pill. Groups
  // have no genesis, so only card anchors populate this.
  const anchorGenesisCache = ref<Record<string, string>>({});

  async function resolveAnchorTitle(kind: string | null, id: string | null): Promise<string | null> {
    if (!kind || !id) return null;
    const k = `${kind}:${id}`;
    if (anchorTitleCache.value[k]) return anchorTitleCache.value[k];
    try {
      let title: string | null = null;
      if (kind === 'card') {
        const card = await apiGetCard(id);
        title = card.title;
        if (card.genesis) {
          anchorGenesisCache.value = { ...anchorGenesisCache.value, [k]: card.genesis };
        }
      } else if (kind === 'group') {
        title = (await apiGetGroup(id)).name;
      }
      if (title) {
        anchorTitleCache.value = { ...anchorTitleCache.value, [k]: title };
      }
      return title;
    } catch {
      return null;
    }
  }

  // The genesis of a card anchor, if resolveAnchorTitle has fetched it. Reads
  // the cache only — call resolveAnchorTitle first to populate it.
  function anchorGenesis(kind: string | null, id: string | null): string | null {
    if (!kind || !id) return null;
    return anchorGenesisCache.value[`${kind}:${id}`] ?? null;
  }

  // The status of the last user message, when it is still the last message.
  // Once the assistant answers, the answer is the status and this is null.
  const threadStatus = computed<{ state: ThreadState; messageID: string } | null>(() => {
    const id = activeID.value;
    if (!id) return null;
    const arr = messagesByConv.value[id] ?? [];
    if (arr.length === 0) return null;

    const last = arr[arr.length - 1];
    if (last.role !== 'user') return null;

    // A late ack still wins: delivered is a fact, the timer was only a guess.
    if (presence.isDelivered(last.id)) return { state: 'delivered', messageID: last.id };
    if (undelivered.value[last.id]) return { state: 'undelivered', messageID: last.id };
    return { state: 'posted', messageID: last.id };
  });

  // Re-publishes a stored message at the currently selected session. It never
  // inserts a row, so the thread does not grow a duplicate.
  async function resend(messageID: string): Promise<void> {
    const id = activeID.value;
    if (!id || !presence.selectedSessionID) return;

    await apiDispatchMessage(id, messageID, { target_session_id: presence.selectedSessionID });
    const next = { ...undelivered.value };
    delete next[messageID];
    undelivered.value = next;
    armUndeliveredTimer(messageID);
  }

  const pendingAssistant = computed<boolean>(() => {
    const id = activeID.value;
    if (!id) return false;
    const arr = messagesByConv.value[id] ?? [];
    if (arr.length === 0) return false;
    return arr[arr.length - 1].role === 'user';
  });

  function $reset() {
    closeWS();
    list.value = [];
    byID.value = {};
    messagesByConv.value = {};
    activeID.value = null;
    listError.value = null;
    pendingIDs.clear();
    for (const t of undeliveredTimers.values()) clearTimeout(t);
    undeliveredTimers.clear();
    undelivered.value = {};
    // The store is a singleton for the app's lifetime, so there is no unmount
    // hook for this; $reset is the only teardown.
    if (typeof document !== 'undefined' && boundVisibilityHandler === onVisibilityChange) {
      document.removeEventListener('visibilitychange', boundVisibilityHandler);
      boundVisibilityHandler = null;
    }
  }

  return {
    list, listLoading, listError, byID, messagesByConv, activeID, pendingAssistant,
    threadStatus, undelivered,
    loadList, mostRecentForAnchor, loadConversation, create, deleteOne, rename, setActive, optimisticPost, resend,
    deliveryState, catchUp,
    resolveAnchorTitle, anchorGenesis,
    $reset,
  };
});
