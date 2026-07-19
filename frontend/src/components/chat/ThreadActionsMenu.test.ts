import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ThreadActionsMenu from './ThreadActionsMenu.vue';
import { useConversationsStore } from '../../stores/conversations';

describe('ThreadActionsMenu', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it('opens on the trigger and closes on Escape', async () => {
    const w = mount(ThreadActionsMenu, { props: { conversationId: 'c1' } });
    expect(w.find('[data-test="thread-menu"]').exists()).toBe(false);
    await w.find('[data-test="thread-menu-trigger"]').trigger('click');
    expect(w.find('[data-test="thread-menu"]').exists()).toBe(true);
    await w.find('[data-test="thread-menu-trigger"]').trigger('keydown', { key: 'Escape' });
    expect(w.find('[data-test="thread-menu"]').exists()).toBe(false);
  });

  it('deletes after confirm', async () => {
    const store = useConversationsStore();
    const spy = vi.spyOn(store, 'deleteOne').mockResolvedValue();
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    const w = mount(ThreadActionsMenu, { props: { conversationId: 'c1' } });
    await w.find('[data-test="thread-menu-trigger"]').trigger('click');
    await w.find('[data-test="thread-delete"]').trigger('click');
    expect(spy).toHaveBeenCalledWith('c1');
  });

  it('does not delete when confirm is dismissed', async () => {
    const store = useConversationsStore();
    const spy = vi.spyOn(store, 'deleteOne').mockResolvedValue();
    vi.spyOn(window, 'confirm').mockReturnValue(false);
    const w = mount(ThreadActionsMenu, { props: { conversationId: 'c1' } });
    await w.find('[data-test="thread-menu-trigger"]').trigger('click');
    await w.find('[data-test="thread-delete"]').trigger('click');
    expect(spy).not.toHaveBeenCalled();
  });

  it('renames with the prompted title', async () => {
    const store = useConversationsStore();
    const spy = vi.spyOn(store, 'rename').mockResolvedValue();
    vi.spyOn(window, 'prompt').mockReturnValue('  New name  ');
    const w = mount(ThreadActionsMenu, { props: { conversationId: 'c1' } });
    await w.find('[data-test="thread-menu-trigger"]').trigger('click');
    await w.find('[data-test="thread-rename"]').trigger('click');
    expect(spy).toHaveBeenCalledWith('c1', 'New name');
  });
});
