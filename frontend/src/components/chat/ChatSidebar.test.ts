import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import ChatSidebar from './ChatSidebar.vue';
import { useChatSidebar } from '../../composables/useChatSidebar';

beforeEach(() => {
  setActivePinia(createPinia());
  localStorage.clear();
  (global as any).WebSocket = class {
    constructor(_url: string) {}
    addEventListener() {} removeEventListener() {}
    close() {} send() {}
  };
  Object.defineProperty(window, 'location', {
    value: { protocol: 'http:', host: '127.0.0.1:5173' },
    writable: true,
  });
  global.fetch = vi.fn().mockResolvedValue({
    ok: true, status: 200,
    json: async () => ({ conversations: [] }),
    text: async () => '',
  }) as unknown as typeof fetch;
  // Reset singleton state between tests
  const s = useChatSidebar();
  s.close();
  s.anchorKind.value = null;
  s.anchorID.value = null;
});

describe('ChatSidebar', () => {
  it('is hidden by default', () => {
    const wrapper = mount(ChatSidebar);
    expect(wrapper.find('aside').exists()).toBe(false);
  });

  it('renders after openFor', async () => {
    const wrapper = mount(ChatSidebar);
    const sidebar = useChatSidebar();
    await sidebar.openFor(null, null);
    await wrapper.vm.$nextTick();
    expect(wrapper.find('aside').exists()).toBe(true);
    expect(wrapper.text()).toContain('New conversation');
  });

  it('close button hides the sidebar', async () => {
    const wrapper = mount(ChatSidebar);
    const sidebar = useChatSidebar();
    await sidebar.openFor(null, null);
    await wrapper.vm.$nextTick();
    await wrapper.find('button[aria-label="Close sidebar"]').trigger('click');
    expect(sidebar.open.value).toBe(false);
  });

  it('renders a RouterLink to /c/<id> with the card title when anchor is card', async () => {
    (global.fetch as any).mockImplementation(async (url: string) => {
      if (url.endsWith('/api/v1/cards/01CARD')) {
        return {
          ok: true, status: 200,
          json: async () => ({ id: '01CARD', title: 'My Card Title', content: '' }),
          text: async () => '',
        };
      }
      return {
        ok: true, status: 200,
        json: async () => ({ conversations: [] }),
        text: async () => '',
      };
    });

    const wrapper = mount(ChatSidebar, {
      global: {
        stubs: {
          RouterLink: {
            template: '<a :data-href="to"><slot /></a>',
            props: ['to'],
          },
        },
      },
    });
    const sidebar = useChatSidebar();
    await sidebar.openFor('card', '01CARD');
    await new Promise((r) => setTimeout(r, 10));
    await wrapper.vm.$nextTick();

    const link = wrapper.find('[data-href="/c/01CARD"]');
    expect(link.exists()).toBe(true);
    expect(link.text()).toContain('My Card Title');
  });
  it('restores a persisted width from localStorage', async () => {
    localStorage.setItem('zen:chatWidth', '520');
    const wrapper = mount(ChatSidebar);
    const sidebar = useChatSidebar();
    await sidebar.openFor(null, null);
    await wrapper.vm.$nextTick();
    expect(wrapper.find('[data-test="chat-panel"]').attributes('style')).toContain('520px');
  });

  it('clamps the width to [320, 720]', () => {
    const wrapper = mount(ChatSidebar);
    const vm = wrapper.vm as unknown as { width: number; setWidth: (n: number) => void };
    vm.setWidth(9999);
    expect(vm.width).toBe(720);
    vm.setWidth(10);
    expect(vm.width).toBe(320);
  });
});

