import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { usePresenceStore } from './presence';
import type { ChannelSession } from '../types/entity';

class MockWS {
  static instances: MockWS[] = [];
  url: string;
  readyState = 1;
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
  close() {
    this.readyState = 3;
    (this.listeners.close ?? []).forEach((c) => c({}));
  }
  open() { (this.listeners.open ?? []).forEach((c) => c({})); }
  simulate(payload: unknown) {
    (this.listeners.message ?? []).forEach((c) => c({ data: JSON.stringify(payload) }));
  }
  static last(): MockWS { return MockWS.instances[MockWS.instances.length - 1]; }
}

function session(id: string, cwd = '/home/x/repo'): ChannelSession {
  return {
    instance_id: 'i-' + id,
    session_id: id,
    cwd,
    started_at: '2026-07-08T10:00:00Z',
    client_name: 'claude-code',
    client_version: '2.1.205',
    connected_at: '2026-07-08T10:00:01Z',
  };
}

function sessionsFrame(...s: ChannelSession[]) {
  return { kind: 'sessions', sessions: s };
}

beforeEach(() => {
  setActivePinia(createPinia());
  localStorage.clear();
  (global as any).WebSocket = MockWS;
  MockWS.instances = [];
  vi.useRealTimers();
});

describe('presence store', () => {
  it('badgeFor prefers a live label, then cwd basename, then a short id', () => {
    const p = usePresenceStore();
    expect(p.badgeFor(null, null)).toBeNull();
    expect(p.badgeFor('sess-A', '/home/x/zen')).toBe('zen');
    p.setLabel('sess-A', 'grid-refactor');
    expect(p.badgeFor('sess-A', '/home/x/zen')).toBe('grid-refactor');
    expect(p.badgeFor('0123456789', '')).toBe('012345');
  });

  it('sessionColor is deterministic and differs across sessions', () => {
    const p = usePresenceStore();
    expect(p.sessionColor('sess-A')).toBe(p.sessionColor('sess-A'));
    expect(p.sessionColor('sess-A')).not.toBe(p.sessionColor('sess-B'));
  });

  it('applies a sessions snapshot', () => {
    const store = usePresenceStore();
    store.connect();
    MockWS.last().simulate(sessionsFrame(session('s1'), session('s2')));

    expect(store.sessions.map((s) => s.session_id)).toEqual(['s1', 's2']);
  });

  it('auto-selects when exactly one session is live', () => {
    const store = usePresenceStore();
    store.connect();
    MockWS.last().simulate(sessionsFrame(session('s1')));

    expect(store.selectedSessionID).toBe('s1');
  });

  it('does not auto-select when several sessions are live', () => {
    const store = usePresenceStore();
    store.connect();
    MockWS.last().simulate(sessionsFrame(session('s1'), session('s2')));

    expect(store.selectedSessionID).toBeNull();
  });

  it('keeps the selection when the selected session is still live', () => {
    const store = usePresenceStore();
    store.connect();
    MockWS.last().simulate(sessionsFrame(session('s1'), session('s2')));
    store.select('s2');
    MockWS.last().simulate(sessionsFrame(session('s1'), session('s2')));

    expect(store.selectedSessionID).toBe('s2');
  });

  // The selected session vanishes mid-thread. If exactly one remains, take it
  // silently — same rule as first open. Otherwise prompt, i.e. clear.
  it('re-selects silently when the selected session dies and one remains', () => {
    const store = usePresenceStore();
    store.connect();
    MockWS.last().simulate(sessionsFrame(session('s1'), session('s2')));
    store.select('s2');
    MockWS.last().simulate(sessionsFrame(session('s1')));

    expect(store.selectedSessionID).toBe('s1');
  });

  it('clears the selection when the selected session dies and several remain', () => {
    const store = usePresenceStore();
    store.connect();
    MockWS.last().simulate(sessionsFrame(session('s1'), session('s2'), session('s3')));
    store.select('s3');
    MockWS.last().simulate(sessionsFrame(session('s1'), session('s2')));

    expect(store.selectedSessionID).toBeNull();
  });

  it('ignores frames of an unknown kind', () => {
    const store = usePresenceStore();
    store.connect();
    MockWS.last().simulate({ kind: 'delivery', message_id: '01K', state: 'delivered' });

    expect(store.sessions).toEqual([]);
  });

  it('reconnects after the socket closes', async () => {
    vi.useFakeTimers();
    const store = usePresenceStore();
    store.connect();
    expect(MockWS.instances.length).toBe(1);

    MockWS.last().close();
    await vi.advanceTimersByTimeAsync(1000);

    expect(MockWS.instances.length).toBe(2);
    vi.useRealTimers();
  });

  it('backs off exponentially across repeated failures', async () => {
    vi.useFakeTimers();
    const store = usePresenceStore();
    store.connect();

    MockWS.last().close();
    await vi.advanceTimersByTimeAsync(1000);
    expect(MockWS.instances.length).toBe(2);

    // Second failure waits 2s, not 1s.
    MockWS.last().close();
    await vi.advanceTimersByTimeAsync(1000);
    expect(MockWS.instances.length).toBe(2);
    await vi.advanceTimersByTimeAsync(1000);
    expect(MockWS.instances.length).toBe(3);

    vi.useRealTimers();
  });

  it('resets backoff after a successful open', async () => {
    vi.useFakeTimers();
    const store = usePresenceStore();
    store.connect();

    MockWS.last().close();
    await vi.advanceTimersByTimeAsync(1000);
    MockWS.last().close();
    await vi.advanceTimersByTimeAsync(2000);
    expect(MockWS.instances.length).toBe(3);

    MockWS.last().open();      // healthy connection
    MockWS.last().close();
    await vi.advanceTimersByTimeAsync(1000);  // back to the 1s floor
    expect(MockWS.instances.length).toBe(4);

    vi.useRealTimers();
  });

  it('disconnect stops reconnecting', async () => {
    vi.useFakeTimers();
    const store = usePresenceStore();
    store.connect();
    store.disconnect();

    await vi.advanceTimersByTimeAsync(5000);
    expect(MockWS.instances.length).toBe(1);
    vi.useRealTimers();
  });

  it('falls back to the cwd basename when no label is set', () => {
    const store = usePresenceStore();
    expect(store.displayName(session('s1', '/home/x/projects/zen'))).toBe('zen');
  });

  it('falls back to a session-id prefix when cwd is empty', () => {
    const store = usePresenceStore();
    expect(store.displayName(session('abcdef123456', ''))).toBe('abcdef');
  });

  it('prefers a user label over the cwd', () => {
    const store = usePresenceStore();
    store.setLabel('s1', 'v0.13 spec');
    expect(store.displayName(session('s1', '/home/x/projects/zen'))).toBe('v0.13 spec');
  });

  it('persists labels across store instances, keyed by session id', () => {
    const first = usePresenceStore();
    first.setLabel('s1', 'grid refactor');

    setActivePinia(createPinia());
    const second = usePresenceStore();
    expect(second.labelFor('s1')).toBe('grid refactor');
  });

  // instance_id is reborn on every channel restart; a rename must survive it.
  it('keeps the label when the same session returns with a new instance id', () => {
    const store = usePresenceStore();
    store.connect();
    MockWS.last().simulate(sessionsFrame(session('s1')));
    store.setLabel('s1', 'my session');

    const reborn = { ...session('s1'), instance_id: 'i-brand-new' };
    MockWS.last().simulate(sessionsFrame(reborn));

    expect(store.displayName(reborn)).toBe('my session');
  });

  it('clears a label when set to empty', () => {
    const store = usePresenceStore();
    store.setLabel('s1', 'x');
    store.setLabel('s1', '  ');
    expect(store.labelFor('s1')).toBeNull();
  });

  it('prunes labels for sessions unseen for 30 days', () => {
    const old = Date.now() - 31 * 24 * 60 * 60 * 1000;
    localStorage.setItem('zen:sessionLabels', JSON.stringify({
      stale: { label: 'old one', lastSeen: old },
      fresh: { label: 'new one', lastSeen: Date.now() },
    }));

    setActivePinia(createPinia());
    const store = usePresenceStore();
    expect(store.labelFor('stale')).toBeNull();
    expect(store.labelFor('fresh')).toBe('new one');
  });

  it('records a delivery frame', () => {
    const store = usePresenceStore();
    store.connect();
    MockWS.last().simulate({ kind: 'delivery', message_id: '01MSG', state: 'delivered' });

    expect(store.isDelivered('01MSG')).toBe(true);
    expect(store.isDelivered('01OTHER')).toBe(false);
  });

  it('keeps sessions and deliveries independent on one socket', () => {
    const store = usePresenceStore();
    store.connect();
    MockWS.last().simulate(sessionsFrame(session('s1')));
    MockWS.last().simulate({ kind: 'delivery', message_id: '01MSG', state: 'delivered' });

    expect(store.sessions.length).toBe(1);
    expect(store.isDelivered('01MSG')).toBe(true);
  });

  it('refreshes lastSeen for sessions present in a snapshot', () => {
    const old = Date.now() - 29 * 24 * 60 * 60 * 1000;
    localStorage.setItem('zen:sessionLabels', JSON.stringify({
      s1: { label: 'kept alive', lastSeen: old },
    }));

    setActivePinia(createPinia());
    const store = usePresenceStore();
    store.connect();
    MockWS.last().simulate(sessionsFrame(session('s1')));

    const raw = JSON.parse(localStorage.getItem('zen:sessionLabels') as string);
    expect(raw.s1.lastSeen).toBeGreaterThan(old);
  });
});
