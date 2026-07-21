import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount, RouterLinkStub, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import { createRouter, createMemoryHistory } from 'vue-router';
import CardView from './CardView.vue';

vi.mock('../api/client', () => ({
  getCard: vi.fn(),
  listCards: vi.fn(),
  createCard: vi.fn(),
  updateCard: vi.fn(),
  deleteCard: vi.fn(),
  restoreCard: vi.fn(),
  purgeCard: vi.fn(),
  listTags: vi.fn().mockResolvedValue([]),
  listChildren: vi.fn().mockResolvedValue([]),
  listConversations: vi.fn().mockResolvedValue({ conversations: [] }),
  getConversation: vi.fn().mockResolvedValue(null),
}));

const mockOpenForConversation = vi.fn().mockResolvedValue(undefined);
vi.mock('../composables/useChatSidebar', async () => {
  const actual = await vi.importActual<typeof import('../composables/useChatSidebar')>(
    '../composables/useChatSidebar',
  );
  return {
    ...actual,
    useChatSidebar: () => ({
      ...actual.useChatSidebar(),
      openForConversation: mockOpenForConversation,
    }),
  };
});

import { getCard, listChildren } from '../api/client';

const liveCard = {
  id: 'c1', title: 'Hello', content: 'body', summary: '', format: 'markdown' as const, level_entry_id: null,
  group_id: 'g1', position: 0, tags: ['x'],
  genesis: '', deleted_at: null,
  parent_card_id: null as string | null, source_conversation_id: null as string | null,
  created_at: 'x', updated_at: 'x',
  review_grade: 'LGTM' as const, review_score: null, reviewed_at: null as string | null,
};
const trashedCard = { ...liveCard, deleted_at: '2026-06-28T00:00:00Z' };

const mountView = (opts: { attachTo?: HTMLElement } = {}) =>
  mount(CardView, {
    props: { cardId: 'c1' },
    attachTo: opts.attachTo,
    global: {
      stubs: {
        'router-link': RouterLinkStub,
        ConfirmDialog: true,
        ContentBody: true,
        AskBubble: true,
        SectionConversationChip: {
          template: '<span data-test="conv-chip" :data-persistent="String(persistent)" :data-disabled="String(disabled)"></span>',
          props: { anchorId: String, sourceConversationId: String, persistent: Boolean, disabled: Boolean },
        },
      },
    },
  });

describe('CardView — live state (regression)', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    (getCard as any).mockResolvedValue(liveCard);
  });

  it('shows the trash ✕ and the card conversation chip; no discuss button; no trash banner', async () => {
    const w = mountView();
    await flushPromises();
    expect(w.find('[data-test="trash-banner"]').exists()).toBe(false);
    expect(w.find('[data-test="card-action-trash"]').exists()).toBe(true); // the ✕
    expect(w.find('[data-test="conv-chip"]').exists()).toBe(true); // persistent card chip
    expect(w.find('[data-test="conv-chip"]').attributes('data-persistent')).toBe('true');
    expect(w.find('[data-test="card-action-discuss"]').exists()).toBe(false); // removed
    expect(w.find('[data-test="card-action-restore"]').exists()).toBe(false);
    expect(w.find('[data-test="card-action-purge"]').exists()).toBe(false);
  });
});

describe('CardView — trashed state', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    (getCard as any).mockResolvedValue(trashedCard);
  });

  it('renders the trashed banner', async () => {
    const w = mountView();
    await flushPromises();
    const banner = w.find('[data-test="trash-banner"]');
    expect(banner.exists()).toBe(true);
    expect(banner.text()).toMatch(/This card is in Trash/i);
  });

  it('trashed: hides the ✕, still shows the (disabled) card chip, shows Restore + Delete Permanently', async () => {
    const w = mountView();
    await flushPromises();
    expect(w.find('[data-test="card-action-trash"]').exists()).toBe(false); // ✕ hidden when trashed
    expect(w.find('[data-test="conv-chip"]').attributes('data-disabled')).toBe('true');
    expect(w.find('[data-test="card-action-discuss"]').exists()).toBe(false);
    expect(w.find('[data-test="card-action-restore"]').exists()).toBe(true);
    expect(w.find('[data-test="card-action-purge"]').exists()).toBe(true);
  });

  it('passes readonly=true to TagChipEditor', async () => {
    const w = mountView();
    await flushPromises();
    const editor = w.findComponent({ name: 'TagChipEditor' });
    expect(editor.props('readonly')).toBe(true);
  });

  it('clicking Restore calls cardsStore.restore', async () => {
    const w = mountView();
    await flushPromises();
    const { useCardsStore } = await import('../stores/cards');
    const store = useCardsStore();
    const spy = vi.spyOn(store, 'restore').mockResolvedValue();
    await w.find('[data-test="card-action-restore"]').trigger('click');
    expect(spy).toHaveBeenCalledWith('c1');
  });

  it('clicking Delete Permanently opens the purge confirm dialog', async () => {
    const w = mountView();
    await flushPromises();
    await w.find('[data-test="card-action-purge"]').trigger('click');
    const purgeDialog = w
      .findAll('*')
      .find((el) => el.attributes('title') === 'Delete card permanently?');
    expect(purgeDialog).toBeDefined();
    expect(purgeDialog!.attributes('open')).toBe('true');
  });
});

