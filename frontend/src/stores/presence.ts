import { ref } from 'vue';
import { defineStore } from 'pinia';
import type { ChannelSession } from '../types/entity';

const DIAL_DELAY_MS = 1000;
const MAX_DELAY_MS = 30000;

const LABELS_KEY = 'zen:sessionLabels';
const LABEL_TTL_MS = 30 * 24 * 60 * 60 * 1000; // 30 days

interface LabelEntry {
  label: string;
  lastSeen: number;
}

// Labels accumulate for sessions that will never return. Drop anything unseen
// for 30 days on load, or the map grows without bound.
function loadLabels(): Record<string, LabelEntry> {
  if (typeof localStorage === 'undefined') return {};
  try {
    const raw = localStorage.getItem(LABELS_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as Record<string, LabelEntry>;
    const cutoff = Date.now() - LABEL_TTL_MS;
    const kept: Record<string, LabelEntry> = {};
    for (const [id, entry] of Object.entries(parsed)) {
      if (entry && typeof entry.label === 'string' && entry.lastSeen > cutoff) {
        kept[id] = entry;
      }
    }
    return kept;
  } catch {
    return {};
  }
}

function saveLabels(labels: Record<string, LabelEntry>) {
  if (typeof localStorage === 'undefined') return;
  try {
    localStorage.setItem(LABELS_KEY, JSON.stringify(labels));
  } catch { /* quota — a lost label is not worth throwing over */ }
}

// "/home/x/projects/zen" -> "zen". Empty cwd yields "".
function basename(cwd: string): string {
  const trimmed = cwd.replace(/\/+$/, '');
  const cut = trimmed.lastIndexOf('/');
  return cut === -1 ? trimmed : trimmed.slice(cut + 1);
}

export const usePresenceStore = defineStore('presence', () => {
  const sessions = ref<ChannelSession[]>([]);
  const selectedSessionID = ref<string | null>(null);
  const connected = ref(false);
  const labels = ref<Record<string, LabelEntry>>(loadLabels());

  // Delivery is a UI hint with no server-side memory: a refresh forgets it,
  // exactly as the spec accepts.
  const deliveries = ref<Record<string, true>>({});

  function isDelivered(messageID: string): boolean {
    return deliveries.value[messageID] === true;
  }

  function markDelivered(messageID: string) {
    if (!messageID) return;
    deliveries.value = { ...deliveries.value, [messageID]: true };
  }

  let ws: WebSocket | null = null;
  let retryTimer: ReturnType<typeof setTimeout> | null = null;
  let backoff = DIAL_DELAY_MS;
  let stopped = false;

  function wsURL(): string {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${proto}//${window.location.host}/api/v1/_sessions/ws`;
  }

  function labelFor(sessionID: string): string | null {
    return labels.value[sessionID]?.label ?? null;
  }

  // A user label, else the cwd's last path segment, else a session-id prefix.
  // The prefix is the degraded case the spec accepts when
  // CLAUDE_CODE_SESSION_ID is absent and session_id is a bare ulid.
  function displayName(s: ChannelSession): string {
    const label = labelFor(s.session_id);
    if (label) return label;
    const base = basename(s.cwd);
    if (base) return base;
    return s.session_id.slice(0, 6);
  }

  // The durable per-turn badge: a live label if you have named the session,
  // else the cwd's last segment, else a short id. Null when the turn has no
  // session (an undelivered post, or a pre-v0.16 row).
  function badgeFor(sessionID?: string | null, sessionCwd?: string | null): string | null {
    if (!sessionID) return null;
    const label = labelFor(sessionID);
    if (label) return label;
    const base = basename(sessionCwd ?? '');
    if (base) return base;
    return sessionID.slice(0, 6);
  }

  // A stable colour per session so two sessions stay visually distinct even
  // when their cwd basenames collide. Deterministic hash → hue.
  function sessionColor(sessionID: string): string {
    let h = 0;
    for (let i = 0; i < sessionID.length; i++) h = (h * 31 + sessionID.charCodeAt(i)) >>> 0;
    return `hsl(${h % 360} 65% 50%)`;
  }

  function setLabel(sessionID: string, label: string) {
    const trimmed = label.trim();
    const next = { ...labels.value };
    if (trimmed === '') {
      delete next[sessionID];
    } else {
      next[sessionID] = { label: trimmed, lastSeen: Date.now() };
    }
    labels.value = next;
    saveLabels(next);
  }

  // Seeing a session refreshes its label's clock, so a session you use daily
  // never ages out of the map.
  function touchLabels(next: ChannelSession[]) {
    let changed = false;
    const updated = { ...labels.value };
    for (const s of next) {
      const entry = updated[s.session_id];
      if (entry) {
        updated[s.session_id] = { ...entry, lastSeen: Date.now() };
        changed = true;
      }
    }
    if (changed) {
      labels.value = updated;
      saveLabels(updated);
    }
  }

  // Snapshots are the whole registry, so this is a replace, never a merge.
  function applySessions(next: ChannelSession[]) {
    sessions.value = next;
    touchLabels(next);

    const live = new Set(next.map((s) => s.session_id));
    const current = selectedSessionID.value;

    if (current && live.has(current)) return;      // still there; leave it
    if (next.length === 1) {
      selectedSessionID.value = next[0].session_id; // lone session: take it
      return;
    }
    // Zero sessions, or several and ours is gone: make the user choose.
    selectedSessionID.value = null;
  }

  function onMessage(raw: string) {
    let frame: { kind?: string; sessions?: ChannelSession[]; message_id?: string; state?: string };
    try {
      frame = JSON.parse(raw);
    } catch {
      return;
    }
    // /_sessions/ws is the SPA's control stream, versioned by `kind`. Unknown
    // kinds are ignored so an old tab survives a newer backend.
    if (frame.kind === 'sessions' && Array.isArray(frame.sessions)) {
      applySessions(frame.sessions);
      return;
    }
    if (frame.kind === 'delivery' && frame.message_id) {
      markDelivered(frame.message_id);
    }
  }

  function scheduleReconnect() {
    if (stopped || retryTimer) return;
    const delay = backoff;
    backoff = Math.min(backoff * 2, MAX_DELAY_MS);
    retryTimer = setTimeout(() => {
      retryTimer = null;
      open();
    }, delay);
  }

  function open() {
    if (stopped || typeof window === 'undefined') return;
    ws = new WebSocket(wsURL());
    ws.addEventListener('open', () => {
      connected.value = true;
      backoff = DIAL_DELAY_MS; // healthy connection resets the ladder
    });
    ws.addEventListener('message', (evt) => onMessage((evt as MessageEvent).data as string));
    ws.addEventListener('close', () => {
      connected.value = false;
      ws = null;
      scheduleReconnect();
    });
  }

  function connect() {
    stopped = false;
    if (ws) return;
    backoff = DIAL_DELAY_MS;
    open();
  }

  function disconnect() {
    stopped = true;
    if (retryTimer) {
      clearTimeout(retryTimer);
      retryTimer = null;
    }
    if (ws) {
      try { ws.close(); } catch { /* ignore */ }
      ws = null;
    }
    connected.value = false;
  }

  function select(sessionID: string | null) {
    selectedSessionID.value = sessionID;
  }

  function $reset() {
    disconnect();
    sessions.value = [];
    selectedSessionID.value = null;
    deliveries.value = {};
  }

  return {
    sessions, selectedSessionID, connected, labels, deliveries,
    connect, disconnect, select,
    labelFor, displayName, setLabel, badgeFor, sessionColor, cwdBasename: basename,
    isDelivered, markDelivered,
    $reset,
  };
});
