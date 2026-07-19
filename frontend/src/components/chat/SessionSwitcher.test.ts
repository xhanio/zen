import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import SessionSwitcher from './SessionSwitcher.vue';
import { usePresenceStore } from '../../stores/presence';
import type { ChannelSession } from '../../types/entity';

const sess = (id: string, cwd: string): ChannelSession => ({
  instance_id: 'i-' + id, session_id: id, cwd,
  started_at: '', client_name: '', client_version: '', connected_at: '',
});

describe('SessionSwitcher', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it('lists connected sessions and marks the selected one', () => {
    const p = usePresenceStore();
    p.sessions = [sess('aaaa1111bbbb', '/home/x/zen'), sess('cccc2222dddd', '/home/x/gopro')];
    p.select('aaaa1111bbbb');
    const w = mount(SessionSwitcher);
    const rows = w.findAll('[data-test="session-row"]');
    expect(rows.length).toBe(2);
    expect(rows[0].classes().join(' ')).toContain('bg-accent-bg');
    // The friendly name (cwd basename) stays as the primary label…
    expect(w.text()).toContain('zen');
    // …but the detail is the short session id, not the full cwd path.
    expect(w.text()).toContain('aaaa1111');
    expect(w.text()).not.toContain('/home/x/zen');
  });

  it('selects a session on click', async () => {
    const p = usePresenceStore();
    p.sessions = [sess('a', '/x/zen'), sess('b', '/x/gopro')];
    p.select('a');
    const spy = vi.spyOn(p, 'select');
    const w = mount(SessionSwitcher);
    await w.findAll('[data-test="session-row"]')[1].trigger('click');
    expect(spy).toHaveBeenCalledWith('b');
  });

  it('shows an empty state when no sessions are connected', () => {
    usePresenceStore().sessions = [];
    const w = mount(SessionSwitcher);
    expect(w.find('[data-test="session-row"]').exists()).toBe(false);
    expect(w.text()).toMatch(/no session/i);
  });
});
