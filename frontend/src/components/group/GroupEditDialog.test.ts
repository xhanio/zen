import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import GroupEditDialog from './GroupEditDialog.vue';
import { useGroupsStore } from '../../stores/groups';
import type { Group } from '../../types/entity';

const group = (over: Partial<Group> = {}): Group => ({
  id: 'G1', name: 'Design',
  rule: 'Card must be in Chinese\n\nBy default, cards in this group should be HTML.',
  position: 0, level_catalog: [{ id: 'E1', weight: 0, name: '原则' }],
  created_at: '', updated_at: '', ...over,
} as Group);

// Stub ConfirmDialog with a button that re-emits confirm, so Delete is testable.
const confirmStub = {
  template: '<button data-test="confirm-delete" @click="$emit(\'confirm\')"></button>',
  props: ['open', 'title', 'description', 'confirmLabel', 'destructive'],
};

beforeEach(() => setActivePinia(createPinia()));

const mountDialog = (g = group()) =>
  mount(GroupEditDialog, { props: { group: g }, global: { stubs: { ConfirmDialog: confirmStub } } });

describe('GroupEditDialog', () => {
  it('loads name, parses the format out of the rule, shows the body', () => {
    const w = mountDialog();
    expect((w.find('[data-test="group-name"]').element as HTMLInputElement).value).toBe('Design');
    expect(w.find('[data-test="fmt-html"]').attributes('aria-pressed')).toBe('true');
    expect((w.find('[data-test="rule-input"]').element as HTMLTextAreaElement).value).toBe('Card must be in Chinese');
  });

  it('preview reflects the format choice', async () => {
    const w = mountDialog(group({ rule: 'hello' }));
    expect(w.find('[data-test="rule-preview"]').text()).toBe('hello');
    await w.find('[data-test="fmt-markdown"]').trigger('click');
    expect(w.find('[data-test="rule-preview"]').text()).toContain('By default, cards in this group should be Markdown.');
  });

  it('Save calls store.update with merged rule + trimmed name + catalog, then closes', async () => {
    const store = useGroupsStore();
    const spy = vi.spyOn(store, 'update').mockResolvedValue(group());
    const w = mountDialog(group({ rule: 'hi' }));
    await w.find('[data-test="fmt-html"]').trigger('click');
    await w.find('[data-test="group-save"]').trigger('click');
    await flushPromises();
    expect(spy).toHaveBeenCalledWith('G1', {
      name: 'Design',
      level_catalog: [{ id: 'E1', weight: 0, name: '原则' }],
      rule: 'hi\n\nBy default, cards in this group should be HTML.',
    });
    expect(w.emitted('close')).toBeTruthy();
  });

  it('empty name disables Save', async () => {
    const w = mountDialog();
    await w.find('[data-test="group-name"]').setValue('   ');
    expect((w.find('[data-test="group-save"]').element as HTMLButtonElement).disabled).toBe(true);
  });

  it('Delete confirm calls store.remove and closes', async () => {
    const store = useGroupsStore();
    const spy = vi.spyOn(store, 'remove').mockResolvedValue();
    const w = mountDialog();
    await w.find('[data-test="confirm-delete"]').trigger('click');
    expect(spy).toHaveBeenCalledWith('G1');
    await flushPromises();
    expect(w.emitted('close')).toBeTruthy();
  });
});
