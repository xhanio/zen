import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import AskBubble from './AskBubble.vue';
import { useChatSidebar } from '../../composables/useChatSidebar';

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
  global.fetch = vi.fn().mockResolvedValue({
    ok: true, status: 200,
    json: async () => ({ conversations: [] }),
    text: async () => '',
  }) as unknown as typeof fetch;
  const s = useChatSidebar();
  s.close();
  s.anchorKind.value = null;
  s.anchorID.value = null;
});

describe('AskBubble', () => {
  it('renders nothing when rect is null', () => {
    const wrapper = mount(AskBubble, {
      props: { rect: null, selectionText: 'x', anchorKind: 'card', anchorId: '01' },
    });
    expect(wrapper.find('button').exists()).toBe(false);
  });

  it('renders the button when rect is provided', () => {
    const rect = new DOMRect(10, 10, 100, 20);
    const wrapper = mount(AskBubble, {
      props: { rect, selectionText: 'hi', anchorKind: 'card', anchorId: '01CARD' },
    });
    expect(wrapper.find('button').exists()).toBe(true);
    expect(wrapper.text()).toContain('Ask');
  });

  it('click opens the sidebar with anchor + selection', async () => {
    const rect = new DOMRect(0, 0, 50, 20);
    const wrapper = mount(AskBubble, {
      props: { rect, selectionText: 'sel', anchorKind: 'card', anchorId: '01CARD' },
    });
    const sidebar = useChatSidebar();
    await wrapper.find('button').trigger('click');
    await wrapper.vm.$nextTick();
    expect(sidebar.open.value).toBe(true);
    expect(sidebar.anchorKind.value).toBe('card');
    expect(sidebar.anchorID.value).toBe('01CARD');
    expect(sidebar.pendingSelection.value).toBe('sel');
  });
});
