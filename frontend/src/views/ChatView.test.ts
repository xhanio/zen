import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import ChatView from './ChatView.vue';
import { useConversationsStore } from '../stores/conversations';
import { useTilePrefsStore } from '../stores/tilePrefs';

beforeEach(() => {
  localStorage.clear();
  setActivePinia(createPinia());
  global.fetch = vi.fn(async (url: unknown): Promise<unknown> => {
    // The anchor pill resolves the card's title via GET /cards/<id>.
    if (typeof url === 'string' && url.includes('/cards/')) {
      return {
        ok: true, status: 200,
        json: async () => ({ id: '01CARD', title: 'Why SQLite + FTS5', genesis: 'Decomposed from card 01ROOT' }),
        text: async () => '',
      };
    }
    return {
      ok: true, status: 200,
      json: async () => ({
        conversations: [
          {
            id: 'C1',
            title: 'hi',
            anchor_kind: 'card',
            anchor_id: '01CARD',
            created_at: '2026-06-27T00:00:00Z',
            last_message_at: '2026-06-27T00:00:00Z',
          },
        ],
      }),
      text: async () => '',
    };
  }) as unknown as typeof fetch;
});

describe('ChatView list', () => {
  it('shows the card title (not the id) on an anchor pill linking to /c/<id>', async () => {
    const wrapper = mount(ChatView, {
      global: {
        stubs: {
          RouterLink: { template: '<a :data-href="to"><slot /></a>', props: ['to'] },
          ChatThread: true,
          ChatComposer: true,
        },
      },
    });
    await flushPromises();
    await wrapper.vm.$nextTick();

    const link = wrapper.find('[data-href="/c/01CARD"]');
    expect(link.exists()).toBe(true);
    expect(link.text()).toContain('Why SQLite + FTS5');
    expect(link.text()).not.toContain('01CARD');
  });

  it('shows the source card genesis as a tooltip on the anchor pill', async () => {
    const wrapper = mount(ChatView, {
      global: {
        stubs: {
          RouterLink: { template: '<a :data-href="to"><slot /></a>', props: ['to'] },
          ChatThread: true,
          ChatComposer: true,
        },
      },
    });
    await flushPromises();
    await wrapper.vm.$nextTick();

    expect(wrapper.find('[data-href="/c/01CARD"]').attributes('title')).toBe('Decomposed from card 01ROOT');
  });

  it('deletes a conversation from the list after a confirm', async () => {
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true);
    const wrapper = mount(ChatView, {
      global: {
        stubs: {
          RouterLink: { template: '<a :data-href="to"><slot /></a>', props: ['to'] },
          ChatThread: true,
          ChatComposer: true,
        },
      },
    });
    await flushPromises();
    const store = useConversationsStore();
    const delSpy = vi.spyOn(store, 'deleteOne').mockResolvedValue();

    await wrapper.find('[data-test="conv-delete"]').trigger('click');
    expect(delSpy).toHaveBeenCalledWith('C1');
    confirmSpy.mockRestore();
  });
});

describe('ChatView list splitter', () => {
  const mountView = () =>
    mount(ChatView, {
      global: {
        stubs: {
          RouterLink: { template: '<a :data-href="to"><slot /></a>', props: ['to'] },
          ChatThread: true,
          ChatComposer: true,
        },
      },
    });

  it('renders a resize splitter between the list and the thread', async () => {
    const w = mountView();
    await flushPromises();
    expect(w.find('[data-test="chat-list-splitter"]').exists()).toBe(true);
  });

  it('binds the persisted list width to the list column', async () => {
    useTilePrefsStore().setChatListWidth(320);
    const w = mountView();
    await flushPromises();
    expect(w.find('[data-test="chat-list-col"]').attributes('style')).toContain('width: 320px');
  });
});
