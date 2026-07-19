import { describe, it, expect, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { mount, RouterLinkStub } from '@vue/test-utils';
import LeftRail from './LeftRail.vue';
import { useRailPrefsStore } from '../../stores/railPrefs';

const mountRail = () =>
  mount(LeftRail, {
    slots: { tree: '<div data-test="tree-slot" />', tags: '<div data-test="tags-slot" />' },
    global: { stubs: { RouterLink: RouterLinkStub, 'router-link': RouterLinkStub } },
  });

beforeEach(() => {
  setActivePinia(createPinia());
  localStorage.clear();
});

describe('LeftRail', () => {
  it('renders the workspace links and both slots', () => {
    const w = mountRail();
    const links = w.findAllComponents(RouterLinkStub).map((l) => l.props('to'));
    expect(links).toEqual(expect.arrayContaining(['/', '/chat', '/trash']));
    expect(w.find('[data-test="tree-slot"]').exists()).toBe(true);
    expect(w.find('[data-test="tags-slot"]').exists()).toBe(true);
  });

  it('shows a Groups eyebrow above the tree', () => {
    expect(mountRail().text()).toContain('Groups');
  });

  it('binds the persisted rail width and exposes a resize handle', () => {
    useRailPrefsStore().setWidth(320);
    const w = mountRail();
    expect(w.find('aside').attributes('style')).toContain('width: 320px');
    expect(w.find('[data-test="rail-resize"]').exists()).toBe(true);
  });
});
