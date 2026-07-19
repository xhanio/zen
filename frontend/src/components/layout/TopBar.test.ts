import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { createRouter, createWebHistory } from 'vue-router';
import TopBar from './TopBar.vue';
import { usePresenceStore } from '../../stores/presence';

vi.mock('../../api/client', () => ({ search: vi.fn(async () => ({ hits: [] })) }));

const router = createRouter({
  history: createWebHistory(),
  routes: [{ path: '/', component: { template: '<div/>' } }],
});

function mountTopBar() {
  return mount(TopBar, { global: { plugins: [router] } });
}

function fakeSession(id: string, cwd: string) {
  return {
    instance_id: 'i-' + id, session_id: id, cwd,
    started_at: '', client_name: '', client_version: '', connected_at: '',
  };
}

beforeEach(() => {
  setActivePinia(createPinia());
  (global as any).WebSocket = class {
    addEventListener() { /* noop */ }
    removeEventListener() { /* noop */ }
    close() { /* noop */ }
  };
});

describe('TopBar presence indicator', () => {
  it('reports no sessions when the registry is empty', async () => {
    const w = mountTopBar();
    await w.vm.$nextTick();
    expect(w.text()).toContain('No AI connected');
  });

  it('pluralises a single session', async () => {
    const store = usePresenceStore();
    const w = mountTopBar();
    store.sessions = [fakeSession('s1', '/repo')];
    await w.vm.$nextTick();
    expect(w.text()).toContain('1 session');
    expect(w.text()).not.toContain('1 sessions');
  });

  it('counts several sessions', async () => {
    const store = usePresenceStore();
    const w = mountTopBar();
    store.sessions = [fakeSession('s1', '/a'), fakeSession('s2', '/b')];
    await w.vm.$nextTick();
    expect(w.text()).toContain('2 sessions');
  });
});

describe('TopBar version label', () => {
  it('shows the app version next to the brand', async () => {
    const w = mountTopBar();
    await w.vm.$nextTick();
    const v = w.find('[data-test="app-version"]');
    expect(v.exists()).toBe(true);
    expect(v.text()).toBe(__APP_VERSION__);
  });
});
