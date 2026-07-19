import { describe, it, expect, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import SessionPresencePill from './SessionPresencePill.vue';
import { usePresenceStore } from '../../stores/presence';
import type { ChannelSession } from '../../types/entity';

const sess = (): ChannelSession => ({
  instance_id: 'i1', session_id: 's1aaaa2222bbbb', cwd: '/home/x/zen',
  started_at: '', client_name: '', client_version: '', connected_at: '',
});

describe('SessionPresencePill', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it('shows the session name and short id, not the full cwd', () => {
    const p = usePresenceStore();
    p.sessions = [sess()]; p.select('s1aaaa2222bbbb');
    const w = mount(SessionPresencePill);
    const t = w.find('[data-test="presence-pill"]').text();
    expect(t).toContain('zen');          // friendly name (cwd basename) stays
    expect(t).toContain('s1aaaa22');     // short session id as the detail
    expect(t).not.toContain('/home/x/zen');
    expect(w.find('[data-test="presence-dot"]').classes().join(' ')).toContain('bg-accent-fg');
  });

  it('shows a muted dot when the selected session is gone', () => {
    const p = usePresenceStore();
    p.sessions = []; p.select('s1aaaa2222bbbb');
    const w = mount(SessionPresencePill);
    expect(w.find('[data-test="presence-dot"]').classes().join(' ')).toContain('bg-muted-fg');
  });

  it('emits toggle when clicked', async () => {
    const p = usePresenceStore(); p.sessions = [sess()]; p.select('s1aaaa2222bbbb');
    const w = mount(SessionPresencePill);
    await w.find('[data-test="presence-pill"]').trigger('click');
    expect(w.emitted('toggle')).toBeTruthy();
  });
});
