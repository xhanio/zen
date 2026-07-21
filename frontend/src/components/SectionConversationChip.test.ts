import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ref } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import SectionConversationChip from './SectionConversationChip.vue';
import { useConversationsStore } from '../stores/conversations';
import type { Conversation } from '../types/entity';

const openForConversation = vi.fn().mockResolvedValue(undefined);
const openFor = vi.fn().mockResolvedValue(undefined);
const openRef = ref(false);
vi.mock('../composables/useChatSidebar', () => ({
  useChatSidebar: () => ({ open: openRef, openFor, openForConversation, close: vi.fn(), clearSelection: vi.fn() }),
}));

const conv = (over: Partial<Conversation>): Conversation => ({
  id: 'x', title: 't', anchor_kind: 'card', anchor_id: 'a', created_at: '', last_message_at: '', ...over,
});

function setup() {
  setActivePinia(createPinia());
  const store = useConversationsStore();
  vi.spyOn(store, 'loadForAnchor').mockResolvedValue();
  vi.spyOn(store, 'ensureConversation').mockResolvedValue();
  return store;
}

beforeEach(() => { openForConversation.mockClear(); openFor.mockClear(); openRef.value = false; });

describe('SectionConversationChip', () => {
  it('renders a lit chip and no ghost when the section has linked conversations', async () => {
    const store = setup();
    store.byAnchor = { 'card:a': [conv({ id: 'd1' })] };
    const w = mount(SectionConversationChip, { props: { anchorId: 'a', sourceConversationId: null } });
    await flushPromises();
    expect(w.find('[data-test="conv-chip"]').exists()).toBe(true);
    expect(w.find('[data-test="conv-chip-ghost"]').exists()).toBe(false);
  });

  it('renders only a ghost when a non-persistent section has none', async () => {
    setup();
    const w = mount(SectionConversationChip, { props: { anchorId: 'a', sourceConversationId: null } });
    await flushPromises();
    expect(w.find('[data-test="conv-chip"]').exists()).toBe(false);
    expect(w.find('[data-test="conv-chip-ghost"]').exists()).toBe(true);
  });

  it('a persistent chip is always shown even with none, and says Card', async () => {
    setup();
    const w = mount(SectionConversationChip, { props: { anchorId: 'a', sourceConversationId: null, persistent: true } });
    await flushPromises();
    const chip = w.find('[data-test="conv-chip"]');
    expect(chip.exists()).toBe(true);
    expect(chip.text()).toContain('Card');
  });

  it('opening the chip lists origin + discussions and rows open the sidebar', async () => {
    const store = setup();
    store.byID = { o1: conv({ id: 'o1', title: 'origin', anchor_kind: null, anchor_id: null }) };
    store.byAnchor = { 'card:a': [conv({ id: 'd1', title: 'why?' })] };
    const w = mount(SectionConversationChip, { props: { anchorId: 'a', sourceConversationId: 'o1' } });
    await flushPromises();
    await w.find('[data-test="conv-chip"]').trigger('click');
    expect(w.find('[data-test="conv-popover"]').exists()).toBe(true);
    await w.find('[data-test="conv-row-o1"]').trigger('click');
    expect(openForConversation).toHaveBeenCalledWith('o1');
  });

  it('new-thread starts a thread anchored to the section', async () => {
    const store = setup();
    store.byAnchor = { 'card:a': [conv({ id: 'd1' })] };
    const w = mount(SectionConversationChip, { props: { anchorId: 'a', sourceConversationId: null } });
    await flushPromises();
    await w.find('[data-test="conv-chip"]').trigger('click');
    await w.find('[data-test="conv-new-thread"]').trigger('click');
    expect(openFor).toHaveBeenCalledWith('card', 'a', null);
  });

  it('Escape closes the popover', async () => {
    const store = setup();
    store.byAnchor = { 'card:a': [conv({ id: 'd1' })] };
    const w = mount(SectionConversationChip, { props: { anchorId: 'a', sourceConversationId: null }, attachTo: document.body });
    await flushPromises();
    await w.find('[data-test="conv-chip"]').trigger('click');
    expect(w.find('[data-test="conv-popover"]').exists()).toBe(true);
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    await flushPromises();
    expect(w.find('[data-test="conv-popover"]').exists()).toBe(false);
    w.unmount();
  });

  it('a disabled (trashed) empty section shows no ghost', async () => {
    setup();
    const w = mount(SectionConversationChip, { props: { anchorId: 'a', sourceConversationId: null, disabled: true } });
    await flushPromises();
    expect(w.find('[data-test="conv-chip-ghost"]').exists()).toBe(false);
  });
});
