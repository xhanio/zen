import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount, RouterLinkStub, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import TrashView from './TrashView.vue';
import { useTrashStore } from '../stores/trash';

vi.mock('../api/client', () => ({
  listTrash: vi.fn().mockResolvedValue({
    cards: [{
      id: 'c1', title: 'Hello card', content: 'x', format: 'markdown', level: null,
      group_id: 'g1', position: 0, tags: [],
      genesis: '', deleted_at: '2026-06-28T00:00:00Z',
      parent_card_id: null, source_conversation_id: null,
      created_at: 'x', updated_at: 'x',
    }],
  }),
  restoreCard: vi.fn(),
  purgeCard: vi.fn(),
}));

describe('TrashView', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it('renders the row title as a router-link to /c/<id>', async () => {
    const store = useTrashStore();
    await store.load();
    const w = mount(TrashView, {
      global: { stubs: { 'router-link': RouterLinkStub, ConfirmDialog: true } },
    });
    await flushPromises();
    const title = w.find('[data-test="trash-row-title"]');
    expect(title.exists()).toBe(true);
    expect(title.text()).toBe('Hello card');
    expect(title.getComponent(RouterLinkStub).props().to).toEqual({
      name: 'card',
      params: { cardId: 'c1' },
    });
  });
});