describe('CardView — highlights', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    (getCard as any).mockResolvedValue({
      ...liveCard,
      content: 'The quick brown fox',
      references: [{
        id: 'r1', source_card_id: 'c1', derived_card_id: 'c2',
        conversation_id: 'conv1', selection_text: 'quick', created_at: 'x',
      }],
    });
  });

  it('clicking an element with data-ref-id opens the chat sidebar on the conversation', async () => {
    mockOpenForConversation.mockClear();
    const w = mountView({ attachTo: document.body });
    await flushPromises();
    const contentRoot = w.find('div.mb-2.relative').element as HTMLElement;
    const dummy = document.createElement('span');
    dummy.setAttribute('data-ref-id', 'r1');
    contentRoot.appendChild(dummy);
    dummy.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    await flushPromises();
    expect(mockOpenForConversation).toHaveBeenCalledWith('conv1');
    w.unmount();
  });
});

describe('CardView — highlights with null conversation', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    (getCard as any).mockResolvedValue({
      ...liveCard,
      content: 'The quick brown fox',
      references: [{
        id: 'r2', source_card_id: 'c1', derived_card_id: 'c2-derived',
        conversation_id: null, selection_text: 'quick', created_at: 'x',
      }],
    });
  });

  it('clicking a highlight with null conversation_id navigates to the derived card', async () => {
    mockOpenForConversation.mockClear();
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/c/:cardId', name: 'card', component: { template: '<div />' } },
      ],
    });
    const pushSpy = vi.spyOn(router, 'push');
    const w = mount(CardView, {
      props: { cardId: 'c1' },
      attachTo: document.body,
      global: {
        plugins: [router],
        stubs: {
          'router-link': RouterLinkStub,
          ConfirmDialog: true,
          ContentBody: true,
          AskBubble: true,
        },
      },
    });
    await flushPromises();
    const contentRoot = w.find('div.mb-2.relative').element as HTMLElement;
    const dummy = document.createElement('span');
    dummy.setAttribute('data-ref-id', 'r2');
    contentRoot.appendChild(dummy);
    dummy.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    await flushPromises();
    expect(mockOpenForConversation).not.toHaveBeenCalled();
    expect(pushSpy).toHaveBeenCalledWith({ name: 'card', params: { cardId: 'c2-derived' } });
    w.unmount();
  });
});

describe('CardView — container mode', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    (getCard as any).mockResolvedValue({
      ...liveCard,
      content: '', // empty content signals potential container
    });
  });

  const containerMount = () =>
    mount(CardView, {
      props: { cardId: 'c1' },
      global: {
        stubs: {
          'router-link': RouterLinkStub,
          ConfirmDialog: true,
          ContentBody: true,
          AskBubble: true,
          ContainerBody: true,
        },
      },
    });

  it('mounts CardBody', async () => {
    const w = containerMount();
    await flushPromises();
    expect(w.findComponent({ name: 'CardBody' }).exists()).toBe(true);
  });

  it('toggles showTrashedSections when the header button is clicked', async () => {
    const w = containerMount();
    await flushPromises();
    const { useCardsStore } = await import('../stores/cards');
    const cs = useCardsStore();
    cs.byChildren['c1'] = [
      { ...liveCard, id: 'a', parent_card_id: 'c1', position: 0, content: 'a' },
    ];
    await flushPromises();
    const btn = w.find('[data-test="toggle-trashed-sections"]');
    expect(btn.exists()).toBe(true);
    await btn.trigger('click');
    const { useTilePrefsStore } = await import('../stores/tilePrefs');
    const prefs = useTilePrefsStore();
    expect(prefs.showTrashedSections).toBe(true);
  });
});

// The pill left the reading column. It now lives in the gutter, which renders
// for a leaf exactly as it does for a container — one section, one dot, one
// cell. Leaf and container finally grade the same way, which is what killed the
// right-[68px] offset: it only ever existed to dodge a chip the leaf never had.
describe('CardView — leaf renders the gutter', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    (getCard as any).mockResolvedValue(liveCard);
  });

  it('puts the pill in the gutter, outside the reading column', async () => {
    const w = mountView();
    await flushPromises();
    const gutter = w.find('[data-test="section-gutter"]');
    expect(gutter.exists()).toBe(true);
    expect(gutter.find('[data-test="grade-pill"]').exists()).toBe(true);
    // and NOT inside the section shell any more
    expect(w.find('section[data-card-id="c1"] [data-test="grade-pill"]').exists()).toBe(false);
  });

  it('draws one dot for the leaf itself', async () => {
    const w = mountView();
    await flushPromises();
    expect(w.find('[data-test="gutter-dot-c1"]').exists()).toBe(true);
  });

  it('has no ↗ open chip left in the reading column', async () => {
    const w = mountView();
    await flushPromises();
    expect(w.html()).not.toContain('↗ open');
  });
});

describe('CardView — container preamble', () => {
  const containerCard = {
    ...liveCard, id: 'c1', content: 'Date: 2026-07-11', title: 'Doc',
  };
  const section = {
    ...liveCard, id: 's1', parent_card_id: 'c1', content: 'section body', title: 'S1',
  };

  beforeEach(() => {
    setActivePinia(createPinia());
    (getCard as any).mockResolvedValue(containerCard);
    (listChildren as any).mockResolvedValue([section]);
  });

  it('renders the preamble block for a container that has content', async () => {
    const w = mountView();
    await flushPromises();
    // isContainerView is true even though content is non-empty
    expect(w.find('[data-test="toggle-trashed-sections"]').exists()).toBe(true);
    expect(w.find('[data-test="container-preamble"]').exists()).toBe(true);
  });

  it('does not render the preamble block for a leaf card', async () => {
    (getCard as any).mockResolvedValue(liveCard); // content 'body', no children
    (listChildren as any).mockResolvedValue([]);
    const w = mountView();
    await flushPromises();
    expect(w.find('[data-test="container-preamble"]').exists()).toBe(false);
  });
});
